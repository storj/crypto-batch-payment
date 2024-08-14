package infura

import (
	"context"
	"sync"
	"time"
)

func NewCache(client Client, expiry time.Duration) Client {
	return newCache(client, expiry, time.Now)
}

func newCache(client Client, expiry time.Duration, now func() time.Time) Client {
	return &cache{
		client:           client,
		expiry:           expiry,
		suggestedGasFees: make(map[int]suggestedGasFeesEntry),
		now:              now,
	}
}

type cache struct {
	client Client
	expiry time.Duration

	mu               sync.Mutex
	suggestedGasFees map[int]suggestedGasFeesEntry

	now func() time.Time
}

func (c *cache) GetSuggestedGasFees(ctx context.Context, chainID int) (*SuggestedGasFees, error) {
	c.mu.Lock()
	entry, ok := c.suggestedGasFees[chainID]
	c.mu.Unlock()

	now := c.now()
	if ok && now.Sub(entry.ts) < c.expiry {
		return entry.data, nil
	}

	data, err := c.client.GetSuggestedGasFees(ctx, chainID)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.suggestedGasFees[chainID] = suggestedGasFeesEntry{
		data: data,
		ts:   now,
	}
	return data, nil
}

type suggestedGasFeesEntry = cacheEntry[*SuggestedGasFees]

type cacheEntry[T any] struct {
	data T
	ts   time.Time
}
