package payer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

// Transaction is a generic representation of a payment.
type Transaction struct {
	// Hash is the generic (unique) identifier of the transaction.
	Hash string

	// Nonce is the nonce from the group.
	Nonce uint64

	// EstimatedGasLimit is an estimate of the amount of gas needed for the
	// transaction. It is usually higher than the real amount used.
	EstimatedGasLimit uint64

	// EstimatedGasFeeCap is an estimate of the most per-gas fee that the
	// transaction will incur. The actual fee will likely be smaller than this.
	EstimatedGasFeeCap *big.Int

	// Raw is the internal representation of transaction data.
	Raw any
}

type TransactionParams struct {
	// Nonce is the nonce to use for the transaction
	Nonce uint64

	// Payee is the recipient of the tokens
	Payee common.Address

	// Tokens is the number of tokens to send
	Tokens *big.Int
}

type GasInfo struct {
	// GasLimit is the gas limit which is typically higher than necessary but
	// provides a meaningful cap on fees.
	GasLimit uint64

	// GasFeeCap is the EIP-1559 max fee, in WEI.
	GasFeeCap *big.Int

	// GasTipCap is the EIP-1559 priority fee, in WEI.
	GasTipCap *big.Int
}

// Payer is responsible for the final payment transfer.
type Payer interface {
	// Strings returns a string describing the payer type.
	String() string

	// ChainID is the ethereum chain ID this payer targets.
	ChainID() int

	// Decimals returns with the decimal precision of the token.
	Decimals() int32

	// NextNonce queries chain for the next available nonce value.
	NextNonce(ctx context.Context) (uint64, error)

	// GetGasInfo returns estimated gas limits and caps.
	GetGasInfo(context.Context) (GasInfo, error)

	// GetETHBalance returns the available ETH balance in WEI.
	GetETHBalance(ctx context.Context) (*big.Int, error)

	// GetTokenBalance returns with the available token balance in real value (with decimals).
	GetTokenBalance(ctx context.Context) (*big.Int, error)

	// CreateRawTransaction creates the chain transaction which will be persisted to the db.
	CreateRawTransaction(ctx context.Context, log *zap.Logger, params TransactionParams) (tx Transaction, from common.Address, err error)

	// SendTransaction submits the transaction created earlier.
	SendTransaction(ctx context.Context, log *zap.Logger, tx Transaction) error

	// CheckNonceGroup returns with the status of the submitted transactions.
	CheckNonceGroup(ctx context.Context, log *zap.Logger, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error)

	// PrintEstimate prints out additional information about the planned actions.
	PrintEstimate(ctx context.Context, remaining int64) error
}
