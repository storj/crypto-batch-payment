package ethtest

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

type networkConfig struct {
	alloc     core.GenesisAlloc
	numBlocks int
	generate  func(int, *core.BlockGen)
}

type NetworkOption func(c *networkConfig)

func WithAccount(account *Account, balance *big.Int) NetworkOption {
	return func(c *networkConfig) {
		c.alloc[account.Address] = core.GenesisAccount{Balance: balance}
	}
}

func WithBlocks(numBlocks int, generate func(int, *core.BlockGen)) NetworkOption {
	return func(c *networkConfig) {
		c.numBlocks = numBlocks
		c.generate = generate
	}
}

type Network struct {
	tb         testing.TB
	db         ethdb.Database
	engine     *ethash.Ethash
	node       *node.Node
	ethservice *eth.Ethereum
	ethpriv    *eth.PrivateDebugAPI
	clients    []*rpc.Client
}

func NewNetworkAndClient(tb testing.TB, opts ...NetworkOption) (*Network, *ethclient.Client) {
	network := NewNetwork(tb, opts...)
	return network, network.NewClient()
}

func NewNetwork(tb testing.TB, opts ...NetworkOption) *Network {
	config := &networkConfig{
		alloc: make(core.GenesisAlloc),
	}
	for _, opt := range opts {
		opt(config)
	}

	network := &Network{
		tb:     tb,
		db:     rawdb.NewMemoryDatabase(),
		engine: ethash.NewFaker(),
	}

	init := false
	defer func() {
		if !init {
			network.Close()
		}
	}()

	// generate the genesis block and initialize a new node
	genesis := &core.Genesis{
		Config:    chainConfig,
		Alloc:     config.alloc,
		ExtraData: []byte("test genesis"),
		Timestamp: 9000,
		GasLimit:  16 * 1024 * 1024,
		BaseFee:   big.NewInt(1),
	}

	var err error
	network.node, err = node.New(&node.Config{})
	require.NoError(tb, err)
	ethConfig := &eth.Config{Genesis: genesis}
	ethConfig.Ethash.PowMode = ethash.ModeFake
	network.ethservice, err = eth.New(network.node, ethConfig)
	require.NoError(tb, err)
	require.NoError(tb, network.node.Start(), "unable to start test node")

	network.insertBlocks(genesis.ToBlock(network.db), config.numBlocks, config.generate)

	network.ethpriv = eth.NewPrivateDebugAPI(network.ethservice)

	init = true
	return network
}

func (network *Network) Close() {
	for _, client := range network.clients {
		client.Close()
	}
	_ = network.node.Close()
}

func (network *Network) NewClient() *ethclient.Client {
	client, _ := network.node.Attach()
	network.clients = append(network.clients, client)
	return ethclient.NewClient(client)
}

func (network *Network) SetTxPoolGasPrice(price int64) {
	network.tb.Logf("Setting txpool gas price to %d", price)
	network.ethservice.TxPool().SetGasPrice(big.NewInt(price))
}

func (network *Network) Commit(hashes ...common.Hash) {
	pending := network.ethservice.TxPool().Pending(false)

	filter := make(map[common.Hash]bool)
	for _, hash := range hashes {
		filter[hash] = true
	}

	for _, txs := range pending {
		for _, tx := range txs {
			if len(filter) > 0 && !filter[tx.Hash()] {
				continue
			}
			network.InsertBlocks(1, func(n int, block *core.BlockGen) {
				network.tb.Logf("committing transaction %#x to block %s", tx.Hash(), block.Number())
				block.AddTx(tx)
			})
		}
	}
}

func (network *Network) PendingTransactionCount() int {
	pending, queued := network.ethservice.TxPool().Stats()
	return pending + queued
}

func (network *Network) TraceTransaction(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) {
	raw, err := tracers.NewAPI(network.ethservice.APIBackend).TraceTransaction(ctx, hash, config)
	require.NoError(network.tb, err)
	j, err := json.MarshalIndent(raw, "", "  ")
	require.NoError(network.tb, err)
	err = ioutil.WriteFile(fmt.Sprintf("%x.tx.json", hash), j, 0644)
	require.NoError(network.tb, err)
}

func (network *Network) InsertBlocks(numBlocks int, generator func(int, *core.BlockGen)) {
	network.insertBlocks(network.ethservice.BlockChain().CurrentBlock(), numBlocks, generator)
}

func (network *Network) insertBlocks(parent *types.Block, numBlocks int, generator func(int, *core.BlockGen)) {

	blocks, _ := core.GenerateChain(
		chainConfig,
		parent,
		network.engine,
		network.db,
		numBlocks,
		generator,
	)

	_, err := network.ethservice.BlockChain().InsertChain(blocks)
	require.NoError(network.tb, err, "unable to import blocks")
}
