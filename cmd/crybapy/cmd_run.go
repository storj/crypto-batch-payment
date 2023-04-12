package main

import (
	"context"
	"fmt"
	"path/filepath"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"time"

	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/payouts"
	"storj.io/crypto-batch-payment/pkg/pipeline"
)

type runConfig struct {
	*rootConfig
	PayerConfig
	Name                    string
	SpenderKeyPath          string
	EstimateCacheExpiry     time.Duration
	CoinMarketCapAPIURL     string
	CoinMarketCapAPIKeyPath string
	QuoteCacheExpiry        time.Duration
	PipelineLimit           int
	TxDelay                 time.Duration
	SkipConfirmation        bool
	Drain                   bool
	NodeType                string
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
	return cmd
}

func doRun(config *runConfig) error {

	nodeType, err := pipeline.NodeTypeFromString(config.NodeType)
	if err != nil {
		return err
	}

	coinMarketCapAPIKey, err := loadFirstLine(config.CoinMarketCapAPIKeyPath)
	if err != nil {
		return errs.New("failed to load CoinMarketCap key: %v\n", err)
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

	log, err := openLog(config.DataDir)
	if err != nil {
		return err
	}

	fmt.Printf("Running %q payout...\n", config.Name)
	payer, err := CreatePayer(config.Ctx, log, config.PayerConfig, config.NodeAddress, config.ChainID, config.SpenderKeyPath)
	if err != nil {
		return err
	}

	runDir := filepath.Join(config.DataDir, config.Name)
	dbPath := payouts.DbPathFromDir(runDir)

	db, err := pipelinedb.OpenDB(context.Background(), dbPath, false)
	if err != nil {
		return err
	}
	defer db.Close()

	err = payouts.Run(config.Ctx,
		log,
		payouts.Config{
			DataDir: config.DataDir,
			Name:    config.Name,

			EstimateCacheExpiry: config.EstimateCacheExpiry,
			Quoter:              quoter,

			PipelineLimit: config.PipelineLimit,
			TxDelay:       config.TxDelay,
			Drain:         config.Drain,
			NodeType:      nodeType,
			PromptConfirm: promptConfirm,
		},
		db,
		payer)
	if err != nil {
		return err
	}
	fmt.Println("Payouts complete.")
	return nil
}
