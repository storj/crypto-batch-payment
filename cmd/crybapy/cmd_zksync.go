package main

import (
	"github.com/spf13/cobra"
)

type zkSyncConfig struct {
	*rootConfig
}

func newZkSyncCommand(rootConfig *rootConfig) *cobra.Command {
	config := &zkSyncConfig{
		rootConfig: rootConfig,
	}
	cmd := &cobra.Command{
		Use:   "zksync",
		Short: "ZkSync utility commands",
	}
	cmd.AddCommand(newZkSyncBalanceCommand(config))
	cmd.AddCommand(newZkSyncDepositCommand(config))
	return cmd
}
