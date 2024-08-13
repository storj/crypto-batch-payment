package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipeline"
)

type Config struct {
	Pipeline      Pipeline      `toml:"pipeline"`
	CoinMarketCap CoinMarketCap `toml:"coinmarketcap"`
	Eth           *Eth          `toml:"eth"`
	ZkSyncEra     *ZkSyncEra    `toml:"zksync-era"`
}

func (c *Config) NewPayers(ctx context.Context) (_ Payers, err error) {
	var payers Payers
	defer func() {
		if err != nil {
			payers.Close()
		}
	}()

	if c.Eth != nil {
		p, err := c.Eth.NewPayer(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to init eth payer: %w", err)
		}
		payers.Add(payer.Eth, p)
	}

	if c.ZkSyncEra != nil {
		p, err := c.ZkSyncEra.NewPayer(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to init zksync-era payer: %w", err)
		}
		payers.Add(payer.ZkSyncEra, p)
	}

	return payers, nil
}

func (c *Config) NewAuditors(ctx context.Context) (_ Auditors, err error) {
	var auditors Auditors
	defer func() {
		if err != nil {
			auditors.Close()
		}
	}()

	if c.Eth != nil {
		p, err := c.Eth.NewAuditor(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to init eth auditor: %w", err)
		}
		auditors.Add(payer.Eth, p)
	}

	if c.ZkSyncEra != nil {
		p, err := c.ZkSyncEra.NewAuditor(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to init zksync-era auditor: %w", err)
		}
		auditors.Add(payer.ZkSyncEra, p)
	}

	return auditors, nil
}

type Pipeline struct {
	DepthLimit int      `toml:"depth_limit"`
	TxDelay    Duration `toml:"tx_delay"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to read config: %w", err)
	}
	return Parse(data)
}

func Parse(data []byte) (Config, error) {
	const (
		defaultPipelineDepthLimit       = pipeline.DefaultLimit
		defaultPipelineTxDelay          = Duration(pipeline.DefaultTxDelay)
		defaultCoinMarketCapAPIURL      = coinmarketcap.ProductionAPIURL
		defaultCoinMarketCapKeyPath     = "~/.coinmarketcap"
		defaultCoinMarketCapCacheExpiry = time.Second * 5
	)

	config := Config{
		Pipeline: Pipeline{
			DepthLimit: defaultPipelineDepthLimit,
			TxDelay:    defaultPipelineTxDelay,
		},
		CoinMarketCap: CoinMarketCap{
			APIURL:      defaultCoinMarketCapAPIURL,
			APIKeyPath:  ToPath(defaultCoinMarketCapKeyPath),
			CacheExpiry: Duration(defaultCoinMarketCapCacheExpiry),
		},
	}

	d := toml.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
