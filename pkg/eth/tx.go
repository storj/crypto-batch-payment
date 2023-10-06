package eth

import (
	"context"
	"errors"
	"reflect"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/zeebo/errs"
	"github.com/zeebo/sudo"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

func GetTransactionInfo(ctx context.Context, client *ethclient.Client, hash common.Hash) (pipelinedb.TxState, *types.Transaction, *types.Receipt, error) {
	type rpcTransaction struct {
		tx          *types.Transaction
		BlockNumber *string         `json:"blockNumber,omitempty"`
		BlockHash   *common.Hash    `json:"blockHash,omitempty"`
		From        *common.Address `json:"from,omitempty"`
	}

	var json *rpcTransaction
	var tx *types.Transaction
	var pending bool
	err := sudo.Sudo(reflect.ValueOf(*client).FieldByName("c")).Interface().(*rpc.Client).CallContext(ctx, &json, "eth_getTransactionByHash", hash)
	if err == nil {
		if json == nil {
			err = ethereum.NotFound
		} else {
			pending = json.BlockNumber == nil
		}
	}

	//tx, pending, err := client.TransactionByHash(ctx, hash)
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
	var receipt *types.Receipt
	err = sudo.Sudo(reflect.ValueOf(*client).FieldByName("c")).Interface().(*rpc.Client).CallContext(ctx, &receipt, "eth_getTransactionReceipt", hash)
	if err == nil && receipt == nil {
		err = ethereum.NotFound
	}
	// receipt, err := client.TransactionReceipt(ctx, hash)
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
