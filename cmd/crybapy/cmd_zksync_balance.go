package main

import (
	"context"
	"fmt"
	"io"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"
)

type zkSyncBalanceConfig struct {
	*zkSyncConfig
	Account string
}

func newZkSyncBalanceCommand(zkSyncConfig *zkSyncConfig) *cobra.Command {
	config := &zkSyncBalanceConfig{
		zkSyncConfig: zkSyncConfig,
	}
	cmd := &cobra.Command{
		Use:   "balance ACCOUNT",
		Short: "Checks the zksync (L2) balance for an account",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.Account = args[0]
			return checkCmd(doZkSyncBalance(config, os.Stdout))
		},
	}

	return cmd
}

func doZkSyncBalance(config *zkSyncBalanceConfig, writer io.Writer) error {
	ctx := context.Background()
	client, err := rpc.DialContext(ctx, config.NodeAddress+"/jsrpc")
	if err != nil {
		return err
	}
	defer client.Close()

	result := map[string]interface{}{}
	err = client.CallContext(ctx, &result, "account_info", config.Account)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(writer, "Committed balance:")
	if err != nil {
		return err
	}

	err = printZkSyncBalance(ctx, config.NodeAddress, writer, result["committed"].(map[string]interface{})["balances"])
	if err != nil {
		return err
	}

	return nil
}

func printZkSyncBalance(ctx context.Context, address string, w io.Writer, i interface{}) error {
	for v, k := range i.(map[string]interface{}) {
		value, _ := new(big.Int).SetString(k.(string), 10)
		tokenInfo := struct {
			Result struct {
				Decimals int32
				Symbol   string
			}
		}{}

		err := zkSyncRestAPI(ctx, address, "/api/v0.2/tokens/"+v, &tokenInfo)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "%s: %s\n", v, printToken(value, tokenInfo.Result.Decimals, tokenInfo.Result.Symbol))
		if err != nil {
			return err
		}
	}
	return nil
}
