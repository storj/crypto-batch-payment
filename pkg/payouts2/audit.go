package payouts2

import (
	"context"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
	"golang.org/x/exp/maps"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/receipts"
)

type AuditSink interface {
	ReportStatusf(format string, args ...interface{})
	ReportWarnf(format string, args ...interface{})
	ReportErrorf(format string, args ...interface{})
}

type TransactionStats struct {
	Total           int64
	Confirmed       int64
	FalseConfirmed  int64
	Overpaid        int64
	Skipped         int64
	Unstarted       int64
	Pending         int64
	Failed          int64
	Dropped         int64
	Unknown         int64
	DoublePays      int64
	MismatchedState int64
	DoublePayStorj  big.Int
}

type DBStats struct {
	MissingDBs  int
	MissingCSVs int
	Mismatched  int
}

func AuditDBs(ctx context.Context, dbs map[payer.Type]*pipelinedb.DB, csvPaths []string, sink AuditSink) (*DBStats, error) {
	stats := new(DBStats)

	if len(dbs) == 0 {
		return nil, errors.New("no databases to audit; have you run 'init'?")
	}

	// Make sure each database has the same bonus multipler applied.
	bonusMultiplier, err := auditBonusMultiplier(ctx, maps.Values(dbs))
	if err != nil {
		sink.ReportErrorf("Invalid bonus multipler: %v", err)
	}

	csvPayoutsByType, err := loadCSVs(csvPaths, bonusMultiplier, &auditUI{sink: sink})
	if err != nil {
		return nil, err
	}

	// Determine which payouts type in the CSVs do not have a database
	for payerType := range csvPayoutsByType {
		if _, ok := dbs[payerType]; !ok {
			sink.ReportErrorf("No database for CSV type %q", payerType)
			stats.MissingDBs++
		}
	}

	// Audit each database
	for payerType, db := range dbs {
		csvPayouts, ok := csvPayoutsByType[payerType]
		if !ok {
			sink.ReportErrorf("No CSV for database type %q", payerType)
			stats.MissingCSVs++
			continue
		}

		dbPayouts, err := db.FetchPayouts(ctx)
		if err != nil {
			return nil, errs.Wrap(err)
		}

		stats.Mismatched += comparePayouts(payerType, csvPayouts, dbPayouts, sink)
	}

	return stats, nil
}

