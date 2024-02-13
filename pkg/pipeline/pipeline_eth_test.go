package pipeline

import (
	"context"
	"errors"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	batchpayment "storj.io/crypto-batch-payment/pkg"

	"github.com/ethereum/go-ethereum/core/types"

	"storj.io/crypto-batch-payment/pkg/eth"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/contract"
	"storj.io/crypto-batch-payment/pkg/ethtest"
)

const (
	initialStorj int64 = 1e11
	startingETH  int64 = 1e9
)

var (
	// deployer is the account that deploys the STORJ contract. It grants
	// an initial balance to owner.
	deployer = ethtest.NewAccount()

	// owner is the account that STORJ is drawn from. It is also the account
	// that transactions are paid from (i.e. gas) when using transfer() flow.
	owner = ethtest.NewAccount()

	// spender is the account that transactions are paid from (i.e. gas) when
	// using a transferFrom() flow.
	spender = ethtest.NewAccount()

	alice = ethtest.NewAccount()
	bob   = ethtest.NewAccount()
	chuck = ethtest.NewAccount()
	dave  = ethtest.NewAccount()
	eve   = ethtest.NewAccount()
)

func TestPipelineLotsOfTransactions(t *testing.T) {
	test := NewPipelineTest(t, WithLimit(16))

	payees := []common.Address{
		alice.Address,
		bob.Address,
		chuck.Address,
		dave.Address,
		eve.Address,
	}

	var payouts []*pipelinedb.Payout
	for i := int64(0); i < 200; i++ {
		payout := &pipelinedb.Payout{
			Payee: payees[i%5],
			USD:   decimal.New(i%5+1, 0),
		}
		payouts = append(payouts, payout)
	}
	test.InitializePayoutGroups(payouts)

	test.SetStorjPrice("10.00")
	test.ProcessPayouts(func(step int, _ []*pipelinedb.NonceGroup, _ func()) (bool, error) {
		test.commit()
		// There are 12 1/2 batches of 16 in 200 payouts. Two steps per batch
		// (one to send, one to confirm), so expect the last callback to be on
		// the step 26 (27th step in zero-based indexing where the first step
		// is the initial callout with the empty pipeline)
		return step >= 26, nil
	})
}

