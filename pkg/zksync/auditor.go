package zksync

import (
	"context"

	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

type Auditor struct {
	client *ZkClient
}

func (a Auditor) CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	status, err := a.client.TxStatus(ctx, hash)
	if err != nil {
		return pipelinedb.TxFailed, err
	}
	return TxStateFromZkStatus(status), nil
}

func (a Auditor) CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	// TODO this is a double check based on current logic and later can be removed / simplified.
	status, err := a.client.TxStatus(ctx, hash)
	if err != nil {
		return pipelinedb.TxFailed, err
	}
	return TxStateFromZkStatus(status), nil
}

var (
	_ payer.Auditor = &Auditor{}
)

func NewAuditor(
	url string,
	chainID int) (*Auditor, error) {

	client, err := NewZkClient(nil, url)
	if err != nil {
		return nil, err
	}
	client.ChainID = chainID
	return &Auditor{
		client: &client,
	}, nil
}
