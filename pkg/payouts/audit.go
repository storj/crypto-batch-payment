package payouts

import (
	"bytes"
	"context"
	stdcsv "encoding/csv"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeebo/errs"
	"storj.io/crypto-batch-payment/pkg/eth"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/zksync"

	"storj.io/crypto-batch-payment/pkg/csv"
)

type AuditSink interface {
	ReportStatus(format string, args ...interface{})
	ReportWarn(format string, args ...interface{})
	ReportError(format string, args ...interface{})
}

type AuditStats struct {
	Total          int64
	Confirmed      int64
	FalseConfirmed int64
	Overpaid       int64
	Unstarted      int64
	Pending        int64
	Failed         int64
	Dropped        int64
	Unknown        int64
	Mismatched     int64
	DoublePays     int64
	DoublePayStorj *big.Int
}

func Audit(ctx context.Context, dir string, csvPath string, payerType PayerType, nodeAddress string, chainID int, sink AuditSink, receiptsOut string, receiptsForce bool) (*AuditStats, error) {
	var auditor payer.Auditor
	switch payerType {
	case Eth, Polygon:
		client, err := ethclient.Dial(nodeAddress)
		if err != nil {
			return nil, errs.New("Failed to dial node %q: %v\n", nodeAddress, err)
		}
		defer client.Close()
		auditor, err = eth.NewEthAuditor(nodeAddress)
		if err != nil {
			return nil, err
		}
	case Sim:
		auditor = payer.NewSimAuditor()
	case ZkSync, ZkWithdraw:
		var err error
		auditor, err = zksync.NewAuditor(
			nodeAddress,
			chainID)
		if err != nil {
			return nil, err
		}
	default:
		return nil, errs.New("unsupported auditor type: %v", payerType)
	}

	// Load payouts from the CSV
	rows, err := csv.Load(csvPath)
	if err != nil {
		return nil, err
	}
	csvPayouts := PayoutsFromCSV(rows)

	receiptsBuf := new(bytes.Buffer)
	receiptsCSV := stdcsv.NewWriter(receiptsBuf)
	receiptsCSV.Write([]string{"wallet", "amount", "txhash", "mechanism"})

	// Load the database
	sink.ReportStatus("Loading database...")
	dbDir, err := dbDirFromCSVPath(dir, csvPath)
	if err != nil {
		return nil, err
	}
	db, err := pipelinedb.OpenDB(ctx, dbPathFromDir(dbDir), true)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	// Load payout rows
	sink.ReportStatus("Fetching payouts...")
	dbPayouts, err := db.FetchPayouts(ctx)
	if err != nil {
		return nil, err
	}

	stats := &AuditStats{
		Total:          int64(len(dbPayouts)),
		DoublePayStorj: new(big.Int),
	}

	csvPayoutsByLine := make(map[int]*pipelinedb.Payout)
	for _, csvPayout := range csvPayouts {
		if _, ok := csvPayoutsByLine[csvPayout.CSVLine]; ok {
			// This would only happen if there was a bug loading payouts from CSV
			sink.ReportError("Duplicate CSV line %d detected in CSV payouts", csvPayout.CSVLine)
		}
		csvPayoutsByLine[csvPayout.CSVLine] = csvPayout
	}
	dbPayoutsByLine := make(map[int]*pipelinedb.Payout)
	for _, dbPayout := range dbPayouts {
		if _, ok := dbPayoutsByLine[dbPayout.CSVLine]; ok {
			// This would only happen if there was a bug loading payouts from CSV
			sink.ReportError("Duplicate CSV line %d detected in database payouts", dbPayout.CSVLine)
		}
		dbPayoutsByLine[dbPayout.CSVLine] = dbPayout
	}

	mismatched := map[int]struct{}{}

	// Ensure each CSV payout is represented accurately in the DB
	sink.ReportStatus("Reconciling CSV payout entries...")
	for _, csvPayout := range csvPayouts {
		dbPayout, ok := dbPayoutsByLine[csvPayout.CSVLine]
		if !ok {
			sink.ReportError("No payout for CSV line %d in database", csvPayout.CSVLine)
			mismatched[csvPayout.CSVLine] = struct{}{}
			continue
		}
		if dbPayout.Payee != csvPayout.Payee {
			sink.ReportError("Payee mismatch on CSV line %d: csv=%q db=%q", csvPayout.CSVLine, csvPayout.Payee, dbPayout.Payee)
			mismatched[csvPayout.CSVLine] = struct{}{}
			continue
		}
		if !dbPayout.USD.Equal(csvPayout.USD) {
			sink.ReportError("Amount mismatch on CSV line %d: csv=%q db=%q", csvPayout.CSVLine, csvPayout.USD, dbPayout.USD)
			mismatched[csvPayout.CSVLine] = struct{}{}
			continue
		}
	}

	// Ensure each DB payout is represented accurately in the CSV
	sink.ReportStatus("Reconciling DB payout entries...")
	for _, dbPayout := range dbPayouts {
		csvPayout, ok := csvPayoutsByLine[dbPayout.CSVLine]
		if !ok {
			sink.ReportError("No payout for CSV line %d in database", dbPayout.CSVLine)
			mismatched[dbPayout.CSVLine] = struct{}{}
			continue
		}
		if dbPayout.Payee != csvPayout.Payee {
			sink.ReportError("Payee mismatch on CSV line %d: csv=%q db=%q", csvPayout.CSVLine, csvPayout.Payee, dbPayout.Payee)
			mismatched[dbPayout.CSVLine] = struct{}{}
			continue
		}
		if !dbPayout.USD.Equal(csvPayout.USD) {
			sink.ReportError("Amount mismatch on CSV line %d: csv=%q db=%q", csvPayout.CSVLine, csvPayout.USD, dbPayout.USD)
			mismatched[dbPayout.CSVLine] = struct{}{}
			continue
		}
	}

	stats.Mismatched = int64(len(mismatched))

	// Confirm the status of each transaction to ensure we haven't accidentally
	// overpaid.
	sink.ReportStatus("Confirming TX status (transactions)...")
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
			sink.ReportStatus("Confirming TX status (%d/%d)...", which, len(txs))
		}
		state, err := auditor.CheckTransactionState(ctx, tx.Hash)

		if err != nil {
			return nil, err
		}

		if tx.State == state {
			continue
		}

		if tx.State == pipelinedb.TxDropped && state == pipelinedb.TxConfirmed {
			sink.ReportError("Double pay for payout group %d (tokens=%s)", tx.PayoutGroupID, tx.StorjTokens)
			stats.DoublePays++
			stats.DoublePayStorj.Add(stats.DoublePayStorj, tx.StorjTokens)
		} else {
			sink.ReportWarn("TX state mismatch on hash %q (db=%q, node=%q)", tx.Hash, tx.State, state)
		}
	}

	// For each payout, ensure it belongs to a payout group with a confirmed
	// transaction. Reconfirm the transaction against the blockchain.
	sink.ReportStatus("Checking payouts status...")
	payoutGroupStatus := make(map[int64]string)
	var payoutsConfirmed int64
	for _, dbPayout := range dbPayouts {
		if txHash, ok := payoutGroupStatus[dbPayout.PayoutGroupID]; ok {
			if txHash != "" {
				receiptsCSV.Write([]string{dbPayout.Payee.String(), dbPayout.USD.String(), txHash, string(payerType)})
			}
			continue
		}
		// Mark the payout group status as done with no transaction. It will be
		// marked with the confirming transaction after passing the checks below.
		payoutGroupStatus[dbPayout.PayoutGroupID] = ""

		numPayouts, err := db.FetchPayoutGroupPayoutCount(ctx, dbPayout.PayoutGroupID)
		if err != nil {
			return nil, err
		}

		txs, err := db.FetchPayoutGroupTransactions(ctx, dbPayout.PayoutGroupID)
		if err != nil {
			return nil, err
		}
		if len(txs) == 0 {
			sink.ReportError("Payout of %s to %s on line %d has no transactions",
				dbPayout.USD, dbPayout.Payee.String(), dbPayout.CSVLine)
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
				sink.ReportError("Unexpected tx state %q on %s", tx.State, tx.Hash)
			}
		}

		if len(confirmed) == 0 {
			sink.ReportError("Payout of %s to %s on line %d has no confirmed transactions (pending=%d dropped=%d failed=%d)",
				dbPayout.USD, dbPayout.Payee.String(), dbPayout.CSVLine,
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
				sink.ReportError("Failed to get receipt for transaction %s for payout of %s to %s on line %d",
					tx.Hash, dbPayout.USD, dbPayout.Payee.String(), dbPayout.CSVLine)
			case state != pipelinedb.TxConfirmed:
				sink.ReportError("Transaction %s was %s instead of confirmed for payout of %s to %s on line %d",
					tx.Hash, state, dbPayout.USD, dbPayout.Payee.String(), dbPayout.CSVLine)
			default:
				confirmedCount++
			}
		}

		if confirmedCount > 0 {
			txHash := confirmed[0].Hash
			payoutGroupStatus[dbPayout.PayoutGroupID] = txHash
			receiptsCSV.Write([]string{dbPayout.Payee.String(), dbPayout.USD.String(), txHash, string(payerType)})
			payoutsConfirmed += numPayouts
		}

		switch {
		case confirmedCount > 1:
			sink.ReportError("Payout of %s to %s on line %d has more than one (%d) confirmed transactions recorded",
				dbPayout.USD, dbPayout.Payee.String(), dbPayout.CSVLine,
				len(confirmed))
			stats.Overpaid += numPayouts
		case confirmedCount == 0:
			stats.FalseConfirmed += numPayouts
		default:
			stats.Confirmed += numPayouts
		}
	}

	// If all payout groups are confirmed and a receipts output has been
	// configured then dump the receipts CSV.
	switch {
	case receiptsOut == "":
	case payoutsConfirmed == stats.Total || receiptsForce:
		receiptsCSV.Flush()
		sink.ReportStatus("Writing receipts to %s...", receiptsOut)
		if err := ioutil.WriteFile(receiptsOut, receiptsBuf.Bytes(), 0644); err != nil {
			return nil, err
		}
	default:
		sink.ReportStatus("Skipping writing receipts to %s; only %d of %d payouts confirmed", receiptsOut, payoutsConfirmed, stats.Total)
	}

	sink.ReportStatus("Done.")

	return stats, nil
}
