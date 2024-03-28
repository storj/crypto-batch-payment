package contract

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient/simulated"
	"github.com/stretchr/testify/require"

	"storj.io/crypto-batch-payment/pkg/ethtest"
)

var (
	ctx = context.Background()

	initialBalance, _ = new(big.Int).SetString("90000000000000000", 10)
)

func TestDeployToken(t *testing.T) {
	alice := ethtest.NewAccount()
	bob := ethtest.NewAccount()

	alloc := core.DefaultGenesisBlock().Alloc
	alloc[alice.Address] = types.Account{Balance: initialBalance}
	alloc[bob.Address] = types.Account{Balance: initialBalance}

	backend := simulated.NewBackend(alloc) //, simulated.WithBlockGasLimit(16*1024*1024))
	client := backend.Client()

	chainID, err := client.ChainID(ctx)
	require.NoError(t, err)

	// Initial STORJ supply
	initialSupply, ok := new(big.Int).SetString("100000000000", 10)
	require.True(t, ok)

	// Deploy the initial token contract with Alice
	auth, err := bind.NewKeyedTransactorWithChainID(alice.Key, chainID)
	require.NoError(t, err)
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

	requireBalance := func(address common.Address, want *big.Int) {
		got, err := contract.BalanceOf(nil, address)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}

	// Commit the token contract
	commitHash := backend.Commit()

	// Assert the initial supply
	totalSupply, err := contract.TotalSupply(&bind.CallOpts{
		BlockHash: commitHash,
	})
	require.NoError(t, err)
	require.Equal(t, initialSupply, totalSupply)

	// Assert Alice's initial token balance
	requireBalance(alice.Address, initialSupply)

	// Send some token to Bob
	amnt := big.NewInt(1e11)
	_, err = contract.Transfer(auth, bob.Address, amnt)
	require.NoError(t, err)

	// Commit the transfer
	backend.Commit()

	requireBalance(alice.Address, new(big.Int).Add(initialSupply, new(big.Int).Neg(amnt)))
	requireBalance(bob.Address, amnt)
}
