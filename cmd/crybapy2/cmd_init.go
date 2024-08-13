package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"slices"

	"github.com/kyokomi/emoji/v2"
	"github.com/shopspring/decimal"
	"github.com/zeebo/clingy"
	"golang.org/x/exp/maps"

	"storj.io/crypto-batch-payment/pkg/payouts2"
)

type cmdInit struct {
	bonusMultiplier decimal.Decimal
}

func (c *cmdInit) Setup(params clingy.Parameters) {
	c.bonusMultiplier = optDecimalFlag(params, "bonus-multiplier", "The bonus multiplier to apply to bonus payouts", "1")
}

func (c *cmdInit) Execute(ctx context.Context) error {
	if !c.bonusMultiplier.IsPositive() {
		return errors.New("bonus-multiplier must be positive")
	}

	csvPaths, err := filepath.Glob("./*-prepayouts.csv")
	if err != nil {
		return fmt.Errorf("unable to locate prepayouts CSVs: %w", err)
	}

	if len(csvPaths) == 0 {
		return errors.New("no prepayout CSVs located in current directory")
	}

	params := payouts2.InitParams{CSVPaths: csvPaths, BonusMultiplier: c.bonusMultiplier}
	ui := &initUI{stdout: clingy.Stdout(ctx)}

	return payouts2.Init(ctx, params, ui)
}

type initCSVStats struct {
	aggregated int
	skipped    [payouts2.RowSkipReasonMax]int
}

type initUI struct {
	stdout      io.Writer
	longestPath int
	csvStats    map[string]*initCSVStats
}

func (i *initUI) Started(evt payouts2.StartedEvent) {
	longestPath := 0
	for _, csvPath := range evt.CSVPaths {
		if len(csvPath) > longestPath {
			longestPath = len(csvPath)
		}
	}
	i.longestPath = longestPath
}

func (i *initUI) CSVLoaded(evt payouts2.CSVLoadedEvent) {
	ji := ":white_check_mark:"
	result := fmt.Sprint("OK")
	if evt.Err != nil {
		ji = ":x:"
		result = evt.Err.Error()
	}
	format := fmt.Sprintf("%s %%%ds: %%s\n", ji, i.longestPath)
	i.printf(format, evt.CSVPath, result)
}

func (i *initUI) RowAggregated(evt payouts2.RowAggregatedEvent) {
	stats := i.csvStatsFor(evt.CSVPath)
	stats.aggregated++
}

func (i *initUI) RowSkipped(evt payouts2.RowSkippedEvent) {
	stats := i.csvStatsFor(evt.CSVPath)
	stats.skipped[evt.Reason]++
}

func (i *initUI) RowsAggregated(evt payouts2.RowsAggregatedEvent) {
	stats := i.csvStatsFor(evt.CSVPath)

	format := fmt.Sprintf(":information_source: %%%ds  ... %%d rows aggregated\n", i.longestPath)
	i.printf(format, "", stats.aggregated)

	for rowSkipReason, count := range stats.skipped {
		if count > 0 {
			format := fmt.Sprintf(":warning: %%%ds  ... %%d %%s rows skipped\n", i.longestPath)
			i.printf(format, "", count, payouts2.RowSkipReason(rowSkipReason))
		}
	}
}

func (i *initUI) CSVsLoaded(evt payouts2.CSVsLoadedEvent) {
	format := fmt.Sprintf(":information_source: %%%ds: %%d\n", i.longestPath)

	typeKeys := maps.Keys(evt.ByType)
	slices.Sort(typeKeys)

	var total int
	for _, payerType := range typeKeys {
		count := len(evt.ByType[payerType])
		i.printf(format, payerType, count)
		total += count
	}
	i.printf(format, "total", total)
}

func (i *initUI) csvStatsFor(csvPath string) *initCSVStats {
	stats, ok := i.csvStats[csvPath]
	if ok {
		return stats
	}

	if i.csvStats == nil {
		i.csvStats = make(map[string]*initCSVStats)
	}

	stats = new(initCSVStats)
	i.csvStats[csvPath] = stats
	return stats
}

func (i *initUI) printf(format string, args ...interface{}) {
	_, _ = emoji.Fprintf(i.stdout, format, args...)
}
