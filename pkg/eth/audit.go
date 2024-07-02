package eth

import (
	"context"

	"github.com/zeebo/errs"

	"github.com/ethereum/go-ethereum/ethclient"

	batchpayment "storj.io/crypto-batch-payment/pkg"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

var (
	_ payer.Auditor = &Auditor{}
)

// Auditor audits eth transactions.
type Auditor struct {
	client *ethclient.Client
}

func NewAuditor(nodeAddress string) (*Auditor, error) {
	client, err := ethclient.Dial(nodeAddress)
	if err != nil {
		return nil, errs.New("Failed to dial node %q: %v\n", nodeAddress, err)
	}

	return &Auditor{
		client: client,
	}, nil
}

func (e *Auditor) CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	txHash, err := batchpayment.HashFromString(hash)
	if err != nil {
		return pipelinedb.TxFailed, err
	}
	state, _, _, err := GetTransactionInfo(ctx, e.client, txHash)
	return state, err

}

func (e *Auditor) CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	txHash, err := batchpayment.HashFromString(hash)
	if err != nil {
		return pipelinedb.TxFailed, err
	}
	receipt, err := e.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return pipelinedb.TxFailed, err
	}

	return TxStateFromReceipt(receipt), nil
}

func (e *Auditor) Close() {
	e.client.Close()
}
