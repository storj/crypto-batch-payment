// Package pipeline tracks and executes payouts
package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"time"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/storjtoken"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
	"go.uber.org/zap"
)

const (
	// txStatusPollInterval is often to poll for transaction status.
	txStatusPollInterval = time.Second

	// notDroppedUntil is how long to wait after submitting a transaction
	// before it should be considered dropped.
	notDroppedUntil = time.Second * 30

	// DefaultLimit is the default limit of concurrent payout group transfers
	// to manage at a time. Both parity and geth have a per-address tx limit
	// lower bound of 16, but can be configured for more. As such, 16 is the
	// safe default.
	DefaultLimit = 16

	// DefaultTxDelay is the default tx delay (see TxDelay in Config).
	DefaultTxDelay = time.Duration(0)
)

var (
	// zero is a big int set to 0 for convenience.
	zero = big.NewInt(0)

	// errSkipped is returned from prepareTransaction when the
	// transaction fees are too high relative to the payout amount.
	errSkipped = errors.New("max fee exceeded")

	// errMaxFeeExceeded is returned from prepareTransaction when the
	// transaction is estimated to exceed the MaxFeeTolerationUSD in fees.
	errMaxFeeExceeded = errors.New("max fee exceeded")
)

type sleepFunc = func(context.Context, time.Duration) error

type Config struct {
	// Log is the logger for logging pipeline progress
	Log *zap.Logger

	// Owner is the address of the account that the STORJ token will be paid
	// from. In a Transfer-based flow, the Owner will match the address derived
	// from the Spender key. In a TransferFrom-based flow, it will be a
	// different account.
	Owner common.Address

	// Quoter is used to get price quotes for STORJ token
	Quoter coinmarketcap.Quoter

	// ThresholdDivisor divides a payout amount to determine the fee threshold.
	// If a payout fee is larger than the fee threshold then the payout is
	// skipped. If ThresholdDivisor <= 0, no payouts will be skipped.
	ThresholdDivisor decimal.Decimal

	// MaxFeeTolerationUSD is the maximum fee to tolerate for a single
	// transaction, in USD. If MaxFeeTolerationUSD <= zero, no maximum is
	// enforced.
	MaxFeeTolerationUSD decimal.Decimal

	// DB is the the payout database
	DB *pipelinedb.DB

	// Limit is how many transfer groups to process at a time. Too few and
	// the payouts won't be done as quickly as they otherwise might be. Too
	// many and the risk of dropped transactions increases (ethereum nodes
	// have limits to how many open transactions from the same source address
	// they will hold at a time). Defaults to DefaultLimit if unset.
	Limit int

	// TxDelay is a delay between issuing consecutive transactions to work
	// around bugs in parity, with race conditions around returning the correct
	// pending nonce. Defaults to zero.
	TxDelay time.Duration

	// Drain, if true, causes the pipeline to only finish processing existing
	// transactions and then halt.
	Drain bool

	// test hook used to step the polling loop
	stepInCh chan chan []*pipelinedb.NonceGroup

	// test hook used to manipulate the polling interval
	pollInterval time.Duration

	// test hook used for sleeping so we aren't dependent on real time
	sleep sleepFunc
}

type Pipeline struct {
	log *zap.Logger

	owner               common.Address
	quoter              coinmarketcap.Quoter
	thresholdDivisor    decimal.Decimal
	maxFeeTolerationUSD decimal.Decimal
	db                  *pipelinedb.DB
	limit               int
	txDelay             time.Duration
	drain               bool
	payer               payer.Payer

	pollInterval  time.Duration
	sleep         sleepFunc
	expectedNonce uint64
	nonceGroups   []*pipelinedb.NonceGroup
}

func New(payer payer.Payer, config Config) (*Pipeline, error) {
	switch {
	case config.Log == nil:
		return nil, errors.New("log is required")
	case config.Quoter == nil:
		return nil, errors.New("quoter is required")
	case config.DB == nil:
		return nil, errors.New("db is required")
	}
	if config.Limit == 0 {
		config.Limit = DefaultLimit
	}
	if config.pollInterval == 0 {
		config.pollInterval = txStatusPollInterval
	}
	if config.sleep == nil {
		config.sleep = sleep
	}

	return &Pipeline{
		log:                 config.Log,
		owner:               config.Owner,
		quoter:              config.Quoter,
		thresholdDivisor:    config.ThresholdDivisor,
		maxFeeTolerationUSD: config.MaxFeeTolerationUSD,
		db:                  config.DB,
		limit:               config.Limit,
		txDelay:             config.TxDelay,
		drain:               config.Drain,
		pollInterval:        config.pollInterval,
		sleep:               config.sleep,
		payer:               payer,
	}, nil
}

