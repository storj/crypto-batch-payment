package payouts

import (
	"context"
	"fmt"
	"strings"
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
	// Quoter is optional when NeedsQuoter returns false.
	Quoter coinmarketcap.Quoter

	PipelineLimit int

	TxDelay time.Duration

	RetrySkipped bool

	Drain bool

	// ThresholdDivisor divides a payout amount to determine the fee threshold.
	// If a payout fee is larger than the fee threshold then the payout is
	// skipped. If ThresholdDivisor <= 0, no payouts will be skipped.
	ThresholdDivisor int

	// ThresholdAmountUSD is a fixed payout threshold, in USD. Payouts below
	// it are skipped. It cannot be combined with ThresholdDivisor. If
	// ThresholdAmountUSD <= 0, it is not enforced.
	ThresholdAmountUSD decimal.Decimal

	// MaxFeeTolerationUSD is the maximum fee to tolerate for a single
	// transaction, in USD. If MaxFeeTolerationUSD <= zero, no maximum is
	// enforced. If the payout fee is larger than the toleration then the
	// pipeline will pause, checking every 5 seconds, until the fee comes down.
	MaxFeeTolerationUSD decimal.Decimal

	PromptConfirm func(label string) error
}

// NeedsQuoter reports whether a quoter is required to pay out the given token
// with this config.
func (c Config) NeedsQuoter(symbol string) bool {
	return pipeline.NeedsQuoter(symbol, decimal.NewFromInt(int64(c.ThresholdDivisor)), c.MaxFeeTolerationUSD)
}

