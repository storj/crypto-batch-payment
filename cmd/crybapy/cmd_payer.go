package main

import (
	"github.com/spf13/cobra"
)

type payerCommandConfig struct {
	*rootConfig
	PayerConfig
}

func newPayerCommand(rootConfig *rootConfig) *cobra.Command {
	config := &payerCommandConfig{
		rootConfig:  rootConfig,
		PayerConfig: PayerConfig{},
	}
	cmd := &cobra.Command{
		Use:   "payer",
		Short: "Access to the low-level payer commands",
	}
	cmd.AddCommand(newPayerBalanceCommand(config))
	cmd.AddCommand(newPayerTransferCommand(config))
	return cmd
}
