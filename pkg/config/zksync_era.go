package config

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum/common"

	"storj.io/crypto-batch-payment/pkg/zksyncera"
)

const (
	defaultZksyncEraChainID = 324
)

type ZkSyncEra struct {
	NodeAddress          string          `toml:"node_address"`
	SpenderKeyPath       Path            `toml:"spender_key_path"`
	ERC20ContractAddress common.Address  `toml:"erc20_contract_address"`
	ChainID              int             `toml:"chain_id"`
	PaymasterAddress     *common.Address `toml:"paymaster_address"`
	PaymasterPayload     HexString       `toml:"paymaster_payload"`
}

func (c ZkSyncEra) NewPayer(ctx context.Context) (_ Payer, err error) {
	// Check for required parameters
	if c.NodeAddress == "" {
		return nil, errors.New("node_address is not configured")
	}
	if c.ERC20ContractAddress == zeroAddress {
		return nil, errors.New("erc20_contract_address is not configured")
	}

	// Apply defaults
	if c.ChainID == 0 {
		c.ChainID = defaultZksyncEraChainID
	}

	spenderKey, _, err := loadSpenderKey(string(c.SpenderKeyPath))
	if err != nil {
		return nil, err
	}

	payer, err := zksyncera.NewPayer(
		c.ERC20ContractAddress,
		c.NodeAddress,
		spenderKey,
		c.ChainID,
		c.PaymasterAddress,
		c.PaymasterPayload)
	if err != nil {
		return nil, err
	}

	return &payerWrapper{
		Payer: payer,
	}, nil
}

func (c ZkSyncEra) NewAuditor(ctx context.Context) (_ Auditor, err error) {
	auditor, err := zksyncera.NewAuditor(c.NodeAddress)
	if err != nil {
		return nil, err
	}
	return &auditorWrapper{
		Auditor: auditor,
	}, nil
}
