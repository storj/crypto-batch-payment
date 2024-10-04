package config

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/shopspring/decimal"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/eth"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipeline"
)

type MissingFieldsError = toml.StrictMissingError

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
	// DepthLimit is how many transactions to batch up at a time.
	DepthLimit int `toml:"depth_limit"`

	// TxDelay is how long to sleep in between issuing transactions.
	TxDelay Duration `toml:"tx_delay"`

	// ThresholdDivisor divides the payout amount to calculate the payout
	// skip threshold per payout. If the estimated maximum payout transaction
	// fees for the payout exceed this threshold then the payout is skipped.
	ThresholdDivisor int `toml:"threshold_divisor"`

	// MaxFeeTolerationUSD is the maximum per-transfer fee to tolerate. Payouts
	// that have an estimated fee higher than this value will be skipped. If
	// <= 0, then no maximum fee is enforced.
	MaxFeeTolerationUSD decimal.Decimal `toml:"max_fee_toleration_usd"`
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
		defaultPipelineDepthLimit = pipeline.DefaultLimit
		defaultPipelineTxDelay    = Duration(pipeline.DefaultTxDelay)
		defaultThresholdDivisor   = 4

		defaultCoinMarketCapKeyPath     = "~/.coinmarketcapkey"
		defaultCoinMarketCapAPIURL      = coinmarketcap.ProductionAPIURL
		defaultCoinMarketCapCacheExpiry = Duration(time.Second * 5)
	)
	var (
		defaultGasFeeCapOverride   = eth.RequireParseUnit("70gwei")
		defaultMaxFeeTolerationUSD = decimal.Decimal{}
	)

	config := Config{
		Pipeline: Pipeline{
			DepthLimit:          defaultPipelineDepthLimit,
			TxDelay:             defaultPipelineTxDelay,
			ThresholdDivisor:    defaultThresholdDivisor,
			MaxFeeTolerationUSD: defaultMaxFeeTolerationUSD,
		},
		CoinMarketCap: CoinMarketCap{
			APIKeyPath:  ToPath(defaultCoinMarketCapKeyPath),
			APIURL:      defaultCoinMarketCapAPIURL,
			CacheExpiry: defaultCoinMarketCapCacheExpiry,
		},
	}

	d := toml.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set ETH defaults.
	if config.Eth != nil && config.Eth.GasFeeCapOverride == nil {
		config.Eth.GasFeeCapOverride = &defaultGasFeeCapOverride
	}

	return config, nil
}

func DumpUnknownFields(err error) string {
	var sme *toml.StrictMissingError
	if errors.As(err, &sme) {
		return sme.String()
	}
	return ""
}