func Preview(ctx context.Context, config Config, db *pipelinedb.DB, paymentPayer payer.Payer) error {
	symbol := paymentPayer.Symbol()
	if config.Quoter == nil && config.NeedsQuoter(symbol) {
		return fmt.Errorf("quoter is required for %s payouts, threshold divisor or max fee toleration", symbol)
	}

	// Without a quoter the ETH price is unknown, so fees are only reported
	// in ETH.
	hasETHPrice := config.Quoter != nil
	var ethPrice decimal.Decimal
	if hasETHPrice {
		ethQuote, err := config.Quoter.GetQuote(ctx, coinmarketcap.ETH)
		if err != nil {
			return err
		}
		ethPrice = ethQuote.Price
	}

	tokenPrice, err := storjtoken.GetPrice(ctx, config.Quoter, symbol)
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
		maxFeeEstimateETH = decimal.NewFromBigInt(gasInfo.GasFeeCap, -18).Mul(decimal.NewFromInt(int64(gasInfo.GasLimit)))
		maxFeeEstimateUSD = maxFeeEstimateETH.Mul(ethPrice)
		// The threshold divisor normally divides the payout and compares that
		// value to the max fee estimate. If smaller than the max fee estimate
		// then the payout is too small to justify the fee amount. This
		// calculation is the same as multiplying the max fee estimate by the
		// threshold divisor and checking if that value exceeds the payout,
		// which we'll do here for calculating which payouts are under the
		// threshold at this point in time. The payout threshold calculated
		// here might be non-positive, which implies no threshold. A fixed
		// threshold amount replaces the divisor based one.
		payoutThresholdUSD = maxFeeEstimateUSD.Mul(decimal.NewFromInt(int64(config.ThresholdDivisor)))
	)
	if config.ThresholdAmountUSD.IsPositive() {
		payoutThresholdUSD = config.ThresholdAmountUSD
	}

	stats, err := db.Stats(ctx, payoutThresholdUSD)
	if err != nil {
		return err
	}

	// feesUSD formats fees (in ETH) as USD, optionally added to a USD
	// amount, or a dash if the ETH price is unknown.
	feesUSD := func(feesETH decimal.Decimal, plusUSD ...decimal.Decimal) string {
		if !hasETHPrice {
			return "-"
		}
		usd := feesETH.Mul(ethPrice)
		for _, d := range plusUSD {
			usd = usd.Add(d)
		}
		return "$" + usd.Truncate(5).String()
	}

	var (
		totalFeesETH                 = maxFeeEstimateETH.Mul(decimal.NewFromInt(stats.TotalPayouts))
		pendingFeesETH               = maxFeeEstimateETH.Mul(decimal.NewFromInt(stats.PendingPayouts))
		pendingBelowThresholdFeesETH = maxFeeEstimateETH.Mul(decimal.NewFromInt(stats.PendingPayoutsBelowThreshold))
		pendingAfterSkippingFeesETH  = maxFeeEstimateETH.Mul(decimal.NewFromInt(stats.PendingPayouts - stats.PendingPayoutsBelowThreshold))
		pendingAfterSkippingUSD      = stats.PendingUSD.Sub(stats.PendingPayoutsBelowThresholdUSD)

		tolerationExceeded = !config.MaxFeeTolerationUSD.IsPositive() || config.MaxFeeTolerationUSD.Cmp(maxFeeEstimateUSD) < 0
		willSkipPayouts    = stats.PendingPayoutsBelowThreshold > 0

		decimals     = paymentPayer.Decimals()
		tokenTotal   = storjtoken.FromUSD(stats.TotalUSD, tokenPrice, decimals)
		tokenPending = storjtoken.FromUSD(stats.PendingUSD, tokenPrice, decimals)

		ethBalanceETH = decimal.NewFromBigInt(ethBalance, -18)

		// The pipeline refuses to send a transaction it cannot cover, so point
		// out up front when the balance is not enough for what is left to pay.
		tokenBalanceShort = tokenBalance.Cmp(tokenPending) < 0
		ethBalanceShort   = ethBalanceETH.Cmp(pendingFeesETH) < 0
	)

	ethPriceText := "-"
	if hasETHPrice {
		ethPriceText = "$" + ethPrice.String()
	}

	fancy.Infof("**PAYMENT TYPE**............................: %s\n", paymentPayer)
	fancy.Infof("Current ETH Price...........................: %s\n", ethPriceText)
	fancy.Infof("%s: $%s\n", dotted("Current "+symbol+" Price"), tokenPrice)
	fancy.Infof("Estimated Per-Gas Fee Cap...................: %s (wei)\n", gasInfo.GasFeeCap)
	fancy.Infof("Estimated Per-Gas Fee Tip...................: %s (wei)\n", gasInfo.GasTipCap)
	infoOrWarnf(tolerationExceeded, "Max Fee Estimate............................: %s (for %d gas)\n", feesUSD(maxFeeEstimateETH), gasInfo.GasLimit)
	infoOrWarnf(tolerationExceeded, "Max Fee Toleration..........................: $%s\n", positiveOrDash(config.MaxFeeTolerationUSD))
	fmt.Println()
	fancy.Infof("Total Payees................................: %d\n", stats.Payees)
	fancy.Infof("Total Payouts...............................: %d\n", stats.TotalPayouts)
	fancy.Infof("Total Payouts USD...........................: $%s\n", stats.TotalUSD)
	fancy.Infof("Total Estimated Fees USD....................: %s\n", feesUSD(totalFeesETH))
	fancy.Infof("Total USD...................................: %s\n", feesUSD(totalFeesETH, stats.TotalUSD))
	fmt.Println()
	fancy.Infof("Pending Payouts.............................: %d\n", stats.PendingPayouts)
	fancy.Infof("Pending Payouts USD.........................: $%s\n", stats.PendingUSD)
	fancy.Infof("Pending Estimated Fees USD..................: %s\n", feesUSD(pendingFeesETH))
	fancy.Infof("Pending Total USD...........................: %s\n", feesUSD(pendingFeesETH, stats.PendingUSD))
	fmt.Println()
	fancy.Infof("Payouts Threshold..........................: $%s\n", payoutThresholdUSD)
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold.............: %d\n", stats.PendingPayoutsBelowThreshold)
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold USD.........: $%s\n", stats.PendingPayoutsBelowThresholdUSD)
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold Fees USD....: %s\n", feesUSD(pendingBelowThresholdFeesETH))
	infoOrWarnf(willSkipPayouts, "Pending Payouts Below Threshold Total USD...: %s\n", feesUSD(pendingBelowThresholdFeesETH, stats.PendingPayoutsBelowThresholdUSD))
	fmt.Println()
	fancy.Infof("Pending Payouts After Skipping..............: %d\n", stats.PendingPayouts-stats.PendingPayoutsBelowThreshold)
	fancy.Infof("Pending Payouts After Skipping USD..........: $%s\n", pendingAfterSkippingUSD)
	fancy.Infof("Pending After Skipping Estimated Fees USD...: %s\n", feesUSD(pendingAfterSkippingFeesETH))
	fancy.Infof("Pending After Skipping Total USD............: %s\n", feesUSD(pendingAfterSkippingFeesETH, pendingAfterSkippingUSD))
	fmt.Println()
	fancy.Infof("Total Fees in ETH...........................: %s\n", totalFeesETH)
	fancy.Infof("Pending Fees in ETH.........................: %s\n", pendingFeesETH)
	fancy.Infof("Pending After Skips Fees in ETH.............: %s\n", pendingAfterSkippingFeesETH)
	infoOrWarnf(ethBalanceShort, "Current ETH balance ........................: %s\n", ethBalanceETH)
	fmt.Println()
	fancy.Infof("%s: %s\n", dotted("Total in "+symbol+" ~ "), storjtoken.Pretty(tokenTotal, decimals, symbol))
	fancy.Infof("%s: %s\n", dotted("Pending in "+symbol+" ~ "), storjtoken.Pretty(tokenPending, decimals, symbol))
	infoOrWarnf(tokenBalanceShort, "%s: %s\n", dotted("Current "+symbol+" balance "), storjtoken.Pretty(tokenBalance, decimals, symbol))
	fmt.Println()
	fancy.Infof("Pending Payout Groups.......................: %d\n", stats.PendingPayoutGroups)
	fancy.Infof("  Unstarted (no tx yet).....................: %d\n", stats.UnstartedPayoutGroups)
	fancy.Infof("  In-Flight (tx pending)....................: %d\n", stats.InFlightPayoutGroups)
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
		ThresholdAmountUSD:  config.ThresholdAmountUSD,
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

// dotted pads the label with dots to align it with the other preview lines.
func dotted(label string) string {
	const width = 44
	if len(label) >= width {
		return label
	}
	return label + strings.Repeat(".", width-len(label))
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