func TestPipelineLimits(t *testing.T) {
	test := NewPipelineTest(t, WithLimit(2))

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("1.00"),
		},
		{
			Payee: bob.Address,
			USD:   decimal.RequireFromString("2.00"),
		},
		{
			Payee: chuck.Address,
			USD:   decimal.RequireFromString("3.00"),
		},
		{
			Payee: dave.Address,
			USD:   decimal.RequireFromString("4.00"),
		},
		{
			Payee: eve.Address,
			USD:   decimal.RequireFromString("5.00"),
		},
	})

	test.SetStorjPrice("1.00")

	var (
		tx1Hash string
		tx2Hash string
		tx3Hash string
		tx4Hash string
		tx5Hash string
	)

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			// Pipeline just started with no existing nonce groups
			test.R.Empty(pipeline)
			return false, nil
		case 1:
			test.R.Len(pipeline, 2)

			// Pipeline slot 0 should be nonce 0 with a pending transaction
			// (tx1) for payout group 1.
			test.ValidatePipelineSlot(pipeline[0], 0, 1, pipelinedb.TxPending)
			tx1Hash = pipeline[0].Txs[0].Hash
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(1))

			// Pipeline slot 1 should be nonce 1 with a pending transaction
			// (tx2) for payout group 2.
			test.ValidatePipelineSlot(pipeline[1], 1, 2, pipelinedb.TxPending)
			tx2Hash = pipeline[1].Txs[0].Hash
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(2))

			// Commit tx1
			test.commit(tx1Hash)
			return false, nil
		case 2:
			test.R.Len(pipeline, 2)

			// Pipeline slot 0 should be nonce 0 with no transactions, since
			// tx1 was confirmed. Payout group 1 final hash should be tx1.
			test.ValidatePipelineSlot(pipeline[0], 0, 1)
			test.R.Equal(tx1Hash, test.FetchPayoutGroupFinalTxHash(1).String())
			test.R.Equal(pipelinedb.TxConfirmed, test.FetchTransactionState(tx1Hash))

			// Pipeline slot 1 should be unchanged.
			test.ValidatePipelineSlot(pipeline[1], 1, 2, pipelinedb.TxPending)
			test.R.Equal(tx2Hash, pipeline[1].Txs[0].Hash)
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(2))

			// Commit tx2
			test.commit(tx2Hash)
			return false, nil
		case 3:
			test.R.Len(pipeline, 2)

			// Pipeline slot 0 should now be nonce 1 with no transactions, since
			// tx2 was confirmed. Payout group 2 final hash should be tx2.
			test.ValidatePipelineSlot(pipeline[0], 1, 2)
			test.R.Equal(tx2Hash, test.FetchPayoutGroupFinalTxHash(2).String())
			test.R.Equal(pipelinedb.TxConfirmed, test.FetchTransactionState(tx2Hash))

			// Pipeline slot 1 should be nonce 2 with a pending transaction
			// (tx3) for payout group 3.
			test.ValidatePipelineSlot(pipeline[1], 2, 3, pipelinedb.TxPending)
			tx3Hash = pipeline[1].Txs[0].Hash
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(3))

			return false, nil
		case 4:
			test.R.Len(pipeline, 2)

			// Pipeline slot 0 should now be nonce 2, which is unchanged.
			test.ValidatePipelineSlot(pipeline[0], 2, 3, pipelinedb.TxPending)
			test.R.Equal(tx3Hash, pipeline[0].Txs[0].Hash)
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(3))

			// Pipeline slot 1 should be nonce 3 with a pending transaction
			// (tx4) for payout group 4.
			test.ValidatePipelineSlot(pipeline[1], 3, 4, pipelinedb.TxPending)
			tx4Hash = pipeline[1].Txs[0].Hash
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(4))

			return false, nil
		case 5:
			test.R.Len(pipeline, 2)

			// Assert that pipeline slot 0 remains unchanged
			test.ValidatePipelineSlot(pipeline[0], 2, 3, pipelinedb.TxPending)
			test.R.Equal(tx3Hash, pipeline[0].Txs[0].Hash)
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(3))

			// Assert that pipeline slot 1 remains unchanged
			test.ValidatePipelineSlot(pipeline[1], 3, 4, pipelinedb.TxPending)
			test.R.Equal(tx4Hash, pipeline[1].Txs[0].Hash)
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(4))

			// Commit both tx3 and tx4
			test.commit(tx3Hash, tx4Hash)
			return false, nil
		case 6:
			test.R.Len(pipeline, 2)

			// Pipeline slot 0 should be nonce 2 with no transactions, since
			// tx3 was confirmed. Payout group 3 final hash should be tx3.
			test.ValidatePipelineSlot(pipeline[0], 2, 3)
			test.R.Equal(tx3Hash, test.FetchPayoutGroupFinalTxHash(3).String())
			test.R.Equal(pipelinedb.TxConfirmed, test.FetchTransactionState(tx3Hash))

			// Pipeline slot 1 should be nonce 3 with no transactions, since
			// tx4 was confirmed. Payout group 4 final hash should be tx4.
			test.ValidatePipelineSlot(pipeline[1], 3, 4)
			test.R.Equal(tx4Hash, test.FetchPayoutGroupFinalTxHash(4).String())
			test.R.Equal(pipelinedb.TxConfirmed, test.FetchTransactionState(tx4Hash))

			return false, nil
		case 7:
			test.R.Len(pipeline, 1)

			// Pipeline slot 0 should be nonce 4 with a pending transaction
			// (tx5) for payout group 5.
			test.ValidatePipelineSlot(pipeline[0], 4, 5, pipelinedb.TxPending)
			tx5Hash = pipeline[0].Txs[0].Hash
			test.R.Nil(test.FetchPayoutGroupFinalTxHash(5))

			test.commit(tx5Hash)
			return false, nil
		case 8:
			test.R.Len(pipeline, 1)

			// Pipeline slot 0 should be nonce 4 with no transactions, since
			// tx5 was confirmed. Payout group 5 final hash should be tx5.
			test.ValidatePipelineSlot(pipeline[0], 4, 5)
			test.R.Equal(tx5Hash, test.FetchPayoutGroupFinalTxHash(5).String())
			test.R.Equal(pipelinedb.TxConfirmed, test.FetchTransactionState(tx5Hash))

			// This should be the last step.
			return true, nil
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.RequireEqualBig(big.NewInt(1e8), test.STORJBalance(alice.Address))
	test.RequireEqualBig(big.NewInt(2e8), test.STORJBalance(bob.Address))
	test.RequireEqualBig(big.NewInt(3e8), test.STORJBalance(chuck.Address))
	test.RequireEqualBig(big.NewInt(4e8), test.STORJBalance(dave.Address))
	test.RequireEqualBig(big.NewInt(5e8), test.STORJBalance(eve.Address))
}

