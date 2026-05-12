package config_test

import (
	"os/user"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"storj.io/crypto-batch-payment/pkg/config"
	"storj.io/crypto-batch-payment/pkg/eth"
)

func TestLoad_Defaults(t *testing.T) {
	currentUser, err := user.Current()
	require.NoError(t, err)

	homePath := func(suffix string) config.Path {
		return config.Path(filepath.Join(currentUser.HomeDir, suffix))
	}

	cfg, err := config.Load("./testdata/defaults.toml")
	t.Logf("unknown fields:\n%s", config.DumpUnknownFields(err))
	require.NoError(t, err)

	assert.Equal(t, config.Config{
		Pipeline: config.Pipeline{
			DepthLimit:       16,
			TxDelay:          0,
			ThresholdDivisor: 4,
		},
		CoinMarketCap: config.CoinMarketCap{
			APIKeyPath:  homePath(".coinmarketcapkey"),
			APIURL:      "https://pro-api.coinmarketcap.com",
			CacheExpiry: 5000000000,
		},
		Eth: &config.Eth{
			NodeAddress:          "https://someaddress.test",
			SpenderKeyPath:       homePath("some.key"),
			ERC20ContractAddress: common.HexToAddress("0x1111111111111111111111111111111111111111"),
			ChainID:              0,
			Owner:                nil,
			GasFeeCapOverride:    ptrOf(eth.RequireParseUnit("70gwei")),
			ExtraGasTip:          nil,
		},
		ZkSyncEra: &config.ZkSyncEra{
			NodeAddress:          "https://mainnet.era.zksync.io",
			SpenderKeyPath:       homePath("some.key"),
			ERC20ContractAddress: common.HexToAddress("0x2222222222222222222222222222222222222222"),
			ChainID:              0,
			PaymasterAddress:     nil,
			PaymasterPayload:     nil,
		},
	}, cfg)
}

func TestLoad_Overrides(t *testing.T) {
	cfg, err := config.Load("./testdata/override.toml")
	require.NoError(t, err)

	assert.Equal(t, config.Config{
		Pipeline: config.Pipeline{
			DepthLimit:          24,
			TxDelay:             config.Duration(time.Minute),
			ThresholdDivisor:    5,
			MaxFeeTolerationUSD: decimal.RequireFromString("1.23"),
		},
		CoinMarketCap: config.CoinMarketCap{
			APIURL:      "https://override.test",
			APIKeyPath:  "override",
			CacheExpiry: 10000000000,
		},
		Eth: &config.Eth{
			NodeAddress:          "https://override.test",
			SpenderKeyPath:       "override",
			ERC20ContractAddress: common.HexToAddress("0xe66652d41EE7e81d3fcAe1dF7F9B9f9411ac835e"),
			ChainID:              12345,
			Owner:                ptrOf(common.HexToAddress("0xe66652d41EE7e81d3fcAe1dF7F9B9f9411ac835e")),
			GasFeeCapOverride:    ptrOf(eth.RequireParseUnit("99gwei")),
			ExtraGasTip:          ptrOf(eth.RequireParseUnit("1gwei")),
		},
		ZkSyncEra: &config.ZkSyncEra{
			NodeAddress:          "https://override.test",
			SpenderKeyPath:       "override",
			ERC20ContractAddress: common.HexToAddress("0xe66652d41EE7e81d3fcAe1dF7F9B9f9411ac835e"),
			ChainID:              12345,
			PaymasterAddress:     ptrOf(common.HexToAddress("0xe66652d41EE7e81d3fcAe1dF7F9B9f9411ac835e")),
			PaymasterPayload:     []byte("\x01\x23"),
		},
	}, cfg)
}

func ptrOf[T any](t T) *T {
	return &t
}
