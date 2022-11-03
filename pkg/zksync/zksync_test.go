package zksync

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func Test_Nonce(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/jsrpc")
	client.ChainID = 4
	require.NoError(t, err)
	ctx := context.Background()
	nonce, err := client.GetNonce(ctx)
	require.NoError(t, err)
	require.Greater(t, nonce, uint64(0))
}

func Test_Fee(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/jsrpc")
	client.ChainID = 4
	require.NoError(t, err)
	ctx := context.Background()
	fee, err := client.GetFee(ctx, "Transfer", "0xbd9294F0232a84Da9602B9B13f1Fca173b7A7CA8", "STORJ")
	require.NoError(t, err)
	require.Greater(t, fee.Uint64(), uint64(10))
}

func Test_Transfer(t *testing.T) {
	if os.Getenv("TEST_WRITE") == "" {
		t.Skip("Skipping test which requires on-chain tx")
	}
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/jsrpc")
	client.ChainID = 4
	require.NoError(t, err)
	ctx := context.Background()
	receipt, err := client.Transfer(ctx,
		common.HexToAddress("0xcEE63e28E874eB80AF9eBCeC7b96F7cEbE3e92D8"),
		big.NewInt(10),
		big.NewInt(704000),
		Token{
			Decimals: 8,
			Symbol:   "STORJ",
			ID:       11,
		},
	)
	require.NoError(t, err)
	fmt.Println(receipt)
}

func Test_TransferTwoPhases(t *testing.T) {
	if os.Getenv("TEST_WRITE") == "" {
		t.Skip("Skipping test which requires on-chain tx")
	}
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/jsrpc")
	client.ChainID = 4
	require.NoError(t, err)
	ctx := context.Background()
	txs, err := client.CreateTransferTx(ctx,
		common.HexToAddress("0xcEE63e28E874eB80AF9eBCeC7b96F7cEbE3e92D8"),
		big.NewInt(10),
		big.NewInt(704000),
		Token{
			Decimals: 8,
			Symbol:   "STORJ",
			ID:       11,
		}, 10)
	require.NoError(t, err)
	hash, err := txs.Tx.Hash()
	require.NoError(t, err)
	require.Equal(t, hash, "50986a7189d6babbe3c3ca3ed61dce81611ad177f2d53e86d37e5993872b94f9")

	res, err := client.SubmitTransaction(ctx, txs)
	require.NoError(t, err)
	require.Equal(t, hash, res)
}

func Test_WithdrawTwoPhases(t *testing.T) {
	if os.Getenv("TEST_WRITE") == "" {
		t.Skip("Skipping test which requires on-chain tx")
	}
	pk, err := crypto.HexToECDSA(os.Getenv("TEST_PRIVATE_KEY"))
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/jsrpc")
	client.ChainID = 4
	require.NoError(t, err)
	ctx := context.Background()

	target := "0xB766AE0dd39F26bC5E8f60fC1C95566eCFE3851f"

	nonce, err := client.GetNonce(ctx)
	require.NoError(t, err)

	token, err := client.GetToken(ctx, "STORJ")
	require.NoError(t, err)

	fee, err := client.GetFee(ctx, "Withdraw", crypto.PubkeyToAddress(pk.PublicKey).Hex(), "STORJ")
	require.NoError(t, err)

	txs, err := client.CreateTx(ctx,
		"Withdraw",
		common.HexToAddress(target),
		big.NewInt(6270),
		fee,
		token,
		int(nonce))

	require.NoError(t, err)

	hash, err := txs.Tx.Hash()
	require.NoError(t, err)

	res, err := client.SubmitTransaction(ctx, txs)
	require.NoError(t, err)
	require.Equal(t, hash, res)
}

func Test_PrivateKeySeed(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/jsrpc")
	require.NoError(t, err)
	client.ChainID = 4

	seed, err := client.privateKeySeed()
	require.NoError(t, err)
	require.Equal(t, "286267a4e6b69668c3e5423f0098cb719aa844ae71ee1ad85b2562c37994bbb9539008e1b729af522bc4e11d0e57e06bee9dbcdff50f8d4083a61cc21e35065e1c", common.Bytes2Hex(seed))
}

func Test_PrivateKey(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/")
	require.NoError(t, err)
	client.ChainID = 4

	zkpk, err := client.zkPrivateKey()
	require.NoError(t, err)
	zkpub, err := zkpk.PublicKey()
	require.NoError(t, err)

	require.Equal(t, "931fcb506443c23cd4b23e49ce6a20a72e4826110bd3190a1ac02fdc45b7982d", zkpub.HexString())
}

func Test_Priv(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/")
	require.NoError(t, err)
	client.ChainID = 4

	seed, err := client.privateKeySeed()
	require.NoError(t, err)
	require.Equal(t, "286267a4e6b69668c3e5423f0098cb719aa844ae71ee1ad85b2562c37994bbb9539008e1b729af522bc4e11d0e57e06bee9dbcdff50f8d4083a61cc21e35065e1c", common.Bytes2Hex(seed))
}

func Test_token(t *testing.T) {
	storj := Token{
		Decimals: 8,
		Symbol:   "STORJ",
		ID:       11,
	}

	require.Equal(t, "0.12345678", storj.Format(big.NewInt(12345678)))
}

func Test_GetToken(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	checkToken(t, pk, "https://rinkeby-api.zksync.io/", "STORJ", 11, 8)
	checkToken(t, pk, "https://api.zksync.io/", "STORJ", 24, 8)
}

