package ethtest

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

type Account struct {
	Key     *ecdsa.PrivateKey
	Address common.Address
}

func NewAccount() *Account {
	key := NewKey()
	return &Account{
		Key:     key,
		Address: crypto.PubkeyToAddress(key.PublicKey),
	}
}

func (acc *Account) AddTxToBlock(tb testing.TB, block *core.BlockGen, to common.Address, amount *big.Int, gasLimit uint64, gasPrice *big.Int) {
	tx := types.NewTransaction(block.TxNonce(acc.Address), to, amount, gasLimit, gasPrice, nil)
	signer := types.MakeSigner(params.AllEthashProtocolChanges, block.Number(), block.Timestamp())
	tx, err := types.SignTx(tx, signer, acc.Key)
	require.NoError(tb, err)
	block.AddTx(tx)
}