func TestPipelineResumption(t *testing.T) {
	test := NewPipelineTest(t, WithLimit(2))

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("1.00"),
		},
		{
			Payee: bob.Address,
			USD:   decimal.RequireFromString("2.00"),
		},
	})

	test.SetStorjPrice("1.00")

	var (
		tx1Hash string
		tx2Hash string
	)

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			// Pipeline just started with no existing nonce groups
			test.R.Empty(pipeline)
			return false, nil
		case 1:
			test.R.Len(pipeline, 2)

			// Assert nonce group 1 is tracking a pending transaction
			test.R.Equal(uint64(0), pipeline[0].Nonce)
			test.R.Len(pipeline[0].Txs, 1)
			tx1Hash = pipeline[0].Txs[0].Hash

			// Assert nonce group 2 is tracking a pending transaction
			test.R.Equal(uint64(1), pipeline[1].Nonce)
			test.R.Len(pipeline[1].Txs, 1)
			tx2Hash = pipeline[1].Txs[0].Hash

			// Cancel processing
			cancel()
			return true, context.Canceled
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			test.R.Len(pipeline, 2)

			// Assert nonce group 1 is tracking the same pending transaction
			test.R.Equal(uint64(0), pipeline[0].Nonce)
			test.R.Len(pipeline[0].Txs, 1)
			test.R.Equal(tx1Hash, pipeline[0].Txs[0].Hash)

			// Assert nonce group 2 is tracking the same pending transaction
			test.R.Equal(uint64(1), pipeline[1].Nonce)
			test.R.Len(pipeline[1].Txs, 1)
			test.R.Equal(tx2Hash, pipeline[1].Txs[0].Hash)

			test.commit(tx1Hash)

			// Cancel processing
			cancel()
			return true, context.Canceled
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			test.R.Len(pipeline, 2)

			// Assert nonce group 1 transaction has been confirmed
			test.R.Equal(uint64(0), pipeline[0].Nonce)
			test.R.Empty(pipeline[0].Txs, 0)

			// Assert nonce group 2 is tracking the same pending transaction
			test.R.Equal(uint64(1), pipeline[1].Nonce)
			test.R.Len(pipeline[1].Txs, 1)
			test.R.Equal(tx2Hash, pipeline[1].Txs[0].Hash)

			test.commit(tx2Hash)

			// Cancel processing
			cancel()
			return true, context.Canceled
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			test.R.Len(pipeline, 1)

			// Assert nonce group 1 transaction has been confirmed
			test.R.Equal(uint64(1), pipeline[0].Nonce)
			test.R.Empty(pipeline[0].Txs, 0)

			return true, nil
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.RequireEqualBig(big.NewInt(1e8), test.STORJBalance(alice.Address))
	test.RequireEqualBig(big.NewInt(2e8), test.STORJBalance(bob.Address))
}

