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
	_ payer.Auditor = &EthAuditor{}
)

// EthAuditor audits eth transactions.
type EthAuditor struct {
	client *ethclient.Client
}

func NewEthAuditor(nodeAddress string) (*EthAuditor, error) {
	client, err := ethclient.Dial(nodeAddress)
	if err != nil {
		return nil, errs.New("Failed to dial node %q: %v\n", nodeAddress, err)
	}

	return &EthAuditor{
		client: client,
	}, nil
}

func (e *EthAuditor) CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	txHash, err := batchpayment.HashFromString(hash)
	if err != nil {
		return pipelinedb.TxFailed, err
	}
	state, _, _, err := GetTransactionInfo(ctx, e.client, txHash)
	return state, err

}

func (e *EthAuditor) CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
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

func (e *EthAuditor) Close() {
	e.client.Close()
}
