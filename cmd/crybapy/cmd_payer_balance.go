package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

type payerBalanceConfig struct {
	*payerCommandConfig
	Account string
}

func newPayerBalanceCommand(parentConfig *payerCommandConfig) *cobra.Command {
	config := &payerBalanceConfig{
		payerCommandConfig: parentConfig,
	}
	cmd := &cobra.Command{
		Use:   "balance ACCOUNT",
		Short: "Checks the current token balance based with the configured payer implementation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkCmd(doPayerBalance(config, args[0]))
		},
	}
	RegisterFlags(cmd, &config.payerCommandConfig.PayerConfig)
	return cmd
}

func doPayerBalance(config *payerBalanceConfig, spenderKeyPath string) error {
	log, err := openConsoleLog()
	if err != nil {
		return err
	}
	payer, _, err := CreatePayer(config.Ctx, log, config.PayerConfig, config.NodeAddress, config.ChainID, spenderKeyPath)
	if err != nil {
		return err
	}
	balance, err := payer.GetTokenBalance(config.Ctx)
	if err != nil {
		return err
	}

	fmt.Println(printToken(balance, payer.Decimals(), ""))
	return nil
}
