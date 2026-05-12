package config

import (
	"time"

	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
)

type CoinMarketCap struct {
	APIKeyPath  Path     `toml:"api_key_path"`
	APIURL      string   `toml:"api_url"`
	CacheExpiry Duration `toml:"cache_expiry"`
}

func (c CoinMarketCap) NewQuoter() (coinmarketcap.Quoter, error) {
	apiKey, err := loadFirstLine(string(c.APIKeyPath))
	if err != nil {
		return nil, errs.New("failed to load CoinMarketCap key: %v\n", err)
	}

	quoter, err := coinmarketcap.NewCachingClient(c.APIURL, apiKey, time.Duration(c.CacheExpiry))
	if err != nil {
		return nil, errs.New("failed instantiate coinmarketcap client: %v\n", err)
	}

	return quoter, nil
}