func (p *Pipeline) ProcessPayouts(ctx context.Context) error {
	p.log.Info("Processing payouts",
		zap.Int("limit", p.limit),
		zap.String("tx-delay", p.txDelay.String()),
		zap.Bool("drain", p.drain),
	)

	if err := p.initPayout(ctx); err != nil {
		return err
	}

	for {
		done, err := p.payoutStep(ctx)
		if err != nil {
			return err
		}
		if done {
			break
		}

		select {
		case <-time.After(p.pollInterval):
		case <-ctx.Done():
			p.log.Error("Processing interrupted", zap.Error(ctx.Err()))
			return ctx.Err()
		}

	}
	return nil
}

func (p *Pipeline) initPayout(ctx context.Context) error {
	nonceGroups, err := p.db.FetchUnfinishedTransactionsSortedIntoNonceGroups(ctx)
	if err != nil {
		return err
	}

	unstarted, err := p.db.CountUnfinishedUnattachedPayoutGroup(ctx)
	if err != nil {
		return err
	}

	p.log.Info("Payout groups loaded",
		zap.Int64("unstarted", unstarted),
		zap.Int("pending", len(nonceGroups)),
	)

	p.nonceGroups = nonceGroups
	done, err := p.checkNonceGroups(ctx)
	if done {
		return err
	}
	return nil
}

func (p *Pipeline) payoutStep(ctx context.Context) (bool, error) {
	// Trim off nonce groups that have no more transactions. This only
	// happens when a nonce group has been confirmed, failed or is being skipped.
	var finished int
	for len(p.nonceGroups) > 0 && len(p.nonceGroups[0].Txs) == 0 {
		p.log.Info("Nonce group finished", zap.Uint64("nonce", p.nonceGroups[0].Nonce))
		p.nonceGroups = p.nonceGroups[1:]
		finished++
	}
	if finished > 0 {
		p.log.Info("Pipeline status", zap.Int("len", len(p.nonceGroups)), zap.Int("limit", p.limit))
	}

	added, err := p.fillPipeline(ctx)
	if err != nil {
		return true, err
	}

	// Pipeline is empty
	if len(p.nonceGroups) == 0 {
		if p.drain {
			p.log.Info("Drained existing transactions.")
		} else {
			p.log.Info("Processed all payout groups")
		}
		return true, nil
	}

	if added {
		unfinishedPayouts, totalPayouts, err := p.db.FetchPayoutProgress(ctx)
		if err != nil {
			return true, err
		}
		p.log.Info("Waiting on nonce groups...",
			zap.Int("pending", len(p.nonceGroups)),
			zap.Int64("unfinished payouts", unfinishedPayouts),
			zap.Int64("total payouts", totalPayouts),
		)
	}

	done, err := p.checkNonceGroups(ctx)
	if done {
		return true, err
	}
	return false, nil
}

func (p *Pipeline) fillPipeline(ctx context.Context) (_ bool, err error) {
	if len(p.nonceGroups) >= p.limit || p.drain {
		return false, nil
	}

	// Fill up the pipeline
	var added int
	for i := 0; len(p.nonceGroups) < p.limit; i++ {
		payoutGroup, err := p.db.FetchFirstUnfinishedUnattachedPayoutGroup(ctx)
		if err != nil {
			return false, err
		}
		if payoutGroup == nil {
			// no payout groups to add
			break
		}

		// Either continue with the next nonce (if there is a nonce group
		// to increment from) or grab the account nonce according to the
		// blockchain.
		var nextNonce uint64
		if len(p.nonceGroups) > 0 {
			nextNonce = p.nonceGroups[len(p.nonceGroups)-1].Nonce + 1
			p.log.Info("Nonce from nonce group", zap.Uint64("nextNonce", nextNonce))
		} else {
			nextNonce, err = p.payer.NextNonce(ctx)
			if err != nil {
				return false, errs.New("unable to obtain next nonce from blockchain: %v", err)
			}
			p.log.Info("Nonce from chain", zap.Uint64("nextNonce", nextNonce))
			// NonceAt can return an earlier nonce than expected if the
			// just-mined block cleared out the pipeline but the timing is
			// weird enough for the node to not return right nonce based on
			// the transactions in that block. Bail out for now if that
			// happens to prevent recording and sending a transaction with
			// a known bad nonce.
			// TODO: we could spin here for a time until the node returns
			// the expected nonce...
			if p.expectedNonce > 0 && nextNonce < p.expectedNonce {
				return false, errs.New("node returned used nonce %d; expected >= %d", nextNonce, p.expectedNonce)
			}
		}

		tx, err := p.sendTransaction(ctx, payoutGroup.ID, nextNonce)
		switch {
		case errors.Is(err, errSkipped):
			continue
		case err != nil:
			return false, err
		}

		p.expectedNonce = nextNonce + 1
		p.nonceGroups = append(p.nonceGroups, &pipelinedb.NonceGroup{
			Nonce:         tx.Nonce,
			PayoutGroupID: payoutGroup.ID,
			Txs:           []pipelinedb.Transaction{*tx},
		})
		p.log.Debug("Pipeline status", zap.Int("len", len(p.nonceGroups)), zap.Int("limit", p.limit))

		if err := sleepFor(ctx, p.txDelay); err != nil {
			return false, err
		}

		added++
	}

	return added > 0, nil
}

