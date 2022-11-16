package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/payouts"
)

type importConfig struct {
	*rootConfig

	// CSVPath is the path to the CSV file containing payout data
	CSVPath string
}

func newImportCommand(rootConfig *rootConfig) *cobra.Command {
	config := &importConfig{
		rootConfig: rootConfig,
	}
	return &cobra.Command{
		Use:   "import CSVPATH",
		Short: "Imports a payout from a CSV file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.CSVPath = args[0]
			return checkCmd(doImport(config))
		},
	}
}

func doImport(config *importConfig) error {
	fmt.Printf("Importing payouts from %q...\n", config.CSVPath)
	if err := payouts.Import(config.Ctx, config.DataDir, config.CSVPath); err != nil {
		return errs.New("import failed: %v\n", err)
	}
	fmt.Println("Import complete.")
	return nil
}
