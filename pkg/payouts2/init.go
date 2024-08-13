package payouts2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/prepayoutcsv"
)

type InitParams struct {
	CSVPaths        []string
	BonusMultiplier decimal.Decimal
}

func Init(ctx context.Context, params InitParams, ui InitUI) error {
	if len(params.CSVPaths) == 0 {
		return errors.New("prepayout CSVs are required to initialize payouts")
	}

	byType, err := loadCSVs(params.CSVPaths, params.BonusMultiplier, ui)
	if err != nil {
		return err
	}

	for payerType, payouts := range byType {
		if err := initDB(ctx, params.BonusMultiplier, payerType, payouts); err != nil {
			return err
		}
	}

	return nil
}

func loadCSVs(csvPaths []string, bonusMultiplier decimal.Decimal, ui InitUI) (ByType, error) {
	ui.Started(StartedEvent{CSVPaths: csvPaths})

	aggregation := new(payoutAggregation)

	var loadFailed bool
	for _, csvPath := range csvPaths {
		prepayoutRows, err := prepayoutcsv.Load(csvPath)
		if err != nil {
			ui.CSVLoaded(CSVLoadedEvent{CSVPath: csvPath, Err: err})
			loadFailed = true
			continue
		}
		ui.CSVLoaded(CSVLoadedEvent{CSVPath: csvPath, NumRows: len(prepayoutRows)})

		for _, prepayoutRow := range prepayoutRows {
			switch {
			// Filter out rows with invalid addresses
			case prepayoutRow.Address == (common.Address{}):
				ui.RowSkipped(RowSkippedEvent{CSVPath: csvPath, Line: prepayoutRow.Line, Reason: RowInvalid})
				continue
			// Filter out sanctioned rows
			case prepayoutRow.Sanctioned:
				ui.RowSkipped(RowSkippedEvent{CSVPath: csvPath, Line: prepayoutRow.Line, Reason: RowSanctioned})
				continue
			}

			typ, err := payer.TypeFromString(prepayoutRow.Kind)
			if err != nil {
				return nil, errs.New("invalid kind %q: %v", typ, err)
			}

			amount := prepayoutRow.Amount
			if prepayoutRow.Bonus {
				amount = amount.Mul(bonusMultiplier)
			}

			aggregation.Add(typ, pipelinedb.Payout{
				Payee:     prepayoutRow.Address,
				USD:       amount,
				Mandatory: prepayoutRow.Mandatory,
			})

			ui.RowAggregated(RowAggregatedEvent{CSVPath: csvPath, Line: prepayoutRow.Line})
		}

		ui.RowsAggregated(RowsAggregatedEvent{CSVPath: csvPath})
	}

	if loadFailed {
		return nil, errs.New("failed to load one or more prepayout CSVs")
	}

	byType := aggregation.Finalize()

	ui.CSVsLoaded(CSVsLoadedEvent{
		ByType: byType,
	})

	return byType, nil
}

type ByType map[payer.Type][]*pipelinedb.Payout

type payoutAggregation struct {
	byType map[payer.Type]map[common.Address]*pipelinedb.Payout
}

func (agg *payoutAggregation) Add(payerType payer.Type, payout pipelinedb.Payout) {
	if agg.byType == nil {
		agg.byType = make(map[payer.Type]map[common.Address]*pipelinedb.Payout)
	}
	byPayee, ok := agg.byType[payerType]
	if !ok {
		byPayee = make(map[common.Address]*pipelinedb.Payout)
		agg.byType[payerType] = byPayee
	}

	if existing, ok := byPayee[payout.Payee]; ok {
		payout.USD = payout.USD.Add(existing.USD)
		if !existing.Mandatory {
			payout.Mandatory = false
		}
	}

	byPayee[payout.Payee] = &payout
}

func (agg *payoutAggregation) Finalize() ByType {
	final := make(ByType)
	for payerType, byPayee := range agg.byType {
		for _, payout := range byPayee {
			if !decimal.Zero.LessThan(payout.USD) {
				continue
			}
			final[payerType] = append(final[payerType], payout)
		}
	}

	// Now sort payouts by address for a given payer type. This is just a
	// nice-to-have so that progress through the pipeline can be somewhat
	// implied by observing the address space currently being processed.
	for _, payouts := range final {
		sort.Slice(payouts, func(i, j int) bool {
			return bytes.Compare(payouts[i].Payee[:], payouts[j].Payee[:]) < 0
		})
	}
	return final
}

func initDB(ctx context.Context, bonusMultiplier decimal.Decimal, kind payer.Type, payouts []*pipelinedb.Payout) error {
	tmpDir, err := os.MkdirTemp(".", "")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tmpPath := filepath.Join(tmpDir, payoutDBName(kind))

	db, err := pipelinedb.NewDB(ctx, tmpPath)
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = db.Close() }()

	if err := db.SetBonusMultiplier(ctx, bonusMultiplier); err != nil {
		return errs.Wrap(err)
	}

	if err := createPayoutGroups(ctx, db, payouts); err != nil {
		return err
	}

	if err := db.Close(); err != nil {
		return errs.Wrap(err)
	}

	if err := os.Rename(tmpPath, payoutDBName(kind)); err != nil {
		return errs.Wrap(err)
	}

	return nil
}

func payoutDBName(payerType payer.Type) string {
	return fmt.Sprintf("payout.%s.db", payerType)
}

func createPayoutGroups(ctx context.Context, db *pipelinedb.DB, payouts []*pipelinedb.Payout) error {
	const groupSize = 1
	groups := make([][]*pipelinedb.Payout, 0, len(payouts))
	for i := 0; i < len(payouts); {
		end := i + groupSize
		if end > len(payouts) {
			end = len(payouts)
		}
		groups = append(groups, payouts[i:end])
		i = end
	}
	if err := db.CreatePayoutGroups(ctx, groups); err != nil {
		return err
	}
	return nil
}
