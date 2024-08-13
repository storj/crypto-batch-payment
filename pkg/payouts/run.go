package payouts

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipeline"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
	"storj.io/crypto-batch-payment/pkg/txparams"
)

type Config struct {
	Quoter coinmarketcap.Quoter

	GasCaps txparams.Getter

	PipelineLimit int

	TxDelay time.Duration

	Drain bool

	PromptConfirm func(label string) error
}

func Preview(ctx context.Context, config Config, db *pipelinedb.DB, paymentPayer payer.Payer) error {
	stats, err := db.Stats(ctx)
	if err != nil {
		return err
	}

	storjQuote, err := config.Quoter.GetQuote(ctx, coinmarketcap.STORJ)
	if err != nil {
		return err
	}

	balance, err := paymentPayer.GetTokenBalance(ctx)
	if err != nil {
		return err
	}

	decimals := paymentPayer.Decimals()
	estimatedSTORJ := storjtoken.FromUSD(stats.PendingUSD, storjQuote.Price, decimals)

	fmt.Printf("**PAYMENT TYPE**............: %s\n", paymentPayer)
	fmt.Printf("Current STORJ Price.........: $%s\n", storjQuote.Price.String())
	fmt.Println()
	fmt.Printf("Total Payees................: %d\n", stats.Payees)
	fmt.Printf("Total Payouts...............: %d\n", stats.TotalPayouts)
	fmt.Printf("Total Payout Groups.........: %d\n", stats.TotalPayoutGroups)
	fmt.Printf("Total USD...................: $%s\n", stats.TotalUSD.String())
	fmt.Println()
	fmt.Printf("Pending Payouts.............: %d\n", stats.PendingPayouts)
	fmt.Printf("Pending Payout Groups.......: %d\n", stats.PendingPayoutGroups)
	fmt.Printf("Pending USD.................: $%s\n", stats.PendingUSD.String())
	fmt.Printf("Pending in STORJ ~ .........: %s\n", storjtoken.Pretty(estimatedSTORJ, decimals))
	fmt.Printf("Current STORJ balance ......: %s\n", storjtoken.Pretty(balance, decimals))
	fmt.Println()
	fmt.Printf("Total Transactions..........: %d\n", stats.TotalTransactions)
	fmt.Printf("Pending Transactions........: %d\n", stats.PendingTransactions)
	fmt.Printf("Failed Transactions.........: %d\n", stats.FailedTransactions)
	fmt.Printf("Confirmed Transactions......: %d\n", stats.ConfirmedTransactions)
	fmt.Printf("Dropped Transactions........: %d\n", stats.DroppedTransactions)

	err = paymentPayer.PrintEstimate(ctx, stats.PendingPayoutGroups)
	if err != nil {
		return err
	}
	fmt.Println()

	if config.Drain {
		if err := config.PromptConfirm("Drain"); err != nil {
			return err
		}
	} else {
		if err := config.PromptConfirm("Proceed"); err != nil {
			return err
		}
	}

	return nil
}

func Run(ctx context.Context, log *zap.Logger, config Config, db *pipelinedb.DB, paymentPayer payer.Payer) error {
	p, err := pipeline.New(paymentPayer, pipeline.Config{
		Log:     log,
		Quoter:  config.Quoter,
		GasCaps: config.GasCaps,
		DB:      db,
		Limit:   config.PipelineLimit,
		Drain:   config.Drain,
		TxDelay: config.TxDelay,
	})
	if err != nil {
		return err
	}

	if err := p.ProcessPayouts(ctx); err != nil {
		return err
	}

	return nil
}
