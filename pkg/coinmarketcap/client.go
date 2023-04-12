// Package coinmarketcap provides client code for interacting with the
// CoinMarketCap API.
package coinmarketcap

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
)

const (
	apiKeyHeader    = "X-CMC_PRO_API_KEY"
	latestQuotePath = "/v1/cryptocurrency/quotes/latest"
	convert         = "USD"

	SandboxAPIURL    = "https://sandbox-api.coinmarketcap.com"
	SandboxAPIKey    = "b54bcf4d-1bca-4e8e-9a24-22ff2c3d462c"
	ProductionAPIURL = "https://pro-api.coinmarketcap.com"
)

type Symbol string

const (
	STORJ = "STORJ"
)

type Quoter interface {
	GetQuote(ctx context.Context, symbol Symbol) (*Quote, error)
}

type QuoterFunc func(ctx context.Context, symbol Symbol) (*Quote, error)

func (q QuoterFunc) GetQuote(ctx context.Context, symbol Symbol) (*Quote, error) {
	return q(ctx, symbol)
}

type Quote struct {
	Price       decimal.Decimal
	LastUpdated time.Time
}

type Client struct {
	apiKey  string
	baseURL *url.URL
}

func NewClient(apiURL, apiKey string) (*Client, error) {
	baseURL, err := parseAPIURL(apiURL)
	if err != nil {
		return nil, err
	}

	if apiKey == "" {
		return nil, errs.New("API Key is required")
	}

	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

func (cli *Client) GetQuote(ctx context.Context, symbol Symbol) (*Quote, error) {
	u := *cli.baseURL
	u.Path = latestQuotePath

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	req = req.WithContext(ctx)

	req.Header.Set("Accept", "application/json")
	req.Header.Add(apiKeyHeader, cli.apiKey)

	q := url.Values{}
	q.Add("symbol", string(symbol))
	q.Add("convert", convert)
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errs.New("unexpected status %d: %s", resp.StatusCode, tryRead(resp.Body))
	}

	type response struct {
		Status struct {
			ErrorCode    int     `json:"error_code"`
			ErrorMessage *string `json:"error_message"`
		}
		Data map[string]*struct {
			Quote map[string]*struct {
				Price       *decimal.Decimal `json:"price"`
				LastUpdated *string          `json:"last_updated"`
			} `json:"quote"`
		} `json:"data"`
	}

	r := new(response)
	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return nil, errs.New("invalid JSON response: %v", err)
	}

	if r.Status.ErrorCode > 0 {
		return nil, errs.New("error occurred: code=%d msg=%s", r.Status.ErrorCode, safeErrorMessage(r.Status.ErrorMessage))
	}

	data, ok := r.Data[string(symbol)]
	if !ok || data == nil {
		return nil, errs.New("no data returned for symbol %q", symbol)
	}

	quote, ok := data.Quote[convert]
	if !ok || quote == nil {
		return nil, errs.New("no %q quote returned for symbol %q", convert, symbol)
	}

	if quote.Price == nil {
		return nil, errs.New("%q quote missing price for symbol %q", convert, symbol)
	}

	var lastUpdated time.Time
	if quote.LastUpdated != nil {
		lastUpdated, err = time.Parse(time.RFC3339Nano, *quote.LastUpdated)
		if err != nil {
			return nil, errs.New("invalid last_updated value %q: %v", *quote.LastUpdated, err)
		}
	}

	return &Quote{
		Price:       *quote.Price,
		LastUpdated: lastUpdated,
	}, nil
}

func tryRead(r io.Reader) string {
	b := make([]byte, 256)
	n, _ := r.Read(b)
	return string(b[:n])
}

func parseAPIURL(s string) (*url.URL, error) {
	if s == "" {
		return nil, errs.New("API URL is required")
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, errs.New("API URL is malformed: %v", err)
	}
	switch {
	case u.Scheme != "http" && u.Scheme != "https":
		return nil, errs.New("API URL scheme must be http or https")
	case u.User != nil:
		return nil, errs.New("API URL must not have user info")
	case u.Host == "":
		return nil, errs.New("API URL must specify the host")
	case u.Path != "" && u.Path != "/":
		return nil, errs.New("API URL must not have a path")
	case u.RawQuery != "":
		return nil, errs.New("API URL must not have query values")
	case u.Fragment != "":
		return nil, errs.New("API URL must not have a fragment")
	}
	return u, nil
}

func safeErrorMessage(s *string) string {
	if s == nil {
		return "<unknown>"
	}
	return strconv.Quote(*s)
}
