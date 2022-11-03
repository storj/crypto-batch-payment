package payoutdb

import (
	"context"
	"database/sql"

	"github.com/zeebo/errs"
)

const (
	unfinishedUnattachedConditional = `
		WHERE
			final_tx_hash IS NULL
		AND
			id NOT IN (SELECT payout_group_id FROM tx WHERE state == 'pending')
`
)

func (db *DB) FirstUnfinishedUnattachedPayoutGroup(ctx context.Context) (*PayoutGroup, error) {
	// Unfortunately DBX doesn't support the following query. We could select
	// all the fields from payout_group but unfortunately that makes us brittle
	// against field changes, so instead, just grab up the primary key and then
	// issue individual select.

	stmt := `SELECT pk FROM payout_group` + unfinishedUnattachedConditional
	var pk int64
	if err := db.DB.QueryRow(stmt).Scan(&pk); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errs.Wrap(err)
	}

	payoutGroup, err := db.Get_PayoutGroup_By_Pk(ctx, PayoutGroup_Pk(pk))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return payoutGroup, nil
}

func (db *DB) CountUnfinishedUnattachedPayoutGroup(ctx context.Context) (int64, error) {
	// Unfortunately DBX doesn't support the following query. We could select
	// all the fields from payout_group but unfortunately that makes us brittle
	// against field changes, so instead, just grab up the primary key and then
	// issue individual select.
	stmt := `SELECT COUNT(pk) FROM payout_group` + unfinishedUnattachedConditional

	var count int64
	if err := db.DB.QueryRow(stmt).Scan(&count); err != nil {
		return 0, errs.Wrap(err)
	}

	return count, nil
}
