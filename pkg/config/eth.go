package config

import (
	"context"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/eth"
)

const (
	defaultEthChainID = 1
	defaultMaxGas     = "70_000_000_000"
	defaultGasTipCap  = "1_000_000_000"
)

type Eth struct {
	NodeAddress          string          `toml:"node_address"`
	SpenderKeyPath       Path            `toml:"spender_key_path"`
	ERC20ContractAddress common.Address  `toml:"erc20_contract_address"`
	ChainID              int             `toml:"chain_id"`
	Owner                *common.Address `toml:"owner"`
	MaxGas               *big.Int        `toml:"max_gas"`
	GasTipCap            *big.Int        `toml:"gas_tip_cap"`
}

func (c Eth) NewPayer(ctx context.Context) (_ Payer, err error) {
	// Check for required parameters
	if c.NodeAddress == "" {
		return nil, errors.New("node_address is not configured")
	}
	if c.ERC20ContractAddress == zeroAddress {
		return nil, errors.New("erc20_contract_address is not configured")
	}

	// Apply defaults
	if c.ChainID == 0 {
		c.ChainID = defaultEthChainID
	}
	if c.MaxGas == nil {
		c.MaxGas, _ = new(big.Int).SetString(defaultMaxGas, 0)
	}
	if c.GasTipCap == nil {
		c.GasTipCap, _ = new(big.Int).SetString(defaultGasTipCap, 0)
	}

	spenderKey, spenderAddress, err := loadSpenderKey(string(c.SpenderKeyPath))
	if err != nil {
		return nil, err
	}

	owner := spenderAddress
	if c.Owner != nil {
		owner = *c.Owner
	}

	client, err := ethclient.Dial(c.NodeAddress)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	ethPayer, err := eth.NewPayer(ctx,
		client,
		c.ERC20ContractAddress,
		owner,
		spenderKey,
		big.NewInt(int64(c.ChainID)),
		c.GasTipCap,
		c.MaxGas,
	)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return &payerWrapper{
		Payer:     ethPayer,
		closeFunc: client.Close,
	}, nil
}

func (c Eth) NewAuditor(ctx context.Context) (_ Auditor, err error) {
	// Check for required parameters
	if c.NodeAddress == "" {
		return nil, errors.New("node_address is not configured")
	}

	ethAuditor, err := eth.NewAuditor(c.NodeAddress)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return ethAuditor, nil
}
