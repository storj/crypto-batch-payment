package main

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/payouts"
	"storj.io/crypto-batch-payment/pkg/pipeline"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

type payerTransferConfig struct {
	*payerCommandConfig
	Amount      string
	Destination string
	Price       string
}

func newPayerTransferCommand(parentConfig *payerCommandConfig) *cobra.Command {
	config := &payerTransferConfig{
		payerCommandConfig: parentConfig,
	}
	cmd := &cobra.Command{
		Use:   "transfer <privateKeyFile>",
		Short: "Transfer the tokens with the defined valur to the account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkCmd(doPayerTransfer(config, args[0]))
		},
	}
	cmd.Flags().StringVarP(
		&config.Price,
		"price", "",
		"0",
		"Actual price of the STORJ token to be used.")
	cmd.Flags().StringVarP(
		&config.Amount,
		"amount", "",
		"0",
		"Value in USD (!) to transfer")
	cmd.Flags().StringVarP(
		&config.Destination,
		"destination", "",
		"",
		"Destination wallet to use.")
	RegisterFlags(cmd, &config.payerCommandConfig.PayerConfig)
	return cmd
}

func doPayerTransfer(config *payerTransferConfig, spenderKeyPath string) error {
	promptConfirm := promptConfirm

	log, err := openLog(config.DataDir)
	if err != nil {
		return err
	}

	fmt.Println("Running ad-hoc payout (single transfer)...")
	payer, err := CreatePayer(config.Ctx, log, config.PayerConfig, config.NodeAddress, config.ChainID, spenderKeyPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	db, err := pipelinedb.OpenInMemoryDB(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	usdAmount, err := decimal.NewFromString(config.Amount)
	if err != nil {
		return err
	}
	if usdAmount.IntPart() == 0 {
		return errs.New("Please specify USD mount")
	}
	if config.Destination == "" {
		return errs.New("Please specify destination (payee)")
	}
	toAddress := common.HexToAddress(config.Destination)

	if err := db.CreatePayoutGroup(ctx, 1, []*pipelinedb.Payout{
		{
			CSVLine:       1,
			Payee:         toAddress,
			USD:           usdAmount,
			PayoutGroupID: 1,
		},
	}); err != nil {
		return err
	}

	fmt.Printf("Payee ......................: %s\n", toAddress.String())
	err = payouts.Run(config.Ctx,
		log,
		payouts.Config{
			Quoter: coinmarketcap.QuoterFunc(func(ctx context.Context, symbol coinmarketcap.Symbol) (*coinmarketcap.Quote, error) {
				price, err := decimal.NewFromString(config.Price)
				if price.IntPart() == 0 {
					return nil, errs.New("Please set the actual token price")
				}
				return &coinmarketcap.Quote{
					Price:       price,
					LastUpdated: time.Now(),
				}, err
			}),
			PipelineLimit: pipeline.DefaultLimit,
			TxDelay:       pipeline.DefaultTxDelay,
			Drain:         false,
			PromptConfirm: promptConfirm,
		},
		db,
		payer,
	)
	if err != nil {
		return err
	}
	fmt.Println("Payouts complete.")
	payouts, err := db.FetchPayouts(ctx)
	if err != nil {
		return err
	}
	for _, p := range payouts {
		group, err := db.FetchPayoutGroup(ctx, p.PayoutGroupID)
		if err != nil {
			return err
		}
		fmt.Println("Payout TX: ", group.FinalTxHash)
	}
	return nil
}
