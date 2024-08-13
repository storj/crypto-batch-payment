package infura

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache(t *testing.T) {
	ctx := context.Background()
	now := time.Now().Truncate(time.Minute)

	cache := newCache(&testClient{}, time.Minute, func() time.Time {
		return now
	})

	a, err := cache.GetSuggestedGasFees(ctx, 1)
	require.NoError(t, err)

	b, err := cache.GetSuggestedGasFees(ctx, 1)
	require.NoError(t, err)

	now.Add(time.Minute)

	c, err := cache.GetSuggestedGasFees(ctx, 1)
	require.NoError(t, err)

	d, err := cache.GetSuggestedGasFees(ctx, 1)
	require.NoError(t, err)

	// A and B should be the same since B is a cached response of A
	assert.Equal(t, a, b)

	// C and D should be the same since D is a cached response of C
	assert.Equal(t, c, d)

	// A and C (and therefore B and D) should be different.
	assert.NotEqual(t, a, c)
}

type testClient struct {
	calls int64
}

func (c *testClient) GetSuggestedGasFees(ctx context.Context, chainID int) (*SuggestedGasFees, error) {
	c.calls++
	return &SuggestedGasFees{EstimatedBaseFee: decimal.NewFromInt(c.calls)}, nil
}
