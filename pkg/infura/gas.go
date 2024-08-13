package infura

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
)

type SuggestedGasFees struct {
	Low                        RecommendedGasValues `json:"low"`
	Medium                     RecommendedGasValues `json:"medium"`
	High                       RecommendedGasValues `json:"high"`
	EstimatedBaseFee           decimal.Decimal      `json:"estimatedBaseFee"`
	NetworkCongestion          float64              `json:"networkCongestion"`
	LatestPriorityFeeRange     []decimal.Decimal    `json:"latestPriorityFeeRange"`
	HistoricalPriorityFeeRange []decimal.Decimal    `json:"historicalPriorityFeeRange"`
	HistoricalBaseFeeRange     []decimal.Decimal    `json:"historicalBaseFeeRange"`
	PriorityFeeTrend           Trend                `json:"priorityFeeTrend"`
	BaseFeeTrend               Trend                `json:"baseFeeTrend"`
}

type RecommendedGasValues struct {
	SuggestedMaxPriorityFeePerGas decimal.Decimal `json:"suggestedMaxPriorityFeePerGas"`
	SuggestedMaxFeePerGas         decimal.Decimal `json:"suggestedMaxFeePerGas"`
	MinWaitTimeEstimateMillis     int64           `json:"minWaitTimeEstimate"`
	MaxWaitTimeEstimateMillis     int64           `json:"maxWaitTimeEstimate"`
}

type Trend string

const (
	TrendUp   = "up"
	TrendDown = "down"
)

func GetSuggestedGasFees(ctx context.Context, apiKey string, chainID int) (*SuggestedGasFees, error) {
	return GetSuggestedGasFeesFromURL(ctx, fmt.Sprintf("https://gas.api.infura.io/v3/%s/networks/%d/suggestedGasFees", apiKey, chainID))
}

func GetSuggestedGasFeesFromURL(ctx context.Context, reqURL string) (*SuggestedGasFees, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errs.New("expected status code 200 but got %d: %s", resp.StatusCode, tryRead(resp.Body))
	}

	out := new(SuggestedGasFees)
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return nil, errs.Wrap(err)
	}

	return out, nil
}

func tryRead(r io.Reader) string {
	b := make([]byte, 256)
	n, _ := r.Read(b)
	return string(b[:n])
}
