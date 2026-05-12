package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/zeebo/clingy"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"

	"storj.io/crypto-batch-payment/pkg/config"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/payouts"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

type cmdRun struct {
	config           string
	skipConfirmation bool
	drain            bool
	pretend          bool
	retrySkipped     bool
}

func (cmd *cmdRun) Setup(params clingy.Parameters) {
	cmd.config = stringFlag(params, "config", "The configuration file", "./config.toml")
	cmd.skipConfirmation = toggleFlag(params, "skip-confirmation", "Run the payouts without asking for confirmation", false)
	cmd.drain = toggleFlag(params, "drain", "drain existing transactions only", false)
	cmd.pretend = toggleFlag(params, "pretend", "pretend to issue the payouts", false)
	cmd.retrySkipped = toggleFlag(params, "retry-skipped", "retry skipped transactions", false)
}

func (cmd *cmdRun) Execute(ctx context.Context) error {
	cfg, err := config.Load(cmd.config)
	if err != nil {
		var mfe *config.MissingFieldsError
		if errors.As(err, &mfe) {
			return fmt.Errorf("unable to load config:\n%s", mfe.String())
		}
		return fmt.Errorf("unable to load config: %w", err)
	}

	quoter, err := cfg.CoinMarketCap.NewQuoter()
	if err != nil {
		return fmt.Errorf("unable to init coin market cap quoter: %w", err)
	}

	payers, err := cfg.NewPayers(ctx)
	if err != nil {
		return fmt.Errorf("failed to init payers: %w", err)
	}
	defer payers.Close()

	dbs, err := loadDBs(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = dbs.Close() }()

	type payoutRun struct {
		db    *pipelinedb.DB
		payer payer.Payer
	}

	var runs []payoutRun
	dbOrder := maps.Keys(dbs)
	slices.Sort(dbOrder)
	for _, payerType := range dbOrder {
		db := dbs[payerType]

		payer, ok := payers[payerType]
		if !ok {
			return fmt.Errorf("no payer configured for %q payouts database", payerType)
		}

		if cmd.pretend {
			payer = &pretendPayer{Payer: payer}
		}

		runs = append(runs, payoutRun{db: db, payer: payer})
	}

	log, err := openLog(".")
	if err != nil {
		return fmt.Errorf("failed to open log: %w", err)
	}

	promptConfirm := promptConfirm
	if cmd.skipConfirmation {
		promptConfirm = func(label string) error {
			fmt.Printf("Skipping confirmation to %s!\n", label)
			return nil
		}
	}

	payoutsCfg := payouts.Config{
		Quoter:              quoter,
		PipelineLimit:       cfg.Pipeline.DepthLimit,
		TxDelay:             time.Duration(cfg.Pipeline.TxDelay),
		RetrySkipped:        cmd.retrySkipped,
		Drain:               cmd.drain,
		PromptConfirm:       promptConfirm,
		ThresholdDivisor:    cfg.Pipeline.ThresholdDivisor,
		MaxFeeTolerationUSD: cfg.Pipeline.MaxFeeTolerationUSD,
	}

	for _, run := range runs {
		if err := payouts.Preview(ctx, payoutsCfg, run.db, run.payer); err != nil {
			return err
		}
		fmt.Println()
	}

	for _, run := range runs {
		log := log.With(zap.Stringer("payer", run.payer))
		if err := payouts.Run(ctx, log, payoutsCfg, run.db, run.payer); err != nil {
			return err
		}
	}

	return nil
}

type pretendPayer struct {
	config.Payer
	nonce uint64
}

// NextNonce queries chain for the next available nonce value.
func (p *pretendPayer) NextNonce(ctx context.Context) (uint64, error) {
	//	if p.nonce == 0 {
	//		var err error
	//		p.nonce, err = p.Payer.NextNonce(ctx)
	//		return p.nonce, err
	//	}
	p.nonce++
	return p.nonce, nil
}

// GetETHBalance returns the ETH balance (in WEI).
func (pretendPayer) GetETHBalance(ctx context.Context) (*big.Int, error) {
	balance, _ := new(big.Int).SetString("10_000_000_000_000_000_000", 0)
	return balance, nil
}

// GetTokenBalance returns with the available token balance in real value (with decimals).
func (pretendPayer) GetTokenBalance(ctx context.Context) (*big.Int, error) {
	balance, _ := new(big.Int).SetString("1_000_000_000_000_000", 0)
	return balance, nil
}

// // CreateRawTransaction creates the chain transaction which will be persisted to the db.
func (pretendPayer) CreateRawTransaction(ctx context.Context, log *zap.Logger, params payer.TransactionParams) (payer.Transaction, common.Address, error) {
	var hash common.Hash
	var from common.Address
	binary.BigEndian.PutUint64(from[:], params.Nonce)
	binary.BigEndian.PutUint64(hash[:], params.Nonce)

	return payer.Transaction{
		Hash:               hash.Hex(),
		Nonce:              params.Nonce,
		EstimatedGasLimit:  50000,
		EstimatedGasFeeCap: big.NewInt(1000000000), // 1gwei
	}, from, nil
}

func (pretendPayer) SendTransaction(ctx context.Context, log *zap.Logger, tx payer.Transaction) error {
	return nil
}

func (pretendPayer) CheckNonceGroup(ctx context.Context, log *zap.Logger, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	statuses := make([]*pipelinedb.TxStatus, 0, len(nonceGroup.Txs))
	for _, tx := range nonceGroup.Txs {
		statuses = append(statuses, &pipelinedb.TxStatus{
			Hash:  tx.Hash,
			State: pipelinedb.TxConfirmed,
			Receipt: &types.Receipt{
				TxHash: common.HexToHash(tx.Hash),
				Logs:   []*types.Log{},
			},
		})
	}
	return pipelinedb.TxConfirmed, statuses, nil
}

func (pretendPayer) PrintEstimate(ctx context.Context, remaining int64) error {
	return nil
}
