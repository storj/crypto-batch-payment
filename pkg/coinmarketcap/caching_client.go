package coinmarketcap

import (
	"context"
	"sync"
	"time"
)

type quoteCache struct {
	mu      sync.Mutex
	quote   *Quote
	updated time.Time
}

type CachingClient struct {
	client *Client
	expiry time.Duration

	mu    sync.Mutex
	cache map[Symbol]*quoteCache

	now func() time.Time
}

var _ Quoter = (*CachingClient)(nil)

func NewCachingClient(apiURL, apiKey string, expiry time.Duration) (*CachingClient, error) {
	client, err := NewClient(apiURL, apiKey)
	if err != nil {
		return nil, err
	}
	return &CachingClient{
		client: client,
		expiry: expiry,
		cache:  make(map[Symbol]*quoteCache),
		now:    time.Now,
	}, nil
}

func (cli *CachingClient) GetQuote(ctx context.Context, symbol Symbol) (*Quote, error) {
	// Take the client-wide lock and obtain the symbol cache
	cli.mu.Lock()
	cache, ok := cli.cache[symbol]
	if !ok {
		cache = new(quoteCache)
		cli.cache[symbol] = cache
	}
	cli.mu.Unlock()

	// Lock the symbol cache
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// If the cache is populated and has has not expired yet, return the cached
	// quote
	if !cache.updated.IsZero() &&
		cli.now().Before(cache.updated.Add(cli.expiry)) {
		return cache.quote, nil
	}

	// Get the latest quote
	quote, err := cli.client.GetQuote(ctx, symbol)
	if err != nil {
		return nil, err
	}
	cache.quote = quote
	cache.updated = cli.now()

	return quote, nil
}