func TestPipelineTxFailure(t *testing.T) {
	test := NewPipelineTest(t, WithSpender(spender))

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("1.00"),
		},
	})
	test.SetStorjPrice("1.00")

	// This must be greater than 1e8 otherwise for some reason setting the approved
	// amount to zero inside of the payouts process fails.
	test.Approve(owner, spender, big.NewInt(1e8+1))

	var (
		txHash1st string
		txHash2nd string
	)

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			// Pipeline just started with no existing nonce groups
			test.R.Empty(pipeline)
			return false, nil
		case 1:
			test.R.Len(pipeline, 1)

			test.R.Equal(uint64(0), pipeline[0].Nonce)
			test.R.Len(pipeline[0].Txs, 1)
			txHash1st = pipeline[0].Txs[0].Hash

			// Before committing the transaction, remove the spender allowance
			// so that the transaction will fail.
			test.Approve(owner, spender, zero)

			// Commit the transaction. This transaction should fail because
			// the spender account has too little allowance
			test.commit(txHash1st)

			return true, errors.New("One or more transactions failed, possibly due to insufficient balances")
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	// The original transaction should be marked as failed
	test.R.Equal(pipelinedb.TxFailed, test.FetchTransactionState(txHash1st))

	// Re-approve the spender to send one token
	test.Approve(owner, spender, big.NewInt(1e8))

	// Resume the processing
	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			// Pipeline just started with no existing nonce groups
			test.R.Empty(pipeline)
			return false, nil
		case 1:
			test.R.Len(pipeline, 1)

			// Assert that another transaction was sent under a new nonce
			test.R.Equal(uint64(1), pipeline[0].Nonce)
			test.R.Len(pipeline[0].Txs, 1)
			txHash2nd = pipeline[0].Txs[0].Hash
			test.R.Equal(pipelinedb.TxPending, test.FetchTransactionState(txHash2nd))

			// Commit tx2
			test.commit(txHash2nd)
			return false, nil
		case 2:
			test.R.Len(pipeline, 1)

			// Assert that tx2 confirmed state has been recorded and that that
			// the it has been removed from the nonce group. Assert that the
			// payout group final transaction hash has been recorded.
			test.R.Empty(pipeline[0].Txs)
			test.R.Equal(pipelinedb.TxConfirmed, test.FetchTransactionState(txHash2nd))
			test.R.Equal(txHash2nd, test.FetchPayoutGroupFinalTxHash(1).String())

			return true, nil
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.RequireEqualBig(big.NewInt(1e8), test.STORJBalance(alice.Address))
}

func TestPipelineUsesLatestGasEstimateAndStorjPrice(t *testing.T) {
	test := NewPipelineTest(t)

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("1.00"),
		},
		{
			Payee: bob.Address,
			USD:   decimal.RequireFromString("2.00"),
		},
	})
	test.SetStorjPrice("1.00")

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			// Pipeline just started with no existing nonce groups
			test.R.Empty(pipeline)
			return false, nil
		case 1:
			test.R.Len(pipeline, 1)

			test.R.Len(pipeline[0].Txs, 1)
			tx := test.FetchTransaction(pipeline[0].Txs[0].Hash)
			test.R.Equal(decimal.RequireFromString("1.00").String(), tx.StorjPrice.String())
			test.R.Equal(big.NewInt(100000000), tx.StorjTokens)

			test.SetStorjPrice("10.00")

			test.commit(tx.Hash)
			return false, nil
		case 2:
			// This step just confirms the first transaction...
			return false, nil
		case 3:
			test.R.Len(pipeline, 1)

			test.R.Len(pipeline[0].Txs, 1)
			tx := test.FetchTransaction(pipeline[0].Txs[0].Hash)
			test.R.Equal(decimal.RequireFromString("10.00").String(), tx.StorjPrice.String())
			test.R.Equal(big.NewInt(20000000), tx.StorjTokens)

			test.commit(tx.Hash)
			return false, nil
		case 4:
			// This step just confirms the second transaction
			return true, nil
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.RequireEqualBig(big.NewInt(1e8), test.STORJBalance(alice.Address))
	test.RequireEqualBig(big.NewInt(2e7), test.STORJBalance(bob.Address))
}