func checkToken(t *testing.T, pk *ecdsa.PrivateKey, api string, symbol string, id int, decimals int32) {
	client, err := NewZkClient(pk, api)
	require.NoError(t, err)
	client.ChainID = 4

	ctx := context.Background()
	token, err := client.GetToken(ctx, symbol)
	require.NoError(t, err)
	require.Equal(t, symbol, token.Symbol)
	require.Equal(t, id, token.ID)
	require.Equal(t, decimals, token.Decimals)
}

func Test_ethereumSignature(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/")
	require.NoError(t, err)
	client.ChainID = 4

	tx := Tx{
		Type:      "Transfer",
		AccountID: 236843,
		From:      common.HexToAddress("0xA72fD7554c9aC2c89D241f4A4Fd0351A4976c835"),
		To:        common.HexToAddress("0xcEE63e28E874eB80AF9eBCeC7b96F7cEbE3e92D8"),
		Token: Token{
			Symbol:   "STORJ",
			ID:       11,
			Decimals: 8,
		},
		Amount: big.NewInt(2300000000),
		Fee:    big.NewInt(621000),
		Nonce:  2,
	}
	signature, err := client.ethereumSignature(tx)
	require.NoError(t, err)
	require.Equal(t, "55412b1906b455ab43baa47ebdae4ea2b3109d0ac8b19ef885441b155e0fe714670e79facd04468c79ed5049ee5b25dda3fba9077998fcb190cf4942afa560851c",
		common.Bytes2Hex(signature))
}

func Test_zkSignature(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/")
	require.NoError(t, err)
	client.ChainID = 4

	tx := Tx{
		Type:      "Transfer",
		AccountID: 236843,
		From:      common.HexToAddress("0xA72fD7554c9aC2c89D241f4A4Fd0351A4976c835"),
		To:        common.HexToAddress("0xcEE63e28E874eB80AF9eBCeC7b96F7cEbE3e92D8"),
		Token: Token{
			Symbol:   "STORJ",
			ID:       11,
			Decimals: 8,
		},
		Amount: big.NewInt(9990000),
		Fee:    big.NewInt(704000),
		Nonce:  5,
	}
	signature, err := client.zkSignature(tx)
	require.NoError(t, err)
	require.Equal(t, "ed35d342131e73b644456866188098b6696d3ac40e3da04d78e9348ad74d2494efb7e13e957b434183803e97da00a2d14d45c3dbceae0dcd5380d0383fe3b405", common.Bytes2Hex(signature))
}

func Test_txHashTransfer(t *testing.T) {
	tx := Tx{
		Type:      "Transfer",
		AccountID: 945169,
		From:      common.HexToAddress("0x303edcd8dbe1607fe512d45cc15d3e41fa4db44b"),
		To:        common.HexToAddress("0xf993b7a9e7a80d3d4f54c8f996fb247d416313c0"),
		Amount:    big.NewInt(156148485580),
		Fee:       big.NewInt(5170000),
		Nonce:     23523,
		Token: Token{
			Decimals: 8,
			Symbol:   "STORJ",
			ID:       24,
		},
	}
	base, err := tx.encodeForHash()
	require.NoError(t, err)
	require.Equal(t, "FA01000E6C11303EDCD8DBE1607FE512D45CC15D3E41FA4DB44BF993B7A9E7A80D3D4F54C8F996FB247D416313C0000000187456F5C5C140A400005BE30000000000000000FFFFFFFFFFFFFFFF", strings.ToUpper(hex.EncodeToString(base)))
	hash, err := tx.Hash()
	require.NoError(t, err)
	require.Equal(t, "0xf4df3bd1abe68e16b534417f60a7451544e56e2820e1a0a0d06b1bdcfab56b87", hash)
}

func Test_txHashWithdraw(t *testing.T) {
	tx := Tx{
		Type:      "Withdraw",
		AccountID: 362737,
		From:      common.HexToAddress("0x712ce0cbee9423e414493542ffebf418c16c1c96"),
		To:        common.HexToAddress("0xb766ae0dd39f26bc5e8f60fc1c95566ecfe3851f"),
		Amount:    big.NewInt(6270),
		Fee:       big.NewInt(1938000),
		Nonce:     15,
		Token: Token{
			Decimals: 8,
			Symbol:   "STORJ",
			ID:       11,
		},
	}
	hash, err := tx.Hash()
	require.NoError(t, err)
	require.Equal(t, "0xab4555f8111f18a4a8c449094218ab8853e3464bf89500baed7739d6b8f852bd", hash)
}

func Test_TxStatus(t *testing.T) {
	pk, err := crypto.HexToECDSA("add336fb48ab11ee615f67295e80a692e6ea03367f7585ffc40651e65059adf2")
	require.NoError(t, err)

	client, err := NewZkClient(pk, "https://rinkeby-api.zksync.io/")
	client.ChainID = 4
	require.NoError(t, err)
	ctx := context.Background()
	status, err := client.TxStatus(ctx, "0xb06bbafb874b1c8e8f62d1b635ba5fded83c147869566f10e0d855127a087caa")
	require.NoError(t, err)
	require.True(t, status.Executed)
	require.True(t, status.Success)
	require.Equal(t, 67181, status.Block.BlockNumber)
}

func Test_TokenFormat(t *testing.T) {
	token := Token{
		Symbol:   "STORJ",
		Decimals: 8,
		ID:       11,
	}
	require.Equal(t, "1.0", token.Format(big.NewInt(100000000)))
	require.Equal(t, "0.83", token.Format(big.NewInt(83000000)))
}
