package main

import (
	"fmt"
	"math/big"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/params"

	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/payouts"
	"storj.io/crypto-batch-payment/pkg/pipeline"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
)

type runConfig struct {
	*rootConfig
	Name                    string
	SpenderKeyPath          string
	Owner                   string
	ContractAddress         string
	EstimateCacheExpiry     time.Duration
	MaxGas                  string
	MaxFee                  string
	GasTipCap               string
	CoinMarketCapAPIURL     string
	CoinMarketCapAPIKeyPath string
	QuoteCacheExpiry        time.Duration
	PipelineLimit           int
	TxDelay                 time.Duration
	SkipConfirmation        bool
	Drain                   bool
	NodeType                string
	PayerType               string
}

func newRunCommand(rootConfig *rootConfig) *cobra.Command {
	config := &runConfig{
		rootConfig: rootConfig,
	}
	cmd := &cobra.Command{
		Use:   "run NAME SPENDERKEYPATH",
		Short: "Runs payout",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Name = args[0]
			config.SpenderKeyPath = args[1]
			if len(args) > 2 {
				config.Owner = args[2]
			}
			return checkCmd(doRun(config))
		},
	}
	cmd.Flags().StringVarP(
		&config.Owner,
		"owner", "",
		"",
		"Owner of the ERC20 token (spender if unset)")
	cmd.Flags().StringVarP(
		&config.ContractAddress,
		"contract", "",
		storjtoken.DefaultContractAddress.String(),
		"Address of the STORJ contract on the network")
	cmd.Flags().StringVarP(
		&config.CoinMarketCapAPIURL,
		"coinmarkcap-api-url", "",
		coinmarketcap.ProductionAPIURL,
		"CoinMarketCap API URL")
	cmd.Flags().StringVarP(
		&config.CoinMarketCapAPIKeyPath,
		"coinmarkcap-api-key-path", "",
		filepath.Join(homeDir, ".coinmarketcapkey"),
		"Path on disk to the CoinMarketCap API key")
	cmd.Flags().DurationVarP(
		&config.EstimateCacheExpiry,
		"estimate-cache-expiry", "",
		time.Second*5,
		"How often gas estimates should be recalculated")
	cmd.Flags().StringVarP(
		&config.MaxGas,
		"max-gas", "",
		"70"+"000"+"000"+"000",
		"Max gas price we're willing to consider in Wei (tip + base fee). Default: 70 GWei. Only applies to Eth type payment.")
	cmd.Flags().StringVarP(
		&config.MaxFee,
		"max-fee", "",
		"",
		"Max fee we're willing to consider. Only applies to zksync or zkwithdraw type payment.")
	cmd.Flags().StringVarP(
		&config.GasTipCap,
		"gas-tip-cap", "",
		"1000000000",
		"Gas tip for miners on top of EIP-1559 standard gas price (in Wei). Default: use gas oracle.")
	cmd.Flags().DurationVarP(
		&config.QuoteCacheExpiry,
		"quote-cache-expiry", "",
		time.Second*5,
		"How often price quotes for currency should refreshed")
	cmd.Flags().IntVarP(
		&config.PipelineLimit,
		"pipeline-limit", "",
		pipeline.DefaultLimit,
		"How many transactions to have in the pipeline at once")
	cmd.Flags().DurationVarP(
		&config.TxDelay,
		"tx-delay", "",
		pipeline.DefaultTxDelay,
		"How long to wait between sending individual transactions")
	cmd.Flags().BoolVarP(
		&config.SkipConfirmation,
		"skip-confirmation", "",
		false,
		"Skip confirmation")
	cmd.Flags().BoolVarP(
		&config.Drain,
		"drain", "",
		false,
		"Drain existing transactions only")
	cmd.Flags().StringVarP(
		&config.NodeType,
		"node-type", "",
		string(pipeline.Geth),
		"Node type (one of [geth, parity])")
	cmd.Flags().StringVarP(
		&config.PayerType,
		"type", "",
		string(payouts.Eth),
		"Type of the payment (eth,zksync,zkwithdraw,sim,polygon)")
	return cmd
}

func doRun(config *runConfig) error {
	spenderKey, spenderAddress, err := loadETHKey(config.SpenderKeyPath, "spender")
	if err != nil {
		return err
	}

	owner := spenderAddress
	if config.Owner != "" {
		owner, err = convertAddress(config.Owner, "owner")
		if err != nil {
			return err
		}
	}

	contractAddress, err := convertAddress(config.ContractAddress, "contract")
	if err != nil {
		return err
	}

	nodeType, err := pipeline.NodeTypeFromString(config.NodeType)
	if err != nil {
		return err
	}

	payerType, err := payouts.PayerTypeFromString(config.PayerType)
	if err != nil {
		return err
	}

	coinMarketCapAPIKey, err := loadFirstLine(config.CoinMarketCapAPIKeyPath)
	if err != nil {
		return errs.New("failed to load CoinMarketCap key: %v\n", err)
	}

	chainID, err := convertInt(config.ChainID, 0, "chain-id")
	if err != nil {
		return err
	}

	quoter, err := coinmarketcap.NewCachingClient(config.CoinMarketCapAPIURL, coinMarketCapAPIKey, config.QuoteCacheExpiry)
	if err != nil {
		return errs.New("failed instantiate coinmarketcap client: %v\n", err)
	}

	promptConfirm := promptConfirm
	if config.SkipConfirmation {
		promptConfirm = func(label string) error {
			fmt.Printf("Skipping confirmation to %s!\n", label)
			return nil
		}
	}

	var maxGas big.Int
	_, ok := maxGas.SetString(config.MaxGas, 10)
	if !ok {
		return errs.New("invalid max gas setting")
	}

	var maxFee *big.Int
	if config.MaxFee != "" {
		var tmp big.Int
		if _, ok := tmp.SetString(config.MaxFee, 10); !ok {
			return errs.New("invalid max fee setting")
		}
		maxFee = &tmp
	}

	var gasTipCap *big.Int
	if config.GasTipCap != "" {
		gasTipCap = new(big.Int)
		_, ok = gasTipCap.SetString(config.GasTipCap, 10)
		if !ok {
			return errs.New("invalid gas tip cap setting")
		}
		if gasTipCap.Cmp(big.NewInt(30*params.GWei)) > 0 {
			return errs.New("Gas tip cap is too high. Please use value less than 30 gwei")
		}

		if gasTipCap.Cmp(big.NewInt(int64(100))) < 0 {
			return errs.New("Gas tip cap is negligible. Please check if you really used wei unit (or set 0)")
		}
	}

	fmt.Printf("Running %q payout...\n", config.Name)
	err = payouts.Run(config.Ctx, payouts.Config{
		DataDir:             config.DataDir,
		Name:                config.Name,
		Spender:             spenderKey,
		ChainID:             *chainID,
		Owner:               owner,
		EstimateCacheExpiry: config.EstimateCacheExpiry,
		MaxGas:              maxGas,
		MaxFee:              maxFee,
		GasTipCap:           gasTipCap,
		Quoter:              quoter,
		ContractAddress:     contractAddress,
		PipelineLimit:       config.PipelineLimit,
		TxDelay:             config.TxDelay,
		Drain:               config.Drain,
		NodeType:            nodeType,
		PromptConfirm:       promptConfirm,
		PayerType:           payerType,
		NodeAddress:         config.NodeAddress,
	})
	if err != nil {
		return err
	}
	fmt.Println("Payouts complete.")
	return nil
}
