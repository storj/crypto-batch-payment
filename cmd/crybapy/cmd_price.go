package main

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zeebo/errs"
)

type priceConfig struct {
	*rootConfig
}

func newPriceCommand(rootConfig *rootConfig) *cobra.Command {
	config := &priceConfig{
		rootConfig: rootConfig,
	}
	return &cobra.Command{
		Use:   "price",
		Short: "Print out legacy (calculated) vs EIP-1559 gas price information.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return doPrice(config)
		},
	}
}

func doPrice(config *priceConfig) error {
	ctx := context.Background()
	client, err := dialNode(config.NodeAddress)
	if err != nil {
		return err
	}
	defer client.Close()

	lastHeader, err := client.HeaderByNumber(ctx, nil)
	if err != nil {
		return errs.Wrap(err)
	}

	fmt.Println("Last block base fee: " + lastHeader.BaseFee.String())

	return nil
}