func AuditTransactions(ctx context.Context, payerType payer.Type, auditor payer.Auditor, db *pipelinedb.DB, sink AuditSink, receipts *receipts.Buffer) (*TransactionStats, error) {
	sink.ReportStatusf("Fetching payouts...")
	dbPayouts, err := db.FetchPayouts(ctx)
	if err != nil {
		return nil, err
	}

	stats := &TransactionStats{
		Total: int64(len(dbPayouts)),
	}

	// Confirm the status of each transaction to ensure we haven't accidentally
	// overpaid.
	sink.ReportStatusf("Confirming TX status...")
	txs, err := db.FetchTransactions(ctx)
	if err != nil {
		return nil, err
	}

	last := time.Now()
	for i, tx := range txs {
		which := i + 1
		now := time.Now()
		if which == len(txs) || now.Sub(last) > time.Second {
			last = now
			sink.ReportStatusf("Confirming TX status (%d/%d)...", which, len(txs))
		}
		state, err := auditor.CheckTransactionState(ctx, tx.Hash)
		if err != nil {
			return nil, err
		}

		if tx.State == state {
			continue
		}

		if tx.State == pipelinedb.TxDropped && state == pipelinedb.TxConfirmed {
			sink.ReportErrorf("Double pay for payout group %d (tokens=%s)", tx.PayoutGroupID, tx.StorjTokens)
			stats.DoublePays++
			stats.DoublePayStorj.Add(&stats.DoublePayStorj, tx.StorjTokens)
		} else {
			sink.ReportErrorf("TX state mismatch on hash %q (db=%q, node=%q)", tx.Hash, tx.State, state)
			stats.MismatchedState++
		}
	}

	// For each payout, ensure it belongs to a payout group with a confirmed
	// transaction. Reconfirm the transaction against the blockchain.
	sink.ReportStatusf("Checking payouts status...")
	payoutGroupStatus := make(map[int64]string)
	var payoutsConfirmed int64
	for _, dbPayout := range dbPayouts {
		if txHash, ok := payoutGroupStatus[dbPayout.PayoutGroupID]; ok {
			if txHash != "" {
				receipts.Emit(dbPayout.Payee, dbPayout.USD, txHash, payerType)
			}
			continue
		}
		// Mark the payout group status as done with no transaction. It will be
		// marked with the confirming transaction after passing the checks below.
		payoutGroupStatus[dbPayout.PayoutGroupID] = ""

		numPayouts, err := db.FetchPayoutGroupPayoutCount(ctx, dbPayout.PayoutGroupID)
		if err != nil {
			return nil, errs.Wrap(err)
		}

		status, err := db.FetchPayoutGroupStatus(ctx, dbPayout.PayoutGroupID)
		if err != nil {
			return nil, errs.Wrap(err)
		}
		if status == pipelinedb.PayoutGroupSkipped {
			stats.Skipped++
			continue
		}

		txs, err := db.FetchPayoutGroupTransactions(ctx, dbPayout.PayoutGroupID)
		if err != nil {
			return nil, errs.Wrap(err)
		}
		if len(txs) == 0 {
			sink.ReportErrorf("Payout (%s) of %s to %s has no attempted transactions",
				payerType, dbPayout.USD, dbPayout.Payee.String())
			stats.Unstarted += numPayouts
			continue
		}

		var pending []*pipelinedb.Transaction
		var dropped []*pipelinedb.Transaction
		var failed []*pipelinedb.Transaction
		var confirmed []*pipelinedb.Transaction
		for _, tx := range txs {
			switch tx.State {
			case pipelinedb.TxPending:
				pending = append(pending, tx)
			case pipelinedb.TxDropped:
				dropped = append(dropped, tx)
			case pipelinedb.TxFailed:
				failed = append(failed, tx)
			case pipelinedb.TxConfirmed:
				confirmed = append(confirmed, tx)
			default:
				sink.ReportErrorf("Unexpected tx state %q on %s", tx.State, tx.Hash)
			}
		}

		if len(confirmed) == 0 {
			sink.ReportErrorf("Payout of %s to %s has no confirmed transactions (pending=%d dropped=%d failed=%d)",
				dbPayout.USD, dbPayout.Payee.String(),
				len(pending), len(dropped), len(failed))
			switch {
			case len(pending) > 0:
				stats.Pending += numPayouts
			case len(failed) > 0:
				stats.Failed += numPayouts
			case len(dropped) > 0:
				stats.Dropped += numPayouts
			default:
				stats.Unknown += numPayouts
			}
			continue
		}

		var confirmedCount int
		for _, tx := range confirmed {
			state, err := auditor.CheckConfirmedTransactionState(ctx, tx.Hash)
			switch {
			case err != nil:
				sink.ReportErrorf("Failed to get receipt for transaction %s for payout of %s to %s",
					tx.Hash, dbPayout.USD, dbPayout.Payee.String())
			case state != pipelinedb.TxConfirmed:
				sink.ReportErrorf("Transaction %s was %s instead of confirmed for payout of %s to %s",
					tx.Hash, state, dbPayout.USD, dbPayout.Payee.String())
			default:
				confirmedCount++
			}
		}

		if confirmedCount > 0 {
			txHash := confirmed[0].Hash
			payoutGroupStatus[dbPayout.PayoutGroupID] = txHash
			receipts.Emit(dbPayout.Payee, dbPayout.USD, txHash, payerType)
			payoutsConfirmed += numPayouts
		}

		switch {
		case confirmedCount > 1:
			sink.ReportErrorf("Payout of %s to %s has more than one (%d) confirmed transactions recorded",
				dbPayout.USD, dbPayout.Payee.String(),
				len(confirmed))
			stats.Overpaid += numPayouts
		case confirmedCount == 0:
			stats.FalseConfirmed += numPayouts
		default:
			stats.Confirmed += numPayouts
		}
	}

	return stats, nil
}