func TestPipelineTransferFrom(t *testing.T) {
	gasTipCap := big.NewInt(1)
	test := NewPipelineTest(t, WithSpender(spender), WithGasTipCap(gasTipCap))

	test.Approve(owner, spender, big.NewInt(1e8))

	// Owner should have the initial STORJ balance.
	test.RequireEqualBig(big.NewInt(initialStorj), test.STORJBalance(owner.Address))

	// Spender should have an allowance for the approved amount but no actual
	// STORJ token.
	test.RequireEqualBig(big.NewInt(1e8), test.Allowance(owner, spender))
	test.RequireEqualBig(big.NewInt(0), test.STORJBalance(spender.Address))

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("1.00"),
		},
	})

	ctx := context.Background()

	test.SetStorjPrice("1.00")

	var tx *pipelinedb.Transaction

	client := test.Network.NewClient()
	defer client.Close()

	lastBalance, err := client.BalanceAt(ctx, spender.Address, nil)
	test.R.NoError(err)

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, _ func()) (bool, error) {
		switch step {
		case 0:
			// Pipeline just started with no existing nonce groups
			test.R.Empty(pipeline)
			return false, nil
		case 1:
			test.R.Len(pipeline, 1)
			test.R.Len(pipeline[0].Txs, 1)
			tx = test.FetchTransaction(pipeline[0].Txs[0].Hash)
			test.R.Equal(spender.Address, tx.Spender)
			test.commit()
			return false, nil
		case 2:

			txHash, err := batchpayment.HashFromString(tx.Hash)
			test.R.NoError(err)

			// check the raw transaction to assert EIP-1559 gas payment
			ethTx, _, err := client.TransactionByHash(ctx, txHash)
			test.R.NoError(err)

			// 0x02 --> EIP-1559 type transaction.
			test.R.Equal(types.DynamicFeeTxType, int(ethTx.Type()))

			receipt, err := client.TransactionReceipt(ctx, txHash)
			test.R.NoError(err)

			block, err := client.BlockByHash(ctx, receipt.BlockHash)
			test.R.NoError(err)

			newBalance, err := client.BalanceAt(ctx, spender.Address, nil)
			test.R.NoError(err)

			//make sure only gas * (baseFee + tip) is used
			gasCost := new(big.Int).Add(block.BaseFee(), gasTipCap)
			cost := new(big.Int).Mul(gasCost, big.NewInt(int64(receipt.GasUsed)))
			test.R.Equal(new(big.Int).Sub(lastBalance, cost), newBalance)

			return true, nil
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	// Alice should have the correct amount of tokens, paid from owner,
	// via spender.
	test.RequireEqualBig(big.NewInt(1e8), test.STORJBalance(alice.Address))
	test.RequireEqualBig(big.NewInt(initialStorj-1e8), test.STORJBalance(owner.Address))
	test.RequireEqualBig(big.NewInt(0), test.STORJBalance(spender.Address))

	// Spender should have exhausted the allowance.
	test.RequireEqualBig(big.NewInt(0), test.Allowance(owner, spender))
}

func TestPipelineChecksSTORJBalanceBeforeTransfer(t *testing.T) {
	test := NewPipelineTest(t)

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("100.00"),
		},
	})

	// Owner only has 100 storj tokens, it can't cover $100 worth at this
	// price.
	test.SetStorjPrice("0.01")

	pipeline := test.NewPipeline()
	err := pipeline.ProcessPayouts(context.Background())
	test.R.EqualError(err, "not enough STORJ balance to cover transfer (100000000000 < 1000000000000)")

	// No transactions should have been sent
	test.R.Equal(0, test.Network.PendingTransactionCount())

	// Alice should have nothing
	test.RequireEqualBig(big.NewInt(0), test.STORJBalance(alice.Address))

	// Owner should still have the initial balance
	test.RequireEqualBig(big.NewInt(initialStorj), test.STORJBalance(owner.Address))
}

