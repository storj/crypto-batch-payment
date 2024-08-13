package config

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/eth"
)

const (
	defaultEthChainID = 1
)

type Eth struct {
	NodeAddress          string          `toml:"node_address"`
	SpenderKeyPath       Path            `toml:"spender_key_path"`
	ERC20ContractAddress common.Address  `toml:"erc20_contract_address"`
	ChainID              int             `toml:"chain_id"`
	Owner                *common.Address `toml:"owner"`
}

func (c *Eth) NewPayer(ctx context.Context) (_ Payer, err error) {
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
	spenderKey, spenderAddress, err := c.NewSpender()
	if err != nil {
		return nil, err
	}

	owner := spenderAddress
	if c.Owner != nil {
		owner = *c.Owner
	}

	client, err := c.NewClient()
	if err != nil {
		return nil, err
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
		c.ChainID,
	)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return &payerWrapper{
		Payer:     ethPayer,
		closeFunc: client.Close,
	}, nil
}

func (c *Eth) NewAuditor(ctx context.Context) (_ Auditor, err error) {
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

func (c *Eth) NewClient() (*ethclient.Client, error) {
	client, err := ethclient.Dial(c.NodeAddress)
	return client, errs.Wrap(err)
}

func (c *Eth) NewSpender() (*ecdsa.PrivateKey, common.Address, error) {
	return loadSpenderKey(string(c.SpenderKeyPath))
}
