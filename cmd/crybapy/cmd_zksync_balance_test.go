package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_doZkSyncBalance(t *testing.T) {
	t.Skip("Rrequires new deployed contract to the new test chain")
	cfg := zkSyncBalanceConfig{
		Account: "0x712Ce0cBEe9423E414493542FfebF418C16c1C96",
		zkSyncConfig: &zkSyncConfig{
			rootConfig: &rootConfig{
				NodeAddress: "https://goerli-api.zksync.io",
				ChainID:     "4",
			},
		},
	}

	b := bytes.NewBufferString("")
	err := doZkSyncBalance(&cfg, b)
	require.NoError(t, err)
	require.Contains(t, b.String(), "STORJ")
}