func TestPipelineChecksSTORJAllowanceBeforeTransfer(t *testing.T) {
	test := NewPipelineTest(t, WithSpender(spender))

	// Approve for 1/10 of a storj token. With the 1.00 price, spender will
	// need to be approved for at least one token, which it won't have.
	test.Approve(owner, spender, big.NewInt(1e7))

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("1.00"),
		},
	})

	test.SetStorjPrice("1.00")

	pipeline := test.NewPipeline()
	err := pipeline.ProcessPayouts(context.Background())
	test.R.EqualError(err, "not enough STORJ allowance to cover transfer")

	// No transactions should have been sent
	test.R.Equal(0, test.Network.PendingTransactionCount())

	// Alice should have nothing
	test.RequireEqualBig(big.NewInt(0), test.STORJBalance(alice.Address))

	// Owner should still have the initial balance
	test.RequireEqualBig(big.NewInt(initialStorj), test.STORJBalance(owner.Address))

	// Spender should still have the allowance
	test.RequireEqualBig(big.NewInt(1e7), test.Allowance(owner, spender))
}

func TestPipelineVerySmallPayment(t *testing.T) {
	test := NewPipelineTest(t)

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("5e-05"),
		},
	})

	test.SetStorjPrice("0.14")

	test.ProcessPayouts(func(step int, pipeline []*pipelinedb.NonceGroup, cancel func()) (bool, error) {
		switch step {
		case 0:
			// Pipeline just started with no existing nonce groups
			test.R.Empty(pipeline)
			return false, nil
		case 1:
			test.R.Len(pipeline, 1)
			test.R.Len(pipeline[0].Txs, 1)
			test.commit(pipeline[0].Txs[0].Hash)
			return false, nil
		case 2:
			// in this step it was determined that the tx was confirmed
			return true, nil
		default:
			test.Fatalf("not expecting step %d", step)
			return false, nil
		}
	})

	test.RequireEqualBig(big.NewInt(35714), test.STORJBalance(alice.Address))
}

func TestPipelineTooSmallPayment(t *testing.T) {
	test := NewPipelineTest(t)

	test.InitializePayoutGroups([]*pipelinedb.Payout{
		{
			Payee: alice.Address,
			USD:   decimal.RequireFromString("5e-20"),
		},
	})

	test.SetStorjPrice("0.14")

	test.AssertProcessPayoutsFails("cannot transfer 0 tokens for payout group 1: must be more than zero")
}

/////////////////////////////////////////////////////////////////////////////
// Helpers
/////////////////////////////////////////////////////////////////////////////

type PipelineTestOption func(*PipelineTest)

func WithLimit(limit int) PipelineTestOption {
	return func(c *PipelineTest) {
		c.limit = limit
	}
}

func WithSpender(spender *ethtest.Account) PipelineTestOption {
	return func(c *PipelineTest) {
		c.spender = spender
	}
}

func WithGasTipCap(gasTipCap *big.Int) PipelineTestOption {
	return func(c *PipelineTest) {
		c.gasTipCap = gasTipCap
	}
}

type PipelineTest struct {
	*testing.T
	R *require.Assertions

	// config
	limit     int
	spender   *ethtest.Account
	gasTipCap *big.Int
	maxGas    *big.Int

	DB *pipelinedb.DB

	Quoter *ethtest.Quoter

	Network         *ethtest.Network
	Client          *ethclient.Client
	Contract        *contract.Token
	ContractAddress common.Address
}

func NewPipelineTest(t *testing.T, opts ...PipelineTestOption) *PipelineTest {
	dir := t.TempDir()

	test := &PipelineTest{
		T:         t,
		R:         require.New(t),
		limit:     1,
		gasTipCap: big.NewInt(1),
	}

	for _, opt := range opts {
		opt(test)
	}

	db, err := pipelinedb.NewDB(context.Background(), filepath.Join(dir, "payouts.db"))
	test.R.NoError(err)
	t.Cleanup(func() {
		_ = test.DB.Close()
	})

	test.DB = db
	test.Quoter = ethtest.NewQuoter()

	test.initNetwork()

	headBlock, err := test.Client.BlockByNumber(context.Background(), nil)
	require.NoError(t, err)

	// price can be increased by 12.5 % with every block when they are more than 50% full
	// 3x time multiplier is a safe choice to have
	test.maxGas = new(big.Int).Mul(headBlock.BaseFee(), big.NewInt(3))
	return test
}

