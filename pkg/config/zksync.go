package config

import (
	"context"
	"errors"
	"math/big"

	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/zksync"
)

type ZkSync struct {
	NodeAddress    string   `toml:"node_address"`
	SpenderKeyPath Path     `toml:"spender_key_path"`
	ChainID        int      `toml:"chain_id"`
	MaxFee         *big.Int `toml:"max_fee"`
}

func (c ZkSync) NewPayer(ctx context.Context) (_ Payer, err error) {
	// Check for required parameters
	if c.NodeAddress == "" {
		return nil, errors.New("zksync node_address is not configured")
	}

	// Apply defaults
	if c.ChainID == 0 {
		c.ChainID = defaultEthChainID
	}

	spenderKey, _, err := loadSpenderKey(string(c.SpenderKeyPath))
	if err != nil {
		return nil, err
	}

	zkPayer, err := zksync.NewPayer(
		ctx,
		c.NodeAddress,
		spenderKey,
		c.ChainID,
		false,
		c.MaxFee)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return &payerWrapper{
		Payer: zkPayer,
	}, nil
}

func (c ZkSync) NewAuditor(ctx context.Context) (_ Auditor, err error) {
	// Check for required parameters
	if c.NodeAddress == "" {
		return nil, errors.New("zksync node_address is not configured")
	}

	// Apply defaults
	if c.ChainID == 0 {
		c.ChainID = defaultEthChainID
	}

	zkAuditor, err := zksync.NewAuditor(c.NodeAddress, c.ChainID)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return &auditorWrapper{
		Auditor: zkAuditor,
	}, nil
}
