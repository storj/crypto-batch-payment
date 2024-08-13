package payouts

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/fancy"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipeline"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
)

type Config struct {
	Quoter coinmarketcap.Quoter

	PipelineLimit int

	TxDelay time.Duration

	RetrySkipped bool

	Drain bool

	// ThresholdDivisor divides a payout amount to determine the fee threshold.
	// If a payout fee is larger than the fee threshold then the payout is
	// skipped. If ThresholdDivisor <= 0, no payouts will be skipped.
	ThresholdDivisor int

	// MaxFeeTolerationUSD is the maximum fee to tolerate for a single
	// transaction, in USD. If MaxFeeTolerationUSD <= zero, no maximum is
	// enforced. If the payout fee is larger than the toleration then the
	// pipeline will pause, checking every 5 seconds, until the fee comes down.
	MaxFeeTolerationUSD decimal.Decimal

	PromptConfirm func(label string) error
}

func Preview(ctx context.Context, config Config, db *pipelinedb.DB, paymentPayer payer.Payer) error {
	ethQuote, err := config.Quoter.GetQuote(ctx, coinmarketcap.ETH)
	if err != nil {
		return err
	}

	storjQuote, err := config.Quoter.GetQuote(ctx, coinmarketcap.STORJ)
	if err != nil {
		return err
	}

	ethBalance, err := paymentPayer.GetETHBalance(ctx)
	if err != nil {
		return err
	}

	tokenBalance, err := paymentPayer.GetTokenBalance(ctx)
	if err != nil {
		return err
	}

	gasInfo, err := paymentPayer.GetGasInfo(ctx)
	if err != nil {
		return err
	}

	var (
		ethPricePerWEI    = ethQuote.Price.Shift(-18)
		gasFeeCapUSD      = decimal.NewFromBigInt(gasInfo.GasFeeCap, 0).Mul(ethPricePerWEI)
		maxFeeEstimateUSD = gasFeeCapUSD.Mul(decimal.NewFromInt(int64(gasInfo.GasLimit)))
		// The threshold divisor normally divides the payout and compares that
		// value to the max fee estimate. If smaller than the max fee estimate
		// then the payout is too small to justify the fee amount. This
		// calculation is the same as multiplying the max fee estimate by the
		// threshold divisor and checking if that value exceeds the payout,
		// which we'll do here for calculating which payouts are under the
		// threshold at this point in time. The payout threshold calculated
		// here might be non-positive, which implies no threshold.
		payoutThresholdUSD = maxFeeEstimateUSD.Mul(decimal.NewFromInt(int64(config.ThresholdDivisor)))
	)

	stats, err := db.Stats(ctx, payoutThresholdUSD)
	if err != nil {
		return err
	}

	var (
		totalFeesUSD                 = maxFeeEstimateUSD.Mul(decimal.NewFromInt(stats.TotalPayouts))
		pendingFeesUSD               = maxFeeEstimateUSD.Mul(decimal.NewFromInt(stats.PendingPayouts))
		pendingBelowThresholdFeesUSD = maxFeeEstimateUSD.Mul(decimal.NewFromInt(stats.PendingPayoutsBelowThreshold))
		pendingAfterSkippingFeesUSD  = maxFeeEstimateUSD.Mul(decimal.NewFromInt(stats.PendingPayouts - stats.PendingPayoutsBelowThreshold))
		pendingAfterSkippingUSD      = stats.PendingUSD.Sub(stats.PendingPayoutsBelowThresholdUSD)

		tolerationExceeded = !config.MaxFeeTolerationUSD.IsPositive() || config.MaxFeeTolerationUSD.Cmp(maxFeeEstimateUSD) < 0
		willSkipPayouts    = stats.PendingPayoutsBelowThreshold > 0

		decimals     = paymentPayer.Decimals()
		storjTotal   = storjtoken.FromUSD(stats.TotalUSD, storjQuote.Price, decimals)
		storjPending = storjtoken.FromUSD(stats.PendingUSD, storjQuote.Price, decimals)

		ethBalanceETH = decimal.NewFromBigInt(ethBalance, -18)
	)

	fancy.Infof("**PAYMENT TYPE**............................: %s\n", paymentPayer)
	fancy.Infof("Current ETH Price...........................: $%s\n", ethQuote.Price)
	fancy.Infof("Current STORJ Price.........................: $%s\n", storjQuote.Price)
	fancy.Infof("Estimated Per-Gas Fee Cap...................: %s (wei)\n", gasInfo.GasFeeCap)
	fancy.Infof("Estimated Per-Gas Fee Tip...................: %s (wei)\n", gasInfo.GasTipCap)
	infoOrWarnf(tolerationExceeded, "Max Fee Estimate............................: $%s (for %d gas)\n", maxFeeEstimateUSD.Truncate(5), gasInfo.GasLimit)
	infoOrWarnf(tolerationExceeded, "Max Fee Toleration..........................: $%s\n", positiveOrDash(config.MaxFeeTolerationUSD))
	fmt.Println()
	fancy.Infof("Total Payees................................: %d\n", stats.Payees)
	fancy.Infof("Total Payouts...............................: %d\n", stats.TotalPayouts)
	fancy.Infof("Total Payouts USD...........................: $%s\n", stats.TotalUSD)
	fancy.Infof("Total Estimated Fees USD....................: $%s\n", totalFeesUSD.Truncate(5))
	fancy.Infof("Total USD...................................: $%s\n", stats.TotalUSD.Add(totalFeesUSD).Truncate(5))
	fmt.Println()
	fancy.Infof("Pending Payouts.............................: %d\n", stats.PendingPayouts)
	fancy.Infof("Pending Payouts USD.........................: $%s\n", stats.PendingUSD)
	fancy.Infof("Pending Estimated Fees USD..................: $%s\n", pendingFeesUSD.Truncate(5))
	fancy.Infof("Pending Total USD...........................: $%s\n", stats.PendingUSD.Add(pendingFeesUSD).Truncate(5))
	fmt.Println()
	fancy.Infof("Payouts Threshold..........................: $%s\n", payoutThresholdUSD)
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold.............: %d\n", stats.PendingPayoutsBelowThreshold)
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold USD.........: $%s\n", stats.PendingPayoutsBelowThresholdUSD)
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold Fees USD....: $%s\n", pendingBelowThresholdFeesUSD.Truncate(5))
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold Total USD...: $%s\n", stats.PendingPayoutsBelowThresholdUSD.Add(pendingBelowThresholdFeesUSD).Truncate(5))
	fmt.Println()
	fancy.Infof("Pending Payouts After Skipping..............: %d\n", stats.PendingPayouts-stats.PendingPayoutsBelowThreshold)
	fancy.Infof("Pending Payouts After Skipping USD..........: $%s\n", pendingAfterSkippingUSD)
	fancy.Infof("Pending After Skipping Estimated Fees USD...: $%s\n", pendingAfterSkippingFeesUSD.Truncate(5))
	fancy.Infof("Pending After Skipping Total USD............: $%s\n", pendingAfterSkippingUSD.Add(pendingAfterSkippingFeesUSD).Truncate(5))
	fmt.Println()
	fancy.Infof("Total Fees in ETH...........................: %s\n", totalFeesUSD.Div(ethQuote.Price))
	fancy.Infof("Pending Fees in ETH.........................: %s\n", pendingFeesUSD.Div(ethQuote.Price))
	fancy.Infof("Pending After Skips Fees in ETH.............: %s\n", pendingAfterSkippingFeesUSD.Div(ethQuote.Price))
	fancy.Infof("Current ETH balance ........................: %s\n", ethBalanceETH)
	fmt.Println()
	fancy.Infof("Total in STORJ ~ ...........................: %s\n", storjtoken.Pretty(storjTotal, decimals))
	fancy.Infof("Pending in STORJ ~ .........................: %s\n", storjtoken.Pretty(storjPending, decimals))
	fancy.Infof("Current STORJ balance ......................: %s\n", storjtoken.Pretty(tokenBalance, decimals))
	fmt.Println()
	fancy.Infof("Pending Payout Groups.......................: %d\n", stats.PendingPayoutGroups)
	fancy.Infof("Skipped Payout Groups.......................: %d\n", stats.SkippedPayoutGroups)
	fancy.Infof("Total Payout Groups.........................: %d\n", stats.TotalPayoutGroups)
	fmt.Println()
	fancy.Infof("Total Transactions..........................: %d\n", stats.TotalTransactions)
	fancy.Infof("Pending Transactions........................: %d\n", stats.PendingTransactions)
	fancy.Infof("Failed Transactions.........................: %d\n", stats.FailedTransactions)
	fancy.Infof("Confirmed Transactions......................: %d\n", stats.ConfirmedTransactions)
	fancy.Infof("Dropped Transactions........................: %d\n", stats.DroppedTransactions)

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
		Log:                 log,
		Quoter:              config.Quoter,
		DB:                  db,
		Limit:               config.PipelineLimit,
		Drain:               config.Drain,
		TxDelay:             config.TxDelay,
		ThresholdDivisor:    decimal.NewFromInt(int64(config.ThresholdDivisor)),
		MaxFeeTolerationUSD: config.MaxFeeTolerationUSD,
	})
	if err != nil {
		return err
	}

	if err := p.ProcessPayouts(ctx); err != nil {
		return err
	}

	return nil
}

func positiveOrDash(d decimal.Decimal) string {
	if d.IsPositive() {
		return d.String()
	}
	return "-"
}

func infoOrWarnf(warn bool, format string, args ...any) {
	if warn {
		fancy.Warnf(format, args...)
		return
	}
	fancy.Infof(format, args...)
}
