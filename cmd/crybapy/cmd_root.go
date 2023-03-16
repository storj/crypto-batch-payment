package main

import (
	"context"
	"fmt"
	"runtime"
	"runtime/debug"
	"storj.io/crypto-batch-payment/pkg/storjtoken"

	"github.com/spf13/cobra"
)

const (
	defaultNodeAddress = "/home/storj/.ethereum/geth.ipc"

	defaultDataDir = "."
)

type rootConfig struct {
	Ctx context.Context

	NodeAddress string
	ChainID     string
	DataDir     string
}

func newRootCommand() *cobra.Command {
	config := new(rootConfig)
	cmd := &cobra.Command{
		Use:   "payouts",
		Short: "Manage and execute STORJ Token payouts",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			config.Ctx = cmdCtx()
			return nil
		},
		Version: getVersion(),
	}
	cmd.PersistentFlags().StringVarP(
		&config.NodeAddress,
		"node-address", "",
		defaultNodeAddress,
		"Address of the ETH node to use")
	cmd.PersistentFlags().StringVarP(
		&config.ChainID,
		"chain-id", "",
		storjtoken.DefaultChainID.String(),
		"Address of the STORJ contract on the network")
	cmd.PersistentFlags().StringVarP(
		&config.DataDir,
		"data-dir", "",
		defaultDataDir,
		"Directory to store data (e.g. payout metadata)")

	cmd.AddCommand(newImportCommand(config))
	cmd.AddCommand(newRunCommand(config))
	cmd.AddCommand(newAuditCommand(config))
	cmd.AddCommand(newPriceCommand(config))
	cmd.AddCommand(newZkSyncCommand(config))
	return cmd
}

func getVersion() string {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}

	return fmt.Sprintf("%s (built with %s)\n", buildInfo.Main.Version, runtime.Version())
}
