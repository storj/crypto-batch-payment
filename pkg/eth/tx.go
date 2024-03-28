package eth

import (
	"context"
	"errors"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/zeebo/errs"

	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

func GetTransactionInfo(ctx context.Context, client ethereum.TransactionReader, hash common.Hash) (pipelinedb.TxState, *types.Transaction, *types.Receipt, error) {
	tx, pending, err := client.TransactionByHash(ctx, hash)
	switch {
	case err == nil:
		if pending {
			return pipelinedb.TxPending, tx, nil, nil
		}
	case errors.Is(err, ethereum.NotFound):
		return pipelinedb.TxDropped, nil, nil, nil
	default:
		// Failed to issue RPC
		return "", nil, nil, errs.Wrap(err)
	}
	receipt, err := client.TransactionReceipt(ctx, hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return pipelinedb.TxPending, tx, nil, nil
		}
		return "", nil, nil, errs.Wrap(err)
	}
	return TxStateFromReceipt(receipt), tx, receipt, nil
}

func TxStateFromReceipt(receipt *types.Receipt) pipelinedb.TxState {
	if receipt.Status == types.ReceiptStatusSuccessful {
		return pipelinedb.TxConfirmed
	}
	return pipelinedb.TxFailed
}
