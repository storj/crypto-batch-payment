package coinmarketcap

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testAPIKey = "00000000-1111-2222-3333-444444444444"
)

func TestClientNewClient(t *testing.T) {
	testCases := []struct {
		name   string
		url    string
		apiKey string
		err    string
	}{
		{
			name:   "success with prod API URL",
			url:    ProductionAPIURL,
			apiKey: testAPIKey,
		},
		{
			name:   "success with sandbox API URL",
			url:    SandboxAPIURL,
			apiKey: testAPIKey,
		},
		{
			name:   "success with HTTP API URL with empty path",
			url:    "http://host:80/",
			apiKey: testAPIKey,
		},
		{
			name:   "no API URL",
			apiKey: testAPIKey,
			err:    "API URL is required",
		},
		{
			name:   "API URL with invalid scheme",
			url:    "gopher://",
			apiKey: testAPIKey,
			err:    "API URL scheme must be http or https",
		},
		{
			name:   "API URL missing host",
			url:    "https://",
			apiKey: testAPIKey,
			err:    "API URL must specify the host",
		},
		{
			name:   "API URL with user info",
			url:    "https://foo:bar@host/",
			apiKey: testAPIKey,
			err:    "API URL must not have user info",
		},
		{
			name:   "API URL with non-empty path",
			url:    "https://host/path",
			apiKey: testAPIKey,
			err:    "API URL must not have a path",
		},
		{
			name:   "API URL with query values",
			url:    "https://host/?foo=bar",
			apiKey: testAPIKey,
			err:    "API URL must not have query values",
		},
		{
			name:   "API URL with fragment",
			url:    "https://host/#foo",
			apiKey: testAPIKey,
			err:    "API URL must not have a fragment",
		},
		{
			name: "no API Key",
			url:  SandboxAPIURL,
			err:  "API Key is required",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase // silence golint
		t.Run(testCase.name, func(t *testing.T) {
			client, err := NewClient(testCase.url, testCase.apiKey)
			if testCase.err != "" {
				require.EqualError(t, err, testCase.err)
				require.Nil(t, client)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, client)
		})
	}

}

func TestClientGetQuote(t *testing.T) {
	handler := newTestHandler()

	server := httptest.NewServer(handler)
	defer server.Close()

	client, err := NewClient(server.URL, testAPIKey)
	require.NoError(t, err)

	type testQuote struct {
		price       string
		lastUpdated string
	}

	testCases := []struct {
		name   string
		status int
		body   string
		quote  *testQuote
		err    string
	}{
		{
			name:   "invalid status code",
			status: http.StatusInternalServerError,
			body:   `we done broke`,
			err:    "unexpected status 500: we done broke",
		},
		{
			name:   "bad status without message",
			status: http.StatusOK,
			body: `{
				"status": {
					"error_code": 1,
					"error_message": null
				}
			}`,
			err: "error occurred: code=1 msg=<unknown>",
		},
		{
			name:   "bad JSON",
			status: http.StatusOK,
			body:   `{`,
			err:    "invalid JSON response: unexpected EOF",
		},
		{
			name:   "bad status with message",
			status: http.StatusOK,
			body: `{
				"status": {
					"error_code": 1,
					"error_message": "we done broke"
				}
			}`,
			err: `error occurred: code=1 msg="we done broke"`,
		},
		{
			name:   "no STORJ data",
			status: http.StatusOK,
			body: `{
				"data": {}
			}`,
			err: `no data returned for symbol "STORJ"`,
		},
		{
			name:   "null STORJ data",
			status: http.StatusOK,
			body: `{
				"data": {
					"STORJ": null
				}
			}`,
			err: `no data returned for symbol "STORJ"`,
		},
		{
			name:   "no USD quote",
			status: http.StatusOK,
			body: `{
				"data": {
					"STORJ": {
						"quote": {}
					}
				}
			}`,
			err: `no "USD" quote returned for symbol "STORJ"`,
		},
		{
			name:   "null USD quote",
			status: http.StatusOK,
			body: `{
				"data": {
					"STORJ": {
						"quote": {
							"USD": null
						}
					}
				}
			}`,
			err: `no "USD" quote returned for symbol "STORJ"`,
		},
		{
			name:   "USD quote missing price",
			status: http.StatusOK,
			body: `{
				"data": {
					"STORJ": {
						"quote": {
							"USD": {}
						}
					}
				}
			}`,
			err: `"USD" quote missing price for symbol "STORJ"`,
		},
		{
			name:   "invalid last updated time",
			status: http.StatusOK,
			body: `{
				"data": {
					"STORJ": {
						"quote": {
							"USD": {
								"price": 0.1234,
								"last_updated": "BLAH"
							}
						}
					}
				}
			}`,
			err: `invalid last_updated value "BLAH": parsing time "BLAH" as "2006-01-02T15:04:05.999999999Z07:00": cannot parse "BLAH" as "2006"`,
		},
		{
			name:   "success with last updated time",
			status: http.StatusOK,
			body: `{
				"status": {
					"error_code": 0,
					"error_message": null
				},
				"data": {
					"STORJ": {
						"quote": {
							"USD": {
								"price": 0.162645840588,
								"last_updated": "2019-07-18T14:34:05.000Z"
							}
						}
					}
				}
			}`,
			quote: &testQuote{
				price:       "0.162645840588",
				lastUpdated: "2019-07-18T14:34:05Z",
			},
		},
		{
			name:   "success without last updated time",
			status: http.StatusOK,
			body: `{
				"status": {
					"error_code": 0,
					"error_message": null
				},
				"data": {
					"STORJ": {
						"quote": {
							"USD": {
								"price": 0.162645840588
							}
						}
					}
				}
			}`,
			quote: &testQuote{
				price:       "0.162645840588",
				lastUpdated: "0001-01-01T00:00:00Z",
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase // silence golint
		t.Run(testCase.name, func(t *testing.T) {
			handler.SetResponse(testCase.status, testCase.body)

			quote, err := client.GetQuote(context.Background(), STORJ)
			if testCase.err != "" {
				require.EqualError(t, err, testCase.err)
				require.Nil(t, quote)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.quote, &testQuote{
				price:       quote.Price.String(),
				lastUpdated: quote.LastUpdated.Format(time.RFC3339Nano),
			})
		})
	}
}

type testHandler struct {
	status int
	body   string
}

func newTestHandler() *testHandler {
	return &testHandler{}
}

func (handler *testHandler) SetResponse(status int, body string) {
	handler.status = status
	handler.body = body
}

func (handler *testHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, fmt.Sprintf(`expected method "GET"; got %q`, r.Method), http.StatusMethodNotAllowed)
		return
	}
	if convert := r.URL.Query().Get("convert"); convert != "USD" {
		http.Error(w, fmt.Sprintf(`expected convert "USD"; got %q`, convert), http.StatusBadRequest)
		return
	}
	if symbol := r.URL.Query().Get("symbol"); symbol != "STORJ" {
		http.Error(w, fmt.Sprintf(`expected symbol "STORJ"; got %q`, symbol), http.StatusBadRequest)
		return
	}
	if accept := r.Header.Get("accept"); accept != "application/json" {
		http.Error(w, fmt.Sprintf(`expected accept "application/json"; got %q`, accept), http.StatusBadRequest)
		return
	}
	if apiKey := r.Header.Get(apiKeyHeader); apiKey != testAPIKey {
		http.Error(w, fmt.Sprintf(`expected API key %q; got %q`, testAPIKey, apiKey), http.StatusBadRequest)
		return
	}
	w.WriteHeader(handler.status)
	_, _ = w.Write([]byte(handler.body))
}