func auditBonusMultiplier(ctx context.Context, dbs []*pipelinedb.DB) (decimal.Decimal, error) {
	if len(dbs) == 0 {
		return decimal.Decimal{}, nil
	}

	first, err := dbs[0].GetBonusMultiplier(ctx)
	if err != nil {
		return decimal.Decimal{}, err
	}

	for _, db := range dbs[1:] {
		other, err := db.GetBonusMultiplier(ctx)
		if err != nil {
			return decimal.Decimal{}, err
		}
		if !first.Equal(other) {
			return decimal.Decimal{}, errs.New("mismatched bonus multipler: expected %q but got %q", first, other)
		}
	}
	return first, nil
}

// comparePayouts compares the csv and db payouts contents and returns the number of mismatched payouts.
func comparePayouts(payerType payer.Type, csvPayouts, dbPayouts []*pipelinedb.Payout, sink AuditSink) int {
	csvPayoutsByPayee := make(map[common.Address]*pipelinedb.Payout)
	for _, csvPayout := range csvPayouts {
		if _, ok := csvPayoutsByPayee[csvPayout.Payee]; ok {
			// This would only happen if there was a bug loading payouts from CSV
			sink.ReportErrorf("Duplicate %s payee %s detected in CSV payouts", payerType, csvPayout.Payee)
		}
		csvPayoutsByPayee[csvPayout.Payee] = csvPayout
	}
	dbPayoutsByPayee := make(map[common.Address]*pipelinedb.Payout)
	for _, dbPayout := range dbPayouts {
		if _, ok := dbPayoutsByPayee[dbPayout.Payee]; ok {
			// This would only happen if there was a bug loading payouts from CSV
			sink.ReportErrorf("Duplicate %s CSV payee %s detected in database payouts", payerType, dbPayout.Payee)
		}
		dbPayoutsByPayee[dbPayout.Payee] = dbPayout
	}

	mismatched := map[common.Address]struct{}{}

	// Ensure each CSV payout is represented accurately in the DB
	sink.ReportStatusf("Reconciling %s CSV payout entries...", payerType)
	for _, csvPayout := range csvPayouts {
		dbPayout, ok := dbPayoutsByPayee[csvPayout.Payee]
		if !ok {
			sink.ReportErrorf("No %s payout for payee %s in database", payerType, csvPayout.Payee)
			mismatched[csvPayout.Payee] = struct{}{}
			continue
		}
		if !dbPayout.USD.Equal(csvPayout.USD) {
			sink.ReportErrorf("Amount mismatch for %s payee %s: csv=%q db=%q", payerType, csvPayout.Payee, csvPayout.USD, dbPayout.USD)
			mismatched[csvPayout.Payee] = struct{}{}
			continue
		}
	}

	// Ensure each DB payout is represented accurately in the CSV
	sink.ReportStatusf("Reconciling %s DB payout entries...", payerType)
	for _, dbPayout := range dbPayouts {
		if _, ok := csvPayoutsByPayee[dbPayout.Payee]; !ok {
			sink.ReportErrorf("No %s payout for payee %s in CSV", payerType, dbPayout.Payee)
			mismatched[dbPayout.Payee] = struct{}{}
		}
	}

	return len(mismatched)
}

type auditUI struct {
	sink AuditSink
}

func (a *auditUI) Started(evt StartedEvent) {
	a.sink.ReportStatusf("Auditing CSVs: %q", evt.CSVPaths)
}

func (a *auditUI) CSVLoaded(evt CSVLoadedEvent) {
	if evt.Err != nil {
		a.sink.ReportStatusf("Failed to load %q: %v", evt.CSVPath, evt.Err)
	} else {
		a.sink.ReportStatusf("Loaded %q (%d rows)", evt.CSVPath, evt.NumRows)
	}
}

func (a *auditUI) RowAggregated(_ RowAggregatedEvent) {}

func (a *auditUI) RowSkipped(_ RowSkippedEvent) {}

func (a *auditUI) RowsAggregated(_ RowsAggregatedEvent) {}

func (a *auditUI) CSVsLoaded(_ CSVsLoadedEvent) {}
