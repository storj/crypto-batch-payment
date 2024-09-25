package infura

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestReal(t *testing.T) {
	apiKey := os.Getenv("CRYBAPY_INFURA_TEST_APIKEY")
	if apiKey == "" {
		t.Skip("Set CRYBAPY_INFURA_TEST_APIKEY environment variable to run this test")
	}

	out, err := GetSuggestedGasFees(context.Background(), apiKey, 1)
	require.NoError(t, err)
	b, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))

	out, err = GetSuggestedGasFees(context.Background(), apiKey, 324)
	require.NoError(t, err)
	b, _ = json.MarshalIndent(out, "", "  ")
	fmt.Println(string(b))
}

func TestGetSuggestedGasFeesFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ContentType", "application/json")
		_, _ = w.Write([]byte(`
			{
			  "low": {
				"suggestedMaxPriorityFeePerGas": "0.05",
				"suggestedMaxFeePerGas": "16.334026964",
				"minWaitTimeEstimate": 15000,
				"maxWaitTimeEstimate": 30000
			  },
			  "medium": {
				"suggestedMaxPriorityFeePerGas": "0.1",
				"suggestedMaxFeePerGas": "22.083436402",
				"minWaitTimeEstimate": 15000,
				"maxWaitTimeEstimate": 45000
			  },
			  "high": {
				"suggestedMaxPriorityFeePerGas": "0.3",
				"suggestedMaxFeePerGas": "27.982845839",
				"minWaitTimeEstimate": 15000,
				"maxWaitTimeEstimate": 60000
			  },
			  "estimatedBaseFee": "16.284026964",
			  "networkCongestion": 0.5125,
			  "latestPriorityFeeRange": [ "0", "3" ],
			  "historicalPriorityFeeRange": [ "0.000000001", "89" ],
			  "historicalBaseFeeRange": [ "13.773088584", "29.912845463" ],
			  "priorityFeeTrend": "down",
			  "baseFeeTrend": "up"
			}
	`))
	}))

	makeDecimals := func(ss ...string) (dd []decimal.Decimal) {
		for _, s := range ss {
			dd = append(dd, decimal.RequireFromString(s))
		}
		return dd
	}

	want := SuggestedGasFees{
		Low: RecommendedGasValues{
			SuggestedMaxPriorityFeePerGas: decimal.RequireFromString("0.05"),
			SuggestedMaxFeePerGas:         decimal.RequireFromString("16.334026964"),
			MinWaitTimeEstimateMillis:     15000,
			MaxWaitTimeEstimateMillis:     30000,
		},
		Medium: RecommendedGasValues{
			SuggestedMaxPriorityFeePerGas: decimal.RequireFromString("0.1"),
			SuggestedMaxFeePerGas:         decimal.RequireFromString("22.083436402"),
			MinWaitTimeEstimateMillis:     15000,
			MaxWaitTimeEstimateMillis:     45000,
		},
		High: RecommendedGasValues{
			SuggestedMaxPriorityFeePerGas: decimal.RequireFromString("0.3"),
			SuggestedMaxFeePerGas:         decimal.RequireFromString("27.982845839"),
			MinWaitTimeEstimateMillis:     15000,
			MaxWaitTimeEstimateMillis:     60000,
		},
		EstimatedBaseFee:           decimal.RequireFromString("16.284026964"),
		NetworkCongestion:          0.5125,
		LatestPriorityFeeRange:     makeDecimals("0", "3"),
		HistoricalPriorityFeeRange: makeDecimals("0.000000001", "89"),
		HistoricalBaseFeeRange:     makeDecimals("13.773088584", "29.912845463"),
		PriorityFeeTrend:           TrendDown,
		BaseFeeTrend:               TrendUp,
	}

	got, err := GetSuggestedGasFeesFromURL(context.Background(), server.URL)
	require.NoError(t, err)
	require.Equal(t, &want, got)
}
