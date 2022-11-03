package payer

import (
	"context"

	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

// Auditor helps to validate transaction created by the appropriate payer.
type Auditor interface {

	// CheckTransactionState checks the transaction state of any transaction.
	CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error)

	// CheckConfirmedTransactionState checks the state from the confirmation receipt.
	CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error)
}
