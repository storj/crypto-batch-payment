package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"

	"github.com/zeebo/clingy"
	"github.com/zeebo/errs"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"storj.io/crypto-batch-payment/pkg/config"
	"storj.io/crypto-batch-payment/pkg/fancy"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/payouts2"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/receipts"
)

type cmdAudit struct {
	config       string
	force        bool
	pretend      bool
	receiptsPath string
}

func (cmd *cmdAudit) Setup(params clingy.Parameters) {
	cmd.config = stringFlag(params, "config", "The configuration file", "./config.toml")
	cmd.force = toggleFlag(params, "force", "Force writing the receipts even if there is a payouts problem", false)
	cmd.pretend = toggleFlag(params, "pretend", "pretend to audit the payouts", false)
	cmd.receiptsPath = stringArg(params, "RECEIPTSPATH", "Path on disk to write the receipts to")
}

func (cmd *cmdAudit) Execute(ctx context.Context) error {
	stdout := clingy.Stdout(ctx)
	stderr := clingy.Stderr(ctx)
	sink := &auditSink{out: stdout, err: stderr}

	cfg, err := config.Load(cmd.config)
	if err != nil {
		return fmt.Errorf("unable to load config: %w", err)
	}

	auditors, err := cfg.NewAuditors(ctx)
	if err != nil {
		return fmt.Errorf("failed to init payers: %w", err)
	}
	defer auditors.Close()

	dbs, err := loadDBs(ctx)
	if err != nil {
		return err
	}

	csvPaths, err := filepath.Glob("./*-prepayouts.csv")
	if err != nil {
		return fmt.Errorf("unable to locate prepayouts CSVs: %w", err)
	}

	if len(csvPaths) == 0 {
		return errors.New("no prepayout CSVs located in current directory")
	}

	dbStats, err := payouts2.AuditDBs(ctx, dbs, csvPaths, sink)
	if err != nil {
		return err
	}

	fancy.Finfoln(stdout, "DB audit complete.")
	fancy.Fprintf(stdout, errorIfNonZero(dbStats.MissingCSVs),
		"Missing CSVs................: %d\n", dbStats.MissingCSVs)
	fancy.Fprintf(stdout, errorIfNonZero(dbStats.MissingDBs),
		"Missing DBs.................: %d\n", dbStats.MissingDBs)
	fancy.Fprintf(stdout, errorIfNonZero(dbStats.Mismatched),
		"Mismatched Payouts..........: %d\n", dbStats.Mismatched)

	var receipts receipts.Buffer
	var bad bool
	var allTxStats = make(map[payer.Type]*payouts2.TransactionStats)
	for payerType, db := range dbs {
		auditor, ok := auditors[payerType]
		if !ok {
			fancy.Ferrorf(stdout, "No auditor for payer type %q\n", payerType)
			bad = true
			continue
		}
		if cmd.pretend {
			auditor = pretendAuditor{}
		}
		fancy.Finfof(stdout, "Auditing %q transactions...\n", payerType)
		txStats, err := payouts2.AuditTransactions(ctx, payerType, auditor, db, sink, &receipts)
		if err != nil {
			return err
		}
		fancy.Finfof(stdout, "Transactions audit complete (%s)\n", payerType)
		allTxStats[payerType] = txStats
	}

	sortedPayers := maps.Keys(allTxStats)
	slices.Sort(sortedPayers)

	for _, payerType := range sortedPayers {
		txStats := allTxStats[payerType]
		fancy.Finfoln(stdout)
		fancy.Finfof(stdout, "Transactions stats (%s):\n", payerType)
		fancy.Finfof(stdout, "Total.......................: %d\n", txStats.Total)
		if txStats.Confirmed != txStats.Total {
			fancy.Fprintf(stdout, fancy.Warn, "Confirmed...................: %d\n", txStats.Confirmed)
			bad = true
		} else {
			fancy.Fprintf(stdout, fancy.Info, "Confirmed...................: %d\n", txStats.Confirmed)
		}
		fancy.Finfof(stdout, "False Confirmed.............: %d\n", txStats.FalseConfirmed)
		fancy.Finfof(stdout, "Overpaid....................: %d\n", txStats.Overpaid)
		fancy.Finfof(stdout, "Skipped.....................: %d\n", txStats.Skipped)
		fancy.Finfof(stdout, "Unstarted...................: %d\n", txStats.Unstarted)
		fancy.Finfof(stdout, "Pending.....................: %d\n", txStats.Pending)
		fancy.Finfof(stdout, "Failed......................: %d\n", txStats.Failed)
		fancy.Finfof(stdout, "Dropped.....................: %d\n", txStats.Dropped)
		fancy.Finfof(stdout, "Unknown.....................: %d\n", txStats.Unknown)

		if txStats.MismatchedState > 0 {
			fancy.Ferrorf(stdout, "Mismatched State............: %d\n", txStats.MismatchedState)
			bad = true
		}

		if txStats.DoublePays > 0 {
			fancy.Ferrorf(stdout, "Double Pays.................: %d\n", txStats.DoublePays)
			fancy.Ferrorf(stdout, "Double Pay Amount (raw STORJ value): %s\n", &txStats.DoublePayStorj)
			bad = true
		}
	}

	if bad {
		fancy.Finfoln(stdout)
		fancy.Ferrorln(stdout, "There were one or more problems with the payouts")
	}

	// If all payout groups are confirmed and a receipts output has been
	// configured then dump the receipts CSV.
	switch {
	case cmd.receiptsPath == "":
	case !bad || cmd.force:
		fancy.Finfof(stdout, "Writing receipts to %s...\n", cmd.receiptsPath)
		if err := os.WriteFile(cmd.receiptsPath, receipts.Finalize(), 0644); err != nil {
			return errs.Wrap(err)
		}
	default:
		fancy.Fwarnln(stdout, "Skipping writing receipts due to bad payouts (force writing with --force)")
	}

	fancy.Finfoln(stdout, "Done.")

	return nil
}

type auditSink struct {
	out io.Writer
	err io.Writer
}

func (s *auditSink) ReportStatusf(format string, args ...any) {
	fancy.Finfoln(s.out, fmt.Sprintf(format, args...))
}

func (s *auditSink) ReportWarnf(format string, args ...any) {
	fancy.Fwarnln(s.err, fmt.Sprintf(format, args...))
}

func (s *auditSink) ReportErrorf(format string, args ...any) {
	fancy.Ferrorln(s.err, fmt.Sprintf(format, args...))
}

func errorIfNonZero[T constraints.Integer](v T) fancy.Level {
	if v != 0 {
		return fancy.Error
	}
	return fancy.Info
}

type pretendAuditor struct{}

func (pretendAuditor) CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	return pipelinedb.TxConfirmed, nil
}

func (pretendAuditor) CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	return pipelinedb.TxConfirmed, nil
}

func (pretendAuditor) Close() {}
