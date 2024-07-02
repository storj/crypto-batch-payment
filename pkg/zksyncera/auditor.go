package zksyncera

import (
	"context"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zeebo/errs"
	"github.com/zksync-sdk/zksync2-go/clients"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

var _ payer.Auditor = (*Auditor)(nil)

type Auditor struct {
	client clients.Client
}

func NewAuditor(url string) (*Auditor, error) {
	client, err := clients.Dial("https://mainnet.era.zksync.io")
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return &Auditor{client: client}, nil
}

// CheckTransactionState checks the transaction state of any transaction.
func (a *Auditor) CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	txDetails, err := a.client.TransactionDetails(ctx, common.HexToHash(hash))
	switch {
	case err == nil:
		return stateFromStatus(txDetails.Status)
	case errors.Is(err, ethereum.NotFound):
		return pipelinedb.TxDropped, nil
	default:
		return "", errs.Wrap(err)
	}
}

// CheckConfirmedTransactionState checks the state from the confirmation receipt.
func (a *Auditor) CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	txReceipt, err := a.client.TransactionReceipt(ctx, common.HexToHash(hash))
	switch {
	// Receipts aren't available for pending transactions. This function is
	// only called after CheckTransactionState and we know the transaction
	// exists so we don't have to accommodate "dropped" status here.
	case err == nil:
		batchDetails, err := a.client.L1BatchDetails(ctx, txReceipt.L1BatchNumber.ToInt())
		if err != nil {
			return "", errs.Wrap(err)
		}
		return stateFromStatus(batchDetails.Status)
	case errors.Is(err, ethereum.NotFound):
		return pipelinedb.TxPending, nil
	default:
		return "", errs.Wrap(err)
	}
}

func stateFromStatus(status string) (pipelinedb.TxState, error) {
	switch strings.ToLower(status) {
	case "pending", "included":
		return pipelinedb.TxPending, nil
	case "verified":
		return pipelinedb.TxConfirmed, nil
	case "failed":
		return pipelinedb.TxFailed, nil
	default:
		return "", errs.New("unknown zksync-era tx status %q", status)
	}
}
