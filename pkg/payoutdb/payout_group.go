package payoutdb

import (
	"context"
	"database/sql"
	"errors"

	"github.com/zeebo/errs"
)

const (
	// unfinishedCondition matches payout groups that still need to be paid out,
	// i.e. those that have neither been confirmed on chain nor skipped. The
	// status is NULL for a group that has never been attempted and the empty
	// string for a group that has been attempted at least once (the status is
	// cleared right before a transaction is created for it), so both have to be
	// treated as unfinished.
	unfinishedCondition = `
		payout_group.final_tx_hash IS NULL
	AND
		ifnull(payout_group.status, '') == ''
`
	// unattachedCondition additionally excludes the unfinished groups that
	// already have a transaction in flight.
	unattachedCondition = `
	AND
		payout_group.id NOT IN (SELECT payout_group_id FROM tx WHERE state == 'pending')
`

	unfinishedUnattachedConditional = `
	WHERE` + unfinishedCondition + unattachedCondition
)

func (db *DB) FirstUnfinishedUnattachedPayoutGroup(ctx context.Context) (*PayoutGroup, error) {
	// Unfortunately DBX doesn't support the following query. We could select
	// all the fields from payout_group but unfortunately that makes us brittle
	// against field changes, so instead, just grab up the primary key and then
	// issue individual select.

	stmt := `SELECT pk FROM payout_group` + unfinishedUnattachedConditional
	var pk int64
	if err := db.DB.QueryRow(stmt).Scan(&pk); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

// CountUnfinishedPayoutGroup returns the number of payout groups that have not
// been paid out yet, including the ones that already have a transaction in
// flight.
func (db *DB) CountUnfinishedPayoutGroup(ctx context.Context) (int64, error) {
	stmt := `SELECT COUNT(pk) FROM payout_group
	WHERE` + unfinishedCondition

	var count int64
	if err := db.DB.QueryRow(stmt).Scan(&count); err != nil {
		return 0, errs.Wrap(err)
	}

	return count, nil
}

// AllUnfinishedPayout returns the payouts belonging to the payout groups that
// have not been paid out yet.
func (db *DB) AllUnfinishedPayout(ctx context.Context) (_ []*Payout, err error) {
	stmt := `
		SELECT payout.pk, payout.created_at, payout.csv_line, payout.payee, payout.usd, payout.payout_group_id, payout.mandatory
		FROM payout
		JOIN payout_group ON payout.payout_group_id = payout_group.id
		WHERE` + unfinishedCondition

	rows, err := db.DB.QueryContext(ctx, stmt)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	defer func() { err = errs.Combine(err, errs.Wrap(rows.Close())) }()

	var payouts []*Payout
	for rows.Next() {
		payout := &Payout{}
		if err := rows.Scan(&payout.Pk, &payout.CreatedAt, &payout.CsvLine, &payout.Payee, &payout.Usd, &payout.PayoutGroupId, &payout.Mandatory); err != nil {
			return nil, errs.Wrap(err)
		}
		payouts = append(payouts, payout)
	}
	if err := rows.Err(); err != nil {
		return nil, errs.Wrap(err)
	}
	return payouts, nil
}
