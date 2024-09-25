package main

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
	"github.com/spf13/cobra"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/eth"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
	"storj.io/crypto-batch-payment/pkg/zksyncera"
)

type PayerConfig struct {
	PayerType string

	ContractAddress string
	Owner           string

	MaxGas string

	GasTipCap string

	PaymasterAddress string
	PaymasterPayload string
}

func RegisterFlags(cmd *cobra.Command, config *PayerConfig) {
	cmd.Flags().StringVarP(
		&config.GasTipCap,
		"gas-tip-cap", "",
		"1000000000",
		"Gas tip cap, paid on top of the base gas.")
	cmd.Flags().StringVarP(
		&config.Owner,
		"owner", "",
		"",
		"Owner of the ERC20 token (spender if unset)")
	cmd.Flags().StringVarP(
		&config.ContractAddress,
		"contract", "",
		storjtoken.DefaultContractAddress.String(),
		"Address of the STORJ contract on the network")
	cmd.Flags().StringVarP(
		&config.MaxGas,
		"max-gas", "",
		"70"+"000"+"000"+"000",
		"Max gas price we're willing to consider in Wei (tip + base fee). Default: 70 GWei. Only applies to Eth type payment.")
	cmd.Flags().StringVarP(
		&config.PayerType,
		"type", "",
		payer.Eth.String(),
		"Type of the payment (eth,zksync-era,zksync,zkwithdraw,sim,polygon)")
	cmd.Flags().StringVarP(
		&config.PaymasterAddress,
		"paymaster-address", "",
		"",
		"Address of the paymaster to be used.")
	cmd.Flags().StringVarP(
		&config.PaymasterPayload,
		"paymaster-payload", "",
		"",
		"Payload for the paymaster to be used.")
}

func registerNodeAddress(cmd *cobra.Command, addr *string) {
	cmd.Flags().StringVarP(
		addr,
		"node-address", "",
		"/home/storj/.ethereum/geth.ipc",
		"Address of the ETH node to use")
}
func CreatePayer(ctx context.Context, log *zap.Logger, config PayerConfig, nodeAddress string, chain string, spenderKeyPath string) (paymentPayer payer.Payer, err error) {
	spenderKey, spenderAddress, err := loadETHKey(spenderKeyPath, "spender")
	if err != nil {
		return nil, err
	}
	var maxGas big.Int
	_, ok := maxGas.SetString(config.MaxGas, 10)
	if !ok {
		return nil, errs.New("invalid max gas setting")
	}

	owner := spenderAddress
	if config.Owner != "" {
		owner, err = convertAddress(config.Owner, "owner")
		if err != nil {
			return nil, err
		}
	}

	contractAddress, err := convertAddress(config.ContractAddress, "contract")
	if err != nil {
		return nil, err
	}
	chainID, err := convertInt(chain, 0, "chain-id")
	if err != nil {
		return nil, err
	}

	var gasTipCap *big.Int
	if config.GasTipCap != "" {
		gasTipCap = new(big.Int)
		_, ok = gasTipCap.SetString(config.GasTipCap, 10)
		if !ok {
			return nil, errs.New("invalid gas tip cap setting")
		}
		if gasTipCap.Cmp(big.NewInt(30*params.GWei)) > 0 {
			return nil, errs.New("Gas tip cap is too high. Please use value less than 30 gwei")
		}

		if gasTipCap.Cmp(big.NewInt(int64(100))) < 0 {
			return nil, errs.New("Gas tip cap is negligible. Please check if you really used wei unit (or set 0)")
		}
	}

	pt, err := payer.TypeFromString(config.PayerType)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	switch pt {
	case payer.Eth, payer.Polygon:
		var client *ethclient.Client
		client, err = ethclient.Dial(nodeAddress)
		if err != nil {
			return paymentPayer, errs.New("Failed to dial node %q: %v\n", nodeAddress, err)
		}
		defer client.Close()

		paymentPayer, err = eth.NewPayer(ctx,
			client,
			contractAddress,
			owner,
			spenderKey,
			chainID,
			gasTipCap,
			&maxGas,
		)
		if err != nil {
			return nil, errs.Wrap(err)
		}
	case payer.ZkSyncEra:
		var paymasterAddress *common.Address
		var paymasterPayload []byte
		if config.PaymasterAddress != "" {
			a := common.HexToAddress(config.PaymasterAddress)
			paymasterAddress = &a
			paymasterPayload = common.Hex2Bytes(config.PaymasterPayload)
		}
		paymentPayer, err = zksyncera.NewPayer(
			common.HexToAddress(config.ContractAddress),
			nodeAddress,
			spenderKey,
			int(chainID.Int64()),
			paymasterAddress,
			paymasterPayload)
		if err != nil {
			return nil, errs.Wrap(err)
		}
	case payer.Sim:
		paymentPayer, err = payer.NewSimPayer()
		if err != nil {
			return nil, errs.Wrap(err)
		}
	default:
		return nil, errs.New("unsupported payer type: %v", config.PayerType)
	}
	return paymentPayer, nil
}