func (p *Pipeline) checkNonceGroups(ctx context.Context) (bool, error) {
	// Check the status of each nonce group in the pipeline.
	failedCount := 0
checkLoop:
	for i := range p.nonceGroups {
		log := p.log.With(zap.Uint64("nonce", p.nonceGroups[i].Nonce), zap.Int64("payout-group-id", p.nonceGroups[i].PayoutGroupID))

		log.Debug("Checking nonce group",
			zap.Int("txs", len(p.nonceGroups[i].Txs)),
		)

		var err error
		var state pipelinedb.TxState
		var all []*pipelinedb.TxStatus
		state, all, err = p.payer.CheckNonceGroup(ctx, log, p.nonceGroups[i], failedCount > 0)
		if err != nil {
			return true, err
		}

		switch state {
		case pipelinedb.TxDropped:
			// Do not consider transactions dropped until notDroppedUntil time
			// has elapsed since their creation. This avoids false-positives
			// when using networks like Infura, whose nodes are eventually
			// consistent.
			if time.Since(youngestTransactionTime(p.nonceGroups[i].Txs)) < notDroppedUntil {
				break checkLoop
			}
			// All the transactions have been dropped. Send another transaction for
			// this nonce group. Indicate to the caller that there was a drop.
			tx, err := p.sendTransaction(ctx, p.nonceGroups[i].PayoutGroupID, p.nonceGroups[i].Nonce)
			switch {
			case errors.Is(err, errSkipped):
				// Cannot retry the nonce group since it has dropped below
				// the fee threshold.
				p.nonceGroups[i].Txs = nil
			case err != nil:
				return true, err
			default:
				p.nonceGroups[i].Txs = append(p.nonceGroups[i].Txs, *tx)
				break checkLoop
			}
		case pipelinedb.TxPending:
			// This group has not confirmed/failed. Don't look at the rest
			// until we know it's fate. It is dangerous to look further
			// since the node has shown to be unreliable in reporting
			// transaction state for transactions of a later nonce.
			break checkLoop
		case pipelinedb.TxFailed:
			// The transaction has failed. Record the failure and
			// return.
			if err := p.db.FinalizeNonceGroup(ctx, p.nonceGroups[i], all); err != nil {
				return true, err
			}
			p.nonceGroups[i].Txs = nil
			failedCount++
		case pipelinedb.TxConfirmed:
			if err := p.db.FinalizeNonceGroup(ctx, p.nonceGroups[i], all); err != nil {
				return true, err
			}
			p.nonceGroups[i].Txs = nil
		}
	}

	if failedCount > 0 {
		return true, errs.New("One or more transactions failed, possibly due to insufficient balances")
	}
	return false, nil
}

func (p *Pipeline) sendTransaction(ctx context.Context, payoutGroupID int64, nonce uint64) (*pipelinedb.Transaction, error) {
	for {
		tx, err := p.trySendTransaction(ctx, payoutGroupID, nonce)
		switch {
		case errors.Is(err, errMaxFeeExceeded):
			// handled below
		case err != nil:
			return nil, err
		default:
			return tx, nil
		}

		p.log.Info("Max fee toleration was exceeded; waiting 5 seconds before trying again")
		if err := p.sleep(ctx, 5*time.Second); err != nil {
			return nil, err
		}
	}
}

