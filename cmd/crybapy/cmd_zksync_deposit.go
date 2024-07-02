package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
	"github.com/zeebo/errs"
	"storj.io/crypto-batch-payment/pkg/contract"
)

type zkSyncDepositConfig struct {
	*zkSyncConfig
	Account        string
	Amount         string
	SpenderKeyPath string
	GasTipCap      int64
	ZkNodeAddress  string
}

func newZkSyncDepositCommand(zkSyncConfig *zkSyncConfig) *cobra.Command {
	config := &zkSyncDepositConfig{
		zkSyncConfig: zkSyncConfig,
	}
	cmd := &cobra.Command{
		Use:   "deposit SPENDERKEYPATH ACCOUNT AMOUNT",
		Short: "Deposit STORJ token from L1 to L2",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			config.SpenderKeyPath = args[0]
			config.Account = args[1]
			config.Amount = args[2]
			return checkCmd(doZkSyncDeposit(config))
		},
	}
	cmd.PersistentFlags().Int64VarP(
		&config.GasTipCap,
		"gas-tip-cap", "",
		1_000_000_000,
		"Gas tip cap, paid on top of the base gas.")
	cmd.Flags().StringVarP(
		&config.ZkNodeAddress,
		"zk-node-address", "",
		"https://api.zksync.io",
		"ZkSync api address")
	return cmd
}

func doZkSyncDeposit(config *zkSyncDepositConfig) error {
	if strings.Contains(config.NodeAddress, "zksync.io") {
		return errs.New("ZkSync deposit is an L1 transaction. Please defined Ethereum RPC endpoint")
	}

	if config.ZkNodeAddress == "" {
		return errs.New("Please specify both zkSync and ethereum rpc API address.")
	}

	spenderKey, spenderAddress, err := loadETHKey(config.SpenderKeyPath, "spender")
	if err != nil {
		return err
	}

	account, err := convertAddress(config.Account, "account")
	if err != nil {
		return err
	}

	amount, err := convertInt(config.Amount, 0, "amount")
	if err != nil {
		return err
	}

	chainID, err := convertInt(config.ChainID, 0, "chain-id")
	if err != nil {
		return err
	}

	client, err := dialNode(config.NodeAddress)
	if err != nil {
		return err
	}
	defer client.Close()

	ethBalance, err := client.BalanceAt(config.Ctx, spenderAddress, nil)
	if err != nil {
		return errs.Wrap(err)
	}

	zkSyncAddress, err := getZkSyncContract(config.Ctx, config.ZkNodeAddress)
	if err != nil {
		return err
	}

	tokenAddress, err := getStorjContract(config.Ctx, config.ZkNodeAddress)
	if err != nil {
		return err
	}

	tokenContract, err := contract.NewToken(common.HexToAddress(tokenAddress), client)
	if err != nil {
		return errs.Wrap(err)
	}

	zkSyncContract, err := contract.NewZkSyncTransactor(common.HexToAddress(zkSyncAddress), client)
	if err != nil {
		return errs.Wrap(err)
	}

	opts, err := bind.NewKeyedTransactorWithChainID(spenderKey, chainID)
	if err != nil {
		return errs.New("unable to obtain keyed transactor: %+v\n", err)
	}
	opts.Context = config.Ctx

	symbol, err := tokenContract.Symbol(nil)
	if err != nil {
		return errs.Wrap(err)
	}

	dec, err := tokenContract.Decimals(nil)
	if err != nil {
		return errs.Wrap(err)
	}
	storjDecimals := int32(dec.Int64())

	balance, err := tokenContract.BalanceOf(nil, spenderAddress)
	if err != nil {
		return errs.Wrap(err)
	}

	allowance, err := tokenContract.Allowance(nil, spenderAddress, common.HexToAddress(zkSyncAddress))
	if err != nil {
		return errs.Wrap(err)
	}

	estimatedGasPrice, err := setGasCap(config.Ctx, client, opts, config.GasTipCap)
	if err != nil {
		return err
	}

	fmt.Printf("Chain ID               : %s\n", chainID)
	fmt.Printf("Token contract         : %s\n", tokenAddress)
	fmt.Printf("Token symbol:          : %s\n", symbol)
	fmt.Printf("Token balance:         : %s\n", printToken(balance, storjDecimals, symbol))
	fmt.Printf("ETH balance (for fees) : %s\n", printToken(ethBalance, 18, "ETH"))
	fmt.Printf("Allowance:             : %s\n", printToken(allowance, storjDecimals, symbol))
	fmt.Printf("ZkSync contract (L1)   : %s\n", zkSyncAddress)
	fmt.Printf("From (L1)              : %s\n", spenderAddress)
	fmt.Printf("To (L2)                : %s\n", account)
	fmt.Printf("Amount                 : %s\n", printToken(amount, storjDecimals, symbol))
	fmt.Printf("Estimated gas cost     : %s\n", printToken(new(big.Int).Mul(estimatedGasPrice, big.NewInt(111111)), 18, "ETH"))

	if balance.Cmp(amount) < 0 {
		return errs.New("Not enough balance")
	}

	if allowance.Cmp(amount) < 0 {
		return errs.New("Not enough allowance")
	}

	if err := promptConfirm("Deposit"); err != nil {
		return err
	}

	tx, err := zkSyncContract.DepositERC20(opts, common.HexToAddress(tokenAddress), amount, account)
	if err != nil {
		return errs.Wrap(err)
	}

	if err := waitForTransaction(config.Ctx, client, tx.Hash()); err != nil {
		return err
	}

	fmt.Println("Deposited!")
	return nil
}

func printToken(balance *big.Int, dec int32, symbol string) string {
	return fmt.Sprintf("%s (%s %s)", balance, decimal.NewFromBigInt(balance, -1*dec), symbol)
}

func getZkSyncContract(ctx context.Context, nodeAddress string) (string, error) {
	res := struct {
		Result struct {
			Contract string
		}
	}{}
	err := zkSyncRestAPI(ctx, nodeAddress, "/api/v0.2/config", &res)
	if err != nil {
		return "", err
	}
	return res.Result.Contract, nil
}

func getStorjContract(ctx context.Context, nodeAddress string) (string, error) {
	res := struct {
		Result struct {
			Address string
		}
	}{}
	err := zkSyncRestAPI(ctx, nodeAddress, "/api/v0.2/tokens/STORJ", &res)
	if err != nil {
		return "", err
	}
	return res.Result.Address, nil
}

func zkSyncRestAPI(ctx context.Context, nodeAddress string, path string, value interface{}) error {
	if nodeAddress[len(nodeAddress)-1] != '/' {
		nodeAddress += "/"
	}
	if path[0] == '/' {
		path = path[1:]
	}

	url := nodeAddress + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return errs.Wrap(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errs.Wrap(err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		return errs.New("HTTP request to ZkSync api %s is failed: %d %v", url, resp.StatusCode, err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errs.Wrap(err)
	}

	err = json.Unmarshal(body, &value)
	if err != nil {
		return errs.Wrap(err)
	}
	return nil
}
