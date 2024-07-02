package payer

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

// Transaction is a generic representation of a payment.
type Transaction struct {
	// Hash is the generic (unique) identifier of the transaction.
	Hash string

	// Nonce is the nonce from the group.
	Nonce uint64

	// Raw is the internal representation of transaction data.
	Raw any
}

// Payer is responsible for the final payment transfer.
type Payer interface {
	// Strings returns a string describing the payer type.
	String() string

	// NextNonce queries chain for the next available nonce value.
	NextNonce(ctx context.Context) (uint64, error)

	// CheckPreconditions checks if transaction can be initiated. If unmet preconditions are returned then pipeline will try again after a short sleep.
	CheckPreconditions(ctx context.Context) (unmet []string, err error)

	// GetTokenBalance returns with the available token balance in real value (with decimals).
	GetTokenBalance(ctx context.Context) (*big.Int, error)

	// GetTokenDecimals returns with the decimal precision of the token.
	GetTokenDecimals(ctx context.Context) (int32, error)

	// CreateRawTransaction creates the chain transaction which will be persisted to the db.
	CreateRawTransaction(ctx context.Context, log *zap.Logger, payouts []*pipelinedb.Payout, nonce uint64, storjPrice decimal.Decimal) (tx Transaction, from common.Address, err error)

	// SendTransaction submits the transaction created earlier.
	SendTransaction(ctx context.Context, log *zap.Logger, tx Transaction) error

	// CheckNonceGroup returns with the status of the submitted transactions.
	CheckNonceGroup(ctx context.Context, log *zap.Logger, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error)

	// PrintEstimate prints out additional information about the planned actions.
	PrintEstimate(ctx context.Context, remaining int64) error
}
