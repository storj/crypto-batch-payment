package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"os"
	"time"

	cryptohopper "storj.io/crypto-batch-payment/pkg"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/manifoldco/promptui"
	"github.com/zeebo/errs"
)

var (
	usageErr = errs.Class("usage")
)

func loadETHKey(path, which string) (*ecdsa.PrivateKey, common.Address, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, common.Address{}, errs.New("unable to stat %s key: %v\n", which, err)
	}

	if (fi.Mode() & 0177) != 0 {
		return nil, common.Address{}, errs.New("%s mode %#o is too permissive (set to 0600)\n", path, fi.Mode())
	}

	key, err := crypto.LoadECDSA(path)
	if err != nil {
		return nil, common.Address{}, errs.New("unable to load %s key: %v\n", which, err)
	}
	return key, crypto.PubkeyToAddress(key.PublicKey), nil
}

func dialNode(address string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(address)
	if err != nil {
		return nil, errs.New("Failed to dial node %q: %v\n", address, err)
	}
	return client, nil
}

func convertAddress(s, which string) (common.Address, error) {
	address, err := cryptohopper.AddressFromString(s)
	if err != nil {
		return common.Address{}, usageErr.New("invalid %s address: %v\n", which, err)
	}
	return address, nil
}

func convertHash(s string) (common.Hash, error) {
	address, err := cryptohopper.HashFromString(s)
	if err != nil {
		return common.Hash{}, usageErr.New("invalid transaction hash: %v\n", err)
	}
	return address, nil
}

func convertInt(s string, base int, which string) (*big.Int, error) {
	// use float so that we can accept scientific notation
	f, ok := new(big.Float).SetString(s)
	if !ok {
		return nil, usageErr.New("invalid %s integer %q\n", which, s)
	}
	i, a := f.Int(nil)
	// make sure it there is no fractional component
	if a != big.Exact {
		return nil, usageErr.New("invalid %s integer %q\n", which, s)
	}
	return i, nil
}

func promptConfirm(label string) error {
	_, err := (&promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}).Run()
	if err != nil {
		return errors.New("aborted")
	}
	return nil
}

func waitForTransaction(ctx context.Context, client *ethclient.Client, hash common.Hash) error {
	fmt.Printf("Transaction hash is %s\n", hash.String())
	fmt.Printf("Waiting for transaction to be confirmed...\n")
	for {
		fmt.Print(".")
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "Wait canceled (%+v). Transaction may still confirm.\n", ctx.Err())
			return ctx.Err()
		}

		receipt, err := client.TransactionReceipt(ctx, hash)
		switch {
		case err == nil:
			fmt.Println()
			if receipt.Status == types.ReceiptStatusSuccessful {
				return nil
			}
			fmt.Fprintf(os.Stderr, "Transaction failed with status %d\n", receipt.Status)
			return errors.New("transaction failed")
		case err == ethereum.NotFound:
		default:
			fmt.Println()
			fmt.Fprintf(os.Stderr, "Failed to query for transaction receipt: %+v\n", err)
		}

		_, _, err = client.TransactionByHash(ctx, hash)
		switch {
		case err == nil:
		case err == ethereum.NotFound:
			fmt.Println()
			fmt.Fprintf(os.Stderr, "Transaction was dropped\n")
			return errors.New("transaction dropped")
		default:
			fmt.Println()
			fmt.Fprintf(os.Stderr, "Failed to query for transaction by hash: %+v\n", err)
		}
	}
}

func loadFirstLine(p string) (_ string, err error) {
	f, err := os.Open(p)
	if err != nil {
		return "", errs.Wrap(err)
	}
	defer func() {
		errs.Combine(err, f.Close())
	}()
	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return "", errs.Wrap(scanner.Err())
	}
	return scanner.Text(), nil
}
