package payoutdb

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPayoutGroupSelection pins down which payout groups the pipeline picks up.
// Getting this wrong means either dropping a payee or, worse, paying one twice,
// so assert the selection over every combination of status, final tx hash and
// transaction state.
func TestPayoutGroupSelection(t *testing.T) {
	ctx := t.Context()

	db, err := Open("sqlite3", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	defer func() { assert.NoError(t, db.Close()) }()

	_, err = db.Exec(db.Schema())
	require.NoError(t, err)

	type group struct {
		id       int64
		status   interface{}
		hash     interface{}
		txState  string
		selected bool
	}

	var groups []group
	var id int64
	for _, status := range []interface{}{nil, "", "skipped", "complete"} {
		for _, hash := range []interface{}{nil, "0xdeadbeef"} {
			for _, txState := range []string{"", "pending", "dropped", "failed", "confirmed"} {
				id++
				groups = append(groups, group{
					id:      id,
					status:  status,
					hash:    hash,
					txState: txState,
					// A group is up for (re)sending when it has not been
					// confirmed on chain (no final tx hash), has not been
					// skipped or completed (NULL or empty status) and has no
					// transaction in flight.
					selected: hash == nil && (status == nil || status == "") && txState != "pending",
				})
			}
		}
	}

	for _, g := range groups {
		_, err := db.Exec(`INSERT INTO payout_group (created_at, updated_at, id, final_tx_hash, status) VALUES (0, 0, ?, ?, ?)`,
			g.id, g.hash, g.status)
		require.NoError(t, err)
		if g.txState == "" {
			continue
		}
		_, err = db.Exec(`INSERT INTO tx (created_at, updated_at, hash, owner, spender, nonce, estimated_gas_price, storj_price, storj_tokens, payout_group_id, raw, state)
			VALUES (0, 0, ?, 'owner', 'spender', ?, '0', '0', '0', ?, '', ?)`,
			fmt.Sprintf("hash-%d", g.id), g.id, g.id, g.txState)
		require.NoError(t, err)
	}

	var wantUnattached, wantUnfinished int64
	for _, g := range groups {
		if g.selected {
			wantUnattached++
		}
		if g.hash == nil && (g.status == nil || g.status == "") {
			wantUnfinished++
		}
	}

	gotUnattached, err := db.CountUnfinishedUnattachedPayoutGroup(ctx)
	require.NoError(t, err)
	assert.Equal(t, wantUnattached, gotUnattached, "payout groups the pipeline will send")

	gotUnfinished, err := db.CountUnfinishedPayoutGroup(ctx)
	require.NoError(t, err)
	assert.Equal(t, wantUnfinished, gotUnfinished, "payout groups still owed")

	// The first group handed to the pipeline must be one of the selected ones.
	first, err := db.FirstUnfinishedUnattachedPayoutGroup(ctx)
	require.NoError(t, err)
	require.NotNil(t, first)
	for _, g := range groups {
		if g.id == first.Id {
			assert.True(t, g.selected, "payout group %d must not be sent", g.id)
		}
	}
}