func (test *PipelineTest) InitializePayoutGroups(payouts []*pipelinedb.Payout) {
	// Payout groups consist of only single payouts until we support
	// multitransfer.
	for i := range payouts {
		err := test.DB.CreatePayoutGroup(context.Background(), int64(i+1), payouts[i:i+1])
		test.R.NoError(err)
	}
}

func (test *PipelineTest) SetStorjPrice(s string) {
	test.Quoter.SetQuote(coinmarketcap.STORJ, &coinmarketcap.Quote{
		LastUpdated: time.Now(),
		Price:       decimal.RequireFromString(s),
	})
}

func (test *PipelineTest) initNetwork() {
	// Create a network, giving the owner and spender a little bit of cheese
	// to get things going.
	test.Network, test.Client = ethtest.NewNetworkAndClient(test,
		// grant deployer just enough to pay for the contract deployment
		ethtest.WithAccount(deployer, big.NewInt(1200000)),
		ethtest.WithAccount(owner, big.NewInt(startingETH)),
		ethtest.WithAccount(spender, big.NewInt(startingETH)),
	)

	// Deploy the ETH20 contract and commit
	opts, err := bind.NewKeyedTransactorWithChainID(deployer.Key, big.NewInt(1337))
	test.R.NoError(err)
	opts.Nonce = big.NewInt(0)
	opts.GasPrice = big.NewInt(1)
	opts.GasLimit = 1200000
	contractAddress, _, contract, err := contract.DeployToken(
		opts,
		test.Client,
		owner.Address,
		"Storj", "STORJ",
		big.NewInt(initialStorj),
		big.NewInt(8))
	test.R.NoError(err, "unable to deploy contract")
	test.commit()
	test.Contract = contract
	test.ContractAddress = contractAddress
	// 951225 is how much gas it actually takes
	test.R.Equal(big.NewInt(1200000-961413), test.ETHBalance(deployer.Address))
}

func (test *PipelineTest) NewPipeline() *Pipeline {
	return test.newPipeline(nil, time.Millisecond*50)
}

func (test *PipelineTest) newPipeline(stepInCh chan chan []*pipelinedb.NonceGroup, pollInterval time.Duration) *Pipeline {
	spenderKey := owner.Key
	if test.spender != nil {
		spenderKey = test.spender.Key
	}
	payer, err := eth.NewEthPayer(context.Background(),
		zaptest.NewLogger(test),
		test.Client,
		test.ContractAddress,
		owner.Address,
		spenderKey,
		big.NewInt(1337),
		test.gasTipCap,
		test.maxGas)
	test.R.NoError(err)
	pipeline, err := NewPipeline(payer, PipelineConfig{
		Log:          zaptest.NewLogger(test),
		Owner:        owner.Address,
		Quoter:       test.Quoter,
		DB:           test.DB,
		Limit:        test.limit,
		stepInCh:     stepInCh,
		pollInterval: pollInterval,
	})
	test.R.NoError(err)
	return pipeline
}

func (test *PipelineTest) ProcessPayouts(step func(int, []*pipelinedb.NonceGroup, func()) (bool, error)) {
	stepInCh := make(chan chan []*pipelinedb.NonceGroup)
	pipeline := test.newPipeline(stepInCh, time.Minute)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := pipeline.initPayout(ctx)
	test.R.NoError(err)
	for i := 0; ; i++ {

		expectedDone, expectedErr := step(i, pipeline.nonceGroups, cancel)
		test.R.NoError(err)

		if ctx.Err() != nil {
			break
		}

		done, err := pipeline.payoutStep(ctx)
		test.R.Equal(expectedDone, done)
		if expectedErr != nil {
			test.R.EqualError(err, expectedErr.Error())
		} else {
			test.R.NoError(err)
		}

		if done {
			break
		}

	}
}

