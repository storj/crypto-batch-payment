package storjtoken

import (
	"context"
	"math/big"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
)

func TestGetPrice(t *testing.T) {
	ctx := context.Background()

	var requested []coinmarketcap.Symbol
	quoter := coinmarketcap.QuoterFunc(func(ctx context.Context, symbol coinmarketcap.Symbol) (*coinmarketcap.Quote, error) {
		requested = append(requested, symbol)
		if symbol == "FAIL" {
			return nil, errs.New("no quote")
		}
		return &coinmarketcap.Quote{Price: decimal.RequireFromString("0.25")}, nil
	})

	price, err := GetPrice(ctx, quoter, USDC)
	require.NoError(t, err)
	require.Equal(t, "1", price.String())
	require.Empty(t, requested, "USDC price should not be quoted")

	price, err = GetPrice(ctx, quoter, "STORJ")
	require.NoError(t, err)
	require.Equal(t, "0.25", price.String())
	require.Equal(t, []coinmarketcap.Symbol{"STORJ"}, requested)

	_, err = GetPrice(ctx, quoter, "FAIL")
	require.Error(t, err)
}

func TestFromUSD(t *testing.T) {
	one := decimal.NewFromInt(1)

	require.Equal(t, big.NewInt(1_234_567), FromUSD(decimal.RequireFromString("1.2345678"), one, 6))
	require.Equal(t, big.NewInt(123_456_780), FromUSD(decimal.RequireFromString("1.2345678"), one, 8))
	require.Equal(t, big.NewInt(500_000_000), FromUSD(decimal.NewFromInt(1), decimal.RequireFromString("0.2"), 8))

	require.Panics(t, func() { FromUSD(one, one, 5) })
}

func TestPretty(t *testing.T) {
	require.Equal(t, "1500000 (1.5 USDC)", Pretty(big.NewInt(1_500_000), 6, "USDC"))
	require.Equal(t, "150000000 (1.5 STORJ)", Pretty(big.NewInt(150_000_000), 8, "STORJ"))
}
