package pipelinedb

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type TxState string

const (
	// TxPending represents a transactions that has not been mined but is
	// still in the txpool.
	TxPending TxState = "pending"

	// TxDropped represents a transaction that has not been mined and is
	// no longer in the txpool. Safe to retry.
	TxDropped TxState = "dropped"

	// TxFailed represents a transaction that has been either mined and failed or failed during the submit.
	// Not safe to retry.
	TxFailed TxState = "failed"

	// TxConfirmed represents a transaction that has been mined and confirmed.
	TxConfirmed TxState = "confirmed"
)

func TxStateFromString(s string) (TxState, bool) {
	switch TxState(s) {
	case TxPending:
		return TxPending, true
	case TxDropped:
		return TxDropped, true
	case TxFailed:
		return TxFailed, true
	case TxConfirmed:
		return TxConfirmed, true
	}
	return "", false
}

type TxStatus struct {
	Hash    string
	State   TxState
	Receipt *types.Receipt
}