func (test *PipelineTest) AssertProcessPayoutsFails(expectedErr string) {
	pipeline := test.newPipeline(nil, time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	err := pipeline.ProcessPayouts(ctx)
	test.R.EqualError(err, expectedErr)
}

func (test *PipelineTest) FetchTransaction(hashString string) *pipelinedb.Transaction {
	hash, err := batchpayment.HashFromString(hashString)
	test.R.NoError(err)
	tx, err := test.DB.FetchTransaction(context.Background(), hash)
	test.R.NoError(err)
	test.R.NotNil(tx)
	return tx
}

func (test *PipelineTest) FetchTransactionState(hashString string) pipelinedb.TxState {
	return test.FetchTransaction(hashString).State
}

func (test *PipelineTest) FetchPayoutGroup(payoutGroupID int64) *pipelinedb.PayoutGroup {
	tx, err := test.DB.FetchPayoutGroup(context.Background(), payoutGroupID)
	test.R.NoError(err)
	test.R.NotNil(tx)
	return tx
}

func (test *PipelineTest) FetchPayoutGroupFinalTxHash(payoutGroupID int64) *common.Hash {
	return test.FetchPayoutGroup(payoutGroupID).FinalTxHash
}

func (test *PipelineTest) ValidatePipelineSlot(nonceGroup *pipelinedb.NonceGroup, nonce uint64, payoutGroupID int64, txStates ...pipelinedb.TxState) {
	test.R.Equal(nonce, nonceGroup.Nonce, "unexpected nonce")
	test.R.Len(nonceGroup.Txs, len(txStates), "unexpected number of transactions")
	for i, txState := range txStates {
		tx := test.FetchTransaction(nonceGroup.Txs[i].Hash)
		test.R.Equal(txState, tx.State, "unexpected transaction state")
		test.R.Equal(nonce, tx.Nonce, "unexpected transaction nonce")
		test.R.Equal(payoutGroupID, tx.PayoutGroupID, "unexpected transaction payout group id")
	}
}

func (test *PipelineTest) ETHBalance(address common.Address) *big.Int {
	balance, err := test.Client.BalanceAt(context.Background(), address, nil)
	test.R.NoError(err)
	return balance
}

func (test *PipelineTest) STORJBalance(address common.Address) *big.Int {
	balance, err := test.Contract.BalanceOf(nil, address)
	test.R.NoError(err)
	return balance
}

func (test *PipelineTest) Approve(owner, spender *ethtest.Account, amount *big.Int) {
	opts, err := bind.NewKeyedTransactorWithChainID(owner.Key, big.NewInt(1337))
	opts.GasPrice = big.NewInt(1)
	opts.GasLimit = 100000
	test.R.NoError(err)
	tx, err := test.Contract.Approve(opts, spender.Address, amount)
	test.R.NoError(err)
	test.commit(tx.Hash().String())
	state, _, _, err := eth.GetTransactionInfo(context.Background(), test.Client, tx.Hash())
	test.R.NoError(err)
	test.R.Equal(pipelinedb.TxConfirmed, state)
}

func (test *PipelineTest) Allowance(owner, spender *ethtest.Account) *big.Int {
	allowance, err := test.Contract.Allowance(nil, owner.Address, spender.Address)
	test.R.NoError(err)
	return allowance
}

func (test *PipelineTest) RequireEqualBig(expected, actual *big.Int) {
	test.R.Equal(expected.String(), actual.String())
}

func (test *PipelineTest) LogNonceGroups(pipeline []pipelinedb.NonceGroup) {
	for i, nonceGroup := range pipeline {
		for k, tx := range nonceGroup.Txs {
			test.Logf("group=%d tx=%d hash=%s", i, k, tx.Hash)
		}
	}
}

func (test *PipelineTest) commit(hashStrings ...string) {
	var hashes []common.Hash
	for _, h := range hashStrings {
		hash, err := batchpayment.HashFromString(h)
		test.R.NoError(err)
		hashes = append(hashes, hash)
	}
	test.Network.Commit(hashes...)
}
