package pipeline

import (
	"context"
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"

	"storj.io/common/testcontext"
)

func Test_HappyPath(t *testing.T) {
	ctx := context.Background()

	db := createTestDB(ctx, t, []*pipelinedb.Payout{
		{
			Payee: common.HexToAddress("0x58408e92BD76B15b23531F5BA3a6253513748ecA"),
			USD:   decimal.New(1, 0),
		},
		{
			Payee: common.HexToAddress("0x69F195FC69072649183a0F7D5663c53EBD1cDeF0"),
			USD:   decimal.New(1, 0),
		},
		{
			Payee: common.HexToAddress("0xef6458a66605d05C0DAE84EFF844b0d0a7AAb506"),
			USD:   decimal.New(1, 0),
		},
	})
	defer db.Close()

	p, _ := createTestPipeline(ctx, t, db)

	err := p.initPayout(ctx)
	require.NoError(t, err)

	// first iteration
	done, err := p.payoutStep(ctx)
	require.NoError(t, err)
	require.False(t, done)
	assertPaymetGroupStatus(ctx, t, db, 0, pipelinedb.TxConfirmed)

	// second iteration
	done, err = p.payoutStep(ctx)
	require.NoError(t, err)
	require.True(t, done)
	assertPaymetGroupStatus(ctx, t, db, 0, pipelinedb.TxConfirmed)

}

func Test_TestIntermittentTxError(t *testing.T) {
	ctx := testcontext.New(t)

	db := createTestDB(ctx, t, []*pipelinedb.Payout{
		{
			Payee: common.HexToAddress("0x58408e92BD76B15b23531F5BA3a6253513748ecA"),
			USD:   decimal.New(1, 0),
		},
		{
			Payee: common.HexToAddress("0xd32E554823E3b08F80A8173FAcc2B6AD3502376F"),
			USD:   decimal.New(100, 0),
		},
	})
	defer db.Close()

	pipeline, testPayer := createTestPipeline(ctx, t, db)

	err := pipeline.initPayout(ctx)
	require.NoError(t, err)

	testPayer.sendTransactionHandler = txFailsWith(1)
	testPayer.checkNonceGroupHandler = statusFailsWith(1)

	// first iteration, should fail
	_, err = pipeline.payoutStep(ctx)
	require.Error(t, err)
	assertPaymetGroupStatus(ctx, t, db, 0, pipelinedb.TxPending)

	// restart, it's pending, intermittent error is fixed
	pipeline, _ = createTestPipeline(ctx, t, db)

	err = pipeline.initPayout(ctx)
	require.NoError(t, err)
	// we check the status during init
	assertPaymetGroupStatus(ctx, t, db, 0, pipelinedb.TxConfirmed)

	// second attempt, first iteration
	done, err := pipeline.payoutStep(ctx)
	require.NoError(t, err)
	require.True(t, done)
	assertPaymetGroupStatus(ctx, t, db, 0, pipelinedb.TxConfirmed)
	assertPaymetGroupStatus(ctx, t, db, 1, pipelinedb.TxConfirmed)

	// second attempt, second iteration
	// both confirmed, we break at the beginning of the loop
	done, err = pipeline.payoutStep(ctx)
	require.NoError(t, err)
	require.True(t, done)
	assertPaymetGroupStatus(ctx, t, db, 0, pipelinedb.TxConfirmed)
	assertPaymetGroupStatus(ctx, t, db, 1, pipelinedb.TxConfirmed)

}

func statusFailsWith(noncesToFail ...int) func(ctx context.Context, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	return func(ctx context.Context, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
		for _, i := range noncesToFail {
			if uint64(i) == nonceGroup.Nonce {
				return statusResult(pipelinedb.TxFailed, nonceGroup.Txs[0].Hash)
			}
		}
		return statusResult(pipelinedb.TxConfirmed, nonceGroup.Txs[0].Hash)
	}
}

func txFailsWith(noncesToFail ...int) func(ctx context.Context, tx payer.Transaction) error {
	return func(ctx context.Context, tx payer.Transaction) error {
		for _, i := range noncesToFail {
			if uint64(i) == tx.Nonce {
				return errs.New("")
			}
		}
		return nil
	}
}

