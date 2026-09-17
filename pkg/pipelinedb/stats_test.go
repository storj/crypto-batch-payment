package pipelinedb

import (
	"context"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStatsPendingAfterFailedTransaction asserts that payout groups whose
// transaction did not make it on chain are still reported as pending. Their
// status is cleared (i.e. set to the empty string, not NULL) before the
// transaction is created, and they used to disappear from the stats while the
// pipeline happily kept picking them up.
func TestStatsPendingAfterFailedTransaction(t *testing.T) {
	ctx := context.Background()

	db, err := NewDB(ctx, filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer func() { assert.NoError(t, db.Close()) }()

	// 1: never attempted
	// 2: attempted, transaction dropped
	// 3: attempted, transaction still pending
	// 4: skipped because it was below the threshold
	// 5: paid out
	for id := int64(1); id <= 5; id++ {
		require.NoError(t, db.CreatePayoutGroup(ctx, id, []*Payout{{
			CSVLine: int(id),
			Payee:   common.BigToAddress(big.NewInt(id)),
			USD:     decimal.NewFromInt(id),
		}}))
	}

	createTx := func(t *testing.T, payoutGroupID int64, state TxState) {
		hash := common.BigToHash(big.NewInt(payoutGroupID)).String()
		require.NoError(t, db.SetPayoutGroupStatus(ctx, payoutGroupID, ""))
		_, err := db.CreateTransaction(ctx, Transaction{
			Hash:          hash,
			Nonce:         uint64(payoutGroupID),
			PayoutGroupID: payoutGroupID,
			StorjPrice:    decimal.NewFromInt(1),
			StorjTokens:   big.NewInt(1),
		})
		require.NoError(t, err)
		if state != TxPending {
			require.NoError(t, db.UpdateTransactionState(ctx, hash, state))
		}
	}

	createTx(t, 2, TxDropped)
	createTx(t, 3, TxPending)
	require.NoError(t, db.SetPayoutGroupStatus(ctx, 4, PayoutGroupSkipped))
	createTx(t, 5, TxConfirmed)
	require.NoError(t, db.MarkPayoutGroupComplete(ctx, 5, common.BigToHash(big.NewInt(5)).String(), nil))

	stats, err := db.Stats(ctx, decimal.Zero)
	require.NoError(t, err)

	assert.Equal(t, int64(5), stats.TotalPayoutGroups)
	assert.Equal(t, int64(1), stats.SkippedPayoutGroups)

	// Groups 1, 2 and 3 still owe their payee.
	assert.Equal(t, int64(3), stats.PendingPayoutGroups)
	// Groups 1 and 2 are sent as soon as the pipeline starts, group 3 already
	// has a transaction in flight.
	assert.Equal(t, int64(2), stats.UnstartedPayoutGroups)
	assert.Equal(t, int64(1), stats.InFlightPayoutGroups)

	assert.Equal(t, int64(3), stats.PendingPayouts)
	assert.Equal(t, "6", stats.PendingUSD.String())

	// Sanity check that the stats agree with what the pipeline will do.
	unstarted, err := db.CountUnfinishedUnattachedPayoutGroup(ctx)
	require.NoError(t, err)
	assert.Equal(t, stats.UnstartedPayoutGroups, unstarted)
}
