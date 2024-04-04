// Package pipeline tracks and executes payouts
package pipeline

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
)

const (
	// txStatusPollInterval is often to poll for transaction status
	txStatusPollInterval = time.Second

	// DefaultLimit is the default limit of concurrent payout group transfers
	// to manage at a time. Both parity and geth have a per-address tx limit
	// lower bound of 16, but can be configured for more. As such, 16 is the
	// safe default.
	DefaultLimit = 16

	// DefaultTxDelay is the default tx delay (see TxDelay in Config)
	DefaultTxDelay = time.Duration(0)
)

var (
	// zero is a big int set to 0 for convenience.
	zero = big.NewInt(0)
)

type PipelineConfig struct {
	// Log is the logger for logging pipeline progress
	Log *zap.Logger

	// Owner is the address of the account that the STORJ token will be paid
	// from. In a Transfer-based flow, the Owner will match the address derived
	// from the Spender key. In a TransferFrom-based flow, it will be a
	// different account.
	Owner common.Address

	// Quoter is used to get price quotes for STORJ token
	Quoter coinmarketcap.Quoter

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
}

type Pipeline struct {
	log *zap.Logger

	owner   common.Address
	quoter  coinmarketcap.Quoter
	db      *pipelinedb.DB
	limit   int
	txDelay time.Duration
	drain   bool
	payer   payer.Payer

	pollInterval  time.Duration
	expectedNonce uint64
	nonceGroups   []*pipelinedb.NonceGroup
}

func NewPipeline(payer payer.Payer, config PipelineConfig) (*Pipeline, error) {
	if config.Limit == 0 {
		config.Limit = DefaultLimit
	}
	if config.pollInterval == 0 {
		config.pollInterval = txStatusPollInterval
	}

	return &Pipeline{
		log:          config.Log,
		owner:        config.Owner,
		quoter:       config.Quoter,
		db:           config.DB,
		limit:        config.Limit,
		txDelay:      config.TxDelay,
		drain:        config.Drain,
		pollInterval: config.pollInterval,
		payer:        payer,
	}, nil
}

func (p *Pipeline) ProcessPayouts(ctx context.Context) error {
	p.log.Info("Processing payouts",
		zap.Int("limit", p.limit),
		zap.String("tx-delay", p.txDelay.String()),
		zap.Bool("drain", p.drain),
	)

	err := p.initPayout(ctx)
	if err != nil {
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
	// happens when a nonce group has been confirmed or failed.
	var finished int
	for len(p.nonceGroups) > 0 && len(p.nonceGroups[0].Txs) == 0 {
		p.log.Info("Nonce group finished", zap.Uint64("nonce", p.nonceGroups[0].Nonce))
		p.nonceGroups = p.nonceGroups[1:]
		finished++
	}
	if finished > 0 {
		p.log.Info("Pipeline status", zap.Int("len", len(p.nonceGroups)), zap.Int("limit", p.limit))
	}

	// Fill up the pipeline
	var added bool
	for i := 0; len(p.nonceGroups) < p.limit && !p.drain; i++ {
		payoutGroup, err := p.db.FetchFirstUnfinishedUnattachedPayoutGroup(ctx)
		if err != nil {
			return true, err
		}
		if payoutGroup == nil {
			// no payout groups to add
			break
		}

		if i > 0 && p.txDelay > 0 {
			if err := sleepFor(ctx, p.txDelay); err != nil {
				return true, err
			}
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
				return true, errs.New("unable to obtain next nonce from blockchain: %v", err)
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
				return true, errs.New("node returned used nonce %d; expected >= %d", nextNonce, p.expectedNonce)
			}
			p.expectedNonce = nextNonce + 1
		}

		tx, err := p.sendTransaction(ctx, payoutGroup.ID, nextNonce)
		if err != nil {
			return true, err
		}

		p.nonceGroups = append(p.nonceGroups, &pipelinedb.NonceGroup{
			Nonce:         tx.Nonce,
			PayoutGroupID: payoutGroup.ID,
			Txs:           []pipelinedb.Transaction{*tx},
		})
		p.log.Debug("Pipeline status", zap.Int("len", len(p.nonceGroups)), zap.Int("limit", p.limit))
		added = true
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
			// All the transactions have been dropped. Send another transaction for
			// this nonce group. Indicate to the caller that there was a drop.
			tx, err := p.sendTransaction(ctx, p.nonceGroups[i].PayoutGroupID, p.nonceGroups[i].Nonce)
			if err != nil {
				return true, err
			}
			p.nonceGroups[i].Txs = append(p.nonceGroups[i].Txs, *tx)
			break checkLoop
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
	payouts, err := p.db.FetchPayoutGroupPayouts(ctx, payoutGroupID)
	if err != nil {
		return nil, err
	}
	if len(payouts) == 0 {
		return nil, errs.New("no payouts associated with transfer %d", payoutGroupID)
	}

	sumUSD := decimal.Zero
	for _, p := range payouts {
		sumUSD = decimal.Sum(sumUSD, p.USD)
	}

	for {
		unmet, err := p.payer.CheckPreconditions(ctx)
		if err != nil {
			return nil, err
		}
		if len(unmet) == 0 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(5 * time.Second):
			p.log.Info("One or more preconditions are not met, waiting for 5 seconds", zap.Strings("unmet", unmet))
		}
	}

	decimals, err := p.payer.GetTokenDecimals(ctx)
	if err != nil {
		return nil, err
	}

	storjPrice, err := p.getStorjPrice(ctx)
	if err != nil {
		return nil, err
	}

	storjTokens := storjtoken.FromUSD(sumUSD, storjPrice, decimals)
	if storjTokens.Cmp(zero) <= 0 {
		p.log.Error("STORJ token amount must be greater than zero",
			zap.Int64("payout group", payoutGroupID),
			zap.String("usd", sumUSD.String()),
			zap.String("price", storjPrice.String()),
			zap.String("tokens", storjTokens.String()),
		)
		return nil, errs.New("cannot transfer %s tokens for payout group %d: must be more than zero", storjTokens, payoutGroupID)
	}

	// Check the STORJ balance to make sure there is enough.
	storjBalance, err := p.payer.GetTokenBalance(ctx)
	if err != nil {
		return nil, err
	}
	if storjBalance.Cmp(storjTokens) < 0 {
		return nil, errs.New("not enough STORJ balance to cover transfer (%s < %s)", storjBalance, storjTokens)
	}

	txLog := p.log.With(
		zap.Uint64("nonce", nonce),
		zap.Int64("payout-group-id", payoutGroupID),
		zap.String("owner", p.owner.String()),
		zap.String("storj-price", storjPrice.String()),
		zap.String("storj-tokens", storjTokens.String()),
		zap.String("storj-balance", storjBalance.String()))

	rawTx, from, err := p.payer.CreateRawTransaction(ctx, txLog, payouts, nonce, storjPrice)
	if err != nil {
		return nil, err
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

	err = p.payer.SendTransaction(ctx, txLog, rawTx)
	return tx, err
}

func (p *Pipeline) getStorjPrice(ctx context.Context) (decimal.Decimal, error) {
	storjQuote, err := p.quoter.GetQuote(ctx, coinmarketcap.STORJ)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return storjQuote.Price, nil
}

func sleepFor(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