func assertPaymetGroupStatus(ctx context.Context, t *testing.T, db *pipelinedb.DB, groupID int, state pipelinedb.TxState) {
	txs, err := db.FetchPayoutGroupTransactions(ctx, int64(groupID))
	require.NoError(t, err)
	require.Len(t, txs, 1)
	require.Equal(t, state, txs[0].State)
}

func createTestDB(ctx context.Context, t *testing.T, payouts []*pipelinedb.Payout) *pipelinedb.DB {
	dir, err := os.MkdirTemp("", "payouts-pipeline-")
	require.NoError(t, err)

	db, err := pipelinedb.NewDB(context.Background(), filepath.Join(dir, "payouts.db"))
	require.NoError(t, err)

	for i, p := range payouts {
		err = db.CreatePayoutGroup(ctx, int64(i), []*pipelinedb.Payout{p})
	}
	require.NoError(t, err)
	return db

}
func createTestPipeline(ctx context.Context, t *testing.T, db *pipelinedb.DB) (*Pipeline, *TestPayer) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	cfg := PipelineConfig{
		DB:  db,
		Log: logger,
		Quoter: &StaticQuoter{
			value: decimal.New(2, 0),
		},
	}

	testPayer := NewTestPayer()
	p, err := NewPipeline(testPayer, cfg)
	require.NoError(t, err)
	return p, testPayer
}

var _ payer.Payer = &TestPayer{}

type TestPayer struct {
	nextNonce              uint64
	sendTransactionHandler func(ctx context.Context, tx payer.Transaction) error
	checkNonceGroupHandler func(ctx context.Context, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error)
}

func NewTestPayer() *TestPayer {
	return &TestPayer{
		sendTransactionHandler: func(ctx context.Context, tx payer.Transaction) error {
			return nil
		},
		checkNonceGroupHandler: func(ctx context.Context, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
			return statusResult(pipelinedb.TxConfirmed, nonceGroup.Txs[0].Hash)
		},
	}
}

func (t *TestPayer) NextNonce(ctx context.Context) (uint64, error) {
	ret := t.nextNonce
	t.nextNonce++
	return ret, nil
}

func (t *TestPayer) IsPreconditionMet(ctx context.Context) (bool, error) {
	return true, nil
}

func (t *TestPayer) GetTokenBalance(ctx context.Context) (*big.Int, error) {
	return big.NewInt(10_000_00000000), nil
}

func (t *TestPayer) GetTokenDecimals(ctx context.Context) (int32, error) {
	return 8, nil
}

func (t *TestPayer) CreateRawTransaction(ctx context.Context, log *zap.Logger, payouts []*pipelinedb.Payout, nonce uint64, storjPrice decimal.Decimal) (tx payer.Transaction, from common.Address, err error) {
	hash := make([]byte, 32)
	_, err = rand.Read(hash)
	return payer.Transaction{
		Hash:  common.BytesToHash(hash).String(),
		Nonce: nonce,
		Raw:   make(map[string]string),
	}, common.HexToAddress("0x94F31A2f6522dbf0594bf9c37F124fB6EAC4d9cd"), err

}

func (t *TestPayer) SendTransaction(ctx context.Context, tx payer.Transaction) error {
	return t.sendTransactionHandler(ctx, tx)
}

func (t *TestPayer) CheckNonceGroup(ctx context.Context, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	return t.checkNonceGroupHandler(ctx, nonceGroup, checkOnly)
}

func (t *TestPayer) PrintEstimate(ctx context.Context, remaining int64) error {
	panic("implement me")
}

type StaticQuoter struct {
	value decimal.Decimal
}

func (s StaticQuoter) GetQuote(ctx context.Context, symbol coinmarketcap.Symbol) (*coinmarketcap.Quote, error) {
	return &coinmarketcap.Quote{
		Price:       s.value,
		LastUpdated: time.Now(),
	}, nil
}

func statusResult(status pipelinedb.TxState, hash string) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	return status, []*pipelinedb.TxStatus{
		{
			Hash:  hash,
			State: pipelinedb.TxConfirmed,
			Receipt: &types.Receipt{
				Logs: []*types.Log{},
			},
		},
	}, nil
}
