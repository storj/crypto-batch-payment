package contract

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"storj.io/crypto-batch-payment/pkg/ethtest"
)

func TestDeployToken(t *testing.T) {
	alice := ethtest.NewAccount()
	bob := ethtest.NewAccount()

	initialBalance := int64(133700000)
	network, client := ethtest.NewNetworkAndClient(t,
		ethtest.WithAccount(alice, big.NewInt(initialBalance)),
		ethtest.WithAccount(bob, big.NewInt(initialBalance)),
	)

	networkID := big.NewInt(int64(1337))
	gasPrice := big.NewInt(1)

	initialSupply, ok := new(big.Int).SetString("100000000000", 10)
	require.True(t, ok)

	// Deploy the initial token contract with Alice
	auth, err := bind.NewKeyedTransactorWithChainID(alice.Key, networkID)
	require.NoError(t, err)
	auth.GasLimit = 1500000
	auth.GasPrice = gasPrice
	_, _, contract, err := DeployToken(
		auth,
		client,
		alice.Address,
		"Storj",
		"STORJ",
		initialSupply,
		big.NewInt(8),
	)
	require.NoError(t, err)

	// Commit the token contract
	network.Commit()

	// Assert the initial supply
	totalSupply, err := contract.TotalSupply(nil)
	require.NoError(t, err)
	require.Equal(t, initialSupply, totalSupply)

	// Assert Alice's initial token balance
	balance, err := contract.BalanceOf(nil, alice.Address)
	require.NoError(t, err)
	require.Equal(t, initialSupply, balance)

	// Send some token to Bob
	auth, err = bind.NewKeyedTransactorWithChainID(alice.Key, networkID)
	require.NoError(t, err)
	auth.Nonce = big.NewInt(1)
	gasRequiredForTransfer := uint64(47196)
	auth.GasLimit = 25000 + gasRequiredForTransfer
	auth.GasPrice = gasPrice
	tx, err := contract.Transfer(auth, bob.Address, big.NewInt(1e11))
	require.NoError(t, err)

	// Commit the transfer
	network.Commit()

	// Assert the transaction was successful and that the cost was as expected
	rcpt, err := client.TransactionReceipt(context.Background(), tx.Hash())
	require.NoError(t, err)
	require.NotEqual(t, types.ReceiptStatusFailed, rcpt.Status)
	require.Equal(t, gasRequiredForTransfer, rcpt.GasUsed)

	aliceBalance, err := client.BalanceAt(context.Background(), alice.Address, nil)
	require.NoError(t, err)
	contractGasCost := int64(961413)
	require.Equal(t, initialBalance-gasPrice.Int64()*(contractGasCost+int64(gasRequiredForTransfer)), aliceBalance.Int64())
}