func (p *Pipeline) trySendTransaction(ctx context.Context, payoutGroupID int64, nonce uint64) (*pipelinedb.Transaction, error) {
	txLog := p.log.With(
		zap.Int64("payout-group-id", payoutGroupID),
		zap.Stringer("owner", p.owner),
		zap.Uint64("nonce", nonce),
	)

	payouts, err := p.db.FetchPayoutGroupPayouts(ctx, payoutGroupID)
	switch {
	case err != nil:
		return nil, err
	case len(payouts) == 0:
		return nil, errs.New("no payouts associated with transfer %d", payoutGroupID)
	case len(payouts) > 1:
		return nil, errs.New("multitransfer is not supported")
	}

	payout := payouts[0]

	txLog = txLog.With(zap.Stringer("usd", payout.USD))

	ethPrice, err := p.getQuote(ctx, coinmarketcap.ETH)
	if err != nil {
		return nil, err
	}
	txLog = txLog.With(zap.Stringer("eth-price", ethPrice))

	// Calculate how many STORJ tokens are required to cover the payout.
	storjPrice, err := p.getQuote(ctx, coinmarketcap.STORJ)
	if err != nil {
		return nil, err
	}
	txLog = txLog.With(zap.Stringer("storj-price", storjPrice))

	storjTokens := storjtoken.FromUSD(payout.USD, storjPrice, p.payer.Decimals())
	txLog = txLog.With(zap.Stringer("storj-tokens", storjTokens))

	if storjTokens.Cmp(zero) <= 0 {
		txLog.Error("STORJ token amount must be greater than zero")
		return nil, errs.New("cannot transfer %s tokens for payout group %d: must be more than zero", storjTokens, payoutGroupID)
	}

	// Check the STORJ balance to make sure there is enough.
	storjBalance, err := p.payer.GetTokenBalance(ctx)
	if err != nil {
		return nil, err
	}
	txLog = txLog.With(zap.Stringer("storj-balance", storjBalance))

	if storjBalance.Cmp(storjTokens) < 0 {
		return nil, errs.New("not enough STORJ balance to cover transfer (%s < %s)", storjBalance, storjTokens)
	}

	params := payer.TransactionParams{
		Nonce:  nonce,
		Payee:  payout.Payee,
		Tokens: storjTokens,
	}

	rawTx, from, err := p.payer.CreateRawTransaction(ctx, txLog, params)
	if err != nil {
		return nil, err
	}

	maxFeeGWEI := decimal.NewFromInt(int64(rawTx.GasLimit)).Mul(decimal.NewFromBigInt(rawTx.GasFeeCap, 0))
	maxFeeUSD := ethPrice.Mul(maxFeeGWEI).Shift(-18)

	txLog = txLog.With(
		zap.Uint64("gas-limit", rawTx.GasLimit),
		zap.Stringer("gas-fee-cap", rawTx.GasFeeCap),
		zap.Stringer("max-fee-usd", maxFeeUSD),
	)

	if p.maxFeeTolerationUSD.IsPositive() && p.maxFeeTolerationUSD.Cmp(maxFeeUSD) < 0 {
		txLog.Warn("Payout max fee exceeds the max fee toleration", zap.Stringer("max-fee-toleration-usd", p.maxFeeTolerationUSD))
		return nil, errMaxFeeExceeded
	}

	if p.thresholdDivisor.IsPositive() {
		thresholdUSD := payout.USD.Div(p.thresholdDivisor)
		if !payout.Mandatory && thresholdUSD.Cmp(maxFeeUSD) < 0 {
			txLog.Warn("Skipping transaction because payout is below the minimum payout threshold", zap.Stringer("threshold-usd", thresholdUSD))
			if err := p.db.SetPayoutGroupStatus(ctx, payoutGroupID, pipelinedb.PayoutGroupSkipped); err != nil {
				return nil, errs.Wrap(err)
			}
			return nil, errSkipped
		}
	}

	// Clear the payout group status
	if err := p.db.SetPayoutGroupStatus(ctx, payoutGroupID, ""); err != nil {
		return nil, errs.Wrap(err)
	}

	rawTxJSON, err := json.Marshal(rawTx.Raw)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	tx, err := p.db.CreateTransaction(ctx,
		pipelinedb.Transaction{
			PayoutGroupID: payoutGroupID,
			Hash:          rawTx.Hash,
			Nonce:         rawTx.Nonce,
			Owner:         p.owner,
			Spender:       from,
			StorjPrice:    storjPrice,
			StorjTokens:   storjTokens,
			Raw:           rawTxJSON,
		})

	if err != nil {
		return nil, err
	}

	if err := p.payer.SendTransaction(ctx, txLog, rawTx); err != nil {
		return nil, err
	}
	return tx, nil
}

func (p *Pipeline) getQuote(ctx context.Context, symbol coinmarketcap.Symbol) (decimal.Decimal, error) {
	storjQuote, err := p.quoter.GetQuote(ctx, symbol)
	if err != nil {
		return decimal.Decimal{}, errs.Wrap(err)
	}
	return storjQuote.Price, nil
}

func sleepFor(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		timer.Stop()
		return ctx.Err()
	}
}

func youngestTransactionTime(txs []pipelinedb.Transaction) time.Time {
	// This _should_ be the last transaction in the list, but just in case...
	var youngest time.Time
	for _, tx := range txs {
		if youngest.Before(tx.CreatedAt) {
			youngest = tx.CreatedAt
		}
	}
	return youngest
}

func safeBigInt(b *big.Int) string {
	if b == nil {
		return "unset"
	}
	return b.String()
}

func sleep(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return nil
	}
}
