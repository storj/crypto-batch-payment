package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"os/signal"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeebo/errs/v2"

	"github.com/mitchellh/go-homedir"
)

var homeDir string

func init() {
	var err error
	homeDir, err = homedir.Dir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to determine home directory: %v\n", err)
		os.Exit(1)
	}
}

func cmdCtx() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() {
		sig := <-ch
		fmt.Fprintf(os.Stderr, "Signal %q received\n", sig)
		cancel()
	}()
	return ctx
}

func checkCmd(err error) error {
	switch {
	case err == nil:
		return nil
	case usageErr.Has(err):
		// If it is a usage error, return it directly so cobra command will
		// show usage. Otherwise, print and exit with non-zero exit status.
		return err
	}
	// other errors exit with 2
	fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	os.Exit(2)
	return err
}

// setGasCap adjust gas tips in the opts for ad-hoc transactions.
func setGasCap(ctx context.Context, client *ethclient.Client, opts *bind.TransactOpts, gasTipCap int64) (estimatedGasPrice *big.Int, err error) {
	head, err := client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	// We don't use gas oracle here as it's unreliable
	opts.GasTipCap = big.NewInt(gasTipCap)

	// this is the max limit. Only (baseFee + gasTip) * gas will be paid, but as the base fee can be moved meantime we
	// use the standard estimation here: 2 * currentBaseFee + tip
	opts.GasFeeCap = new(big.Int).Add(new(big.Int).Mul(head.BaseFee(), big.NewInt(2)), opts.GasTipCap)

	return new(big.Int).Add(head.BaseFee(), opts.GasTipCap), nil
}
