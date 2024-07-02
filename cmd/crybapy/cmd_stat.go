package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

type statConfig struct {
	rootConfig *rootConfig
	Root       string
}

func newStatCommand(rootConfig *rootConfig) *cobra.Command {
	config := &statConfig{
		rootConfig: rootConfig,
	}
	cmd := &cobra.Command{
		Use:   "stat",
		Short: "Print out statistical information about payment",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				config.Root = args[0]
			} else {
				config.Root = "."
			}
			return checkCmd(doStat(config))
		},
	}

	return cmd
}

func doStat(config *statConfig) error {
	out := csv.NewWriter(os.Stdout)
	defer out.Flush()
	err := out.Write([]string{"file", "confirmed_txs", "all_txs", "payouts", "usd_amount", "used_gas"})
	if err != nil {
		return errs.Wrap(err)
	}
	ctx := context.Background()
	return filepath.WalkDir(config.Root, func(path string, entry os.DirEntry, err error) error {
		if entry.Name() == "payouts.db" {
			err := printFileStat(ctx, out, path)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "Ignoring", path, err.Error())
			}
		}
		return nil
	})
}

func printFileStat(ctx context.Context, out *csv.Writer, f string) error {
	db, err := pipelinedb.OpenDB(ctx, f, false)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()
	payouts, err := db.FetchPayouts(ctx)
	if err != nil {
		return err
	}
	var sum decimal.Decimal
	for _, p := range payouts {
		sum = sum.Add(p.USD)
	}

	txs, err := db.FetchTransactions(ctx)
	if err != nil {
		return err
	}
	gasUsed := uint64(0)
	okTxs := 0
	for _, tx := range txs {
		if tx.Receipt != nil {
			gasUsed += tx.Receipt.GasUsed
		}
		if tx.State == pipelinedb.TxConfirmed {
			okTxs++
		}

	}
	avgUsedGas := 0
	if okTxs > 0 {
		avgUsedGas = int(gasUsed) / okTxs
	}
	err = out.Write([]string{
		f,
		fmt.Sprintf("%d", okTxs),
		fmt.Sprintf("%d", len(txs)),
		fmt.Sprintf("%d", len(payouts)),
		sum.String(),
		fmt.Sprintf("%d", gasUsed),
		fmt.Sprintf("%d", avgUsedGas),
	})
	if err != nil {
		return err
	}

	return nil
}
