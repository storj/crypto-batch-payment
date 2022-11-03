package zksync

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
	zkscrypto "github.com/zksync-sdk/zksync-sdk-go"
)

const feeMantissa = 11
const amountMantissa = 35
const amountExp = 5
const feeExp = 5
const syncTxPrefix = "sync-tx:"

type ZkClient struct {
	rpcURL  string
	httpURL string

	privateKey  *ecdsa.PrivateKey
	ChainID     int
	accountInfo *AccountInfo
}

func NewZkClient(pk *ecdsa.PrivateKey, url string) (ZkClient, error) {
	if url[len(url)-1] == '/' {
		url = url[:len(url)-1]
	}
	return ZkClient{
		rpcURL:     url + "/jsrpc",
		httpURL:    url + "/api/v1",
		privateKey: pk,
		ChainID:    1,
	}, nil
}

type Token struct {
	Symbol   string
	Decimals int32
	ID       int
}

func (t Token) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%d", t.ID)), nil
}

type Tx struct {
	Type      string         `json:"type"`
	AccountID int            `json:"accountId"`
	From      common.Address `json:"from"`
	To        common.Address `json:"to"`
	Token     Token          `json:"token"`
	Amount    *big.Int       `json:"amount"`
	Fee       *big.Int       `json:"fee"`
	Nonce     int            `json:"nonce"`
	Signature TxSignature    `json:"signature"`
}

type TxSignature struct {
	PubKey    string `json:"pubKey"`
	Signature string `json:"signature"`
}
type Signature struct {
	Type      string `json:"type"`
	Signature string `json:"signature"`
}
type TxWithEthSignature struct {
	Tx        Tx
	Signature Signature
}

const EtherSignMessage = "\x19Ethereum Signed Message:\n"
const ZkSyncMessage = "Access zkSync account.\n\nOnly sign this message for a trusted client!"

// GetTypeID returns with the internal byte representation of the transaction type (or 0 if unsupported).
func (tx Tx) GetTypeID() byte {
	switch tx.Type {
	case "Withdraw":
		// See: https://github.com/matter-labs/zksync/blob/b55ab5a7ec1fddd6c236980a54bc59babbe0076e/core/lib/types/src/tx/withdraw.rs#L62
		return byte(3)
	case "Transfer":
		// See: https://github.com/matter-labs/zksync/blob/b55ab5a7ec1fddd6c236980a54bc59babbe0076e/core/lib/types/src/tx/transfer.rs#L60
		return byte(5)
	default:
		return byte(0)
	}

}

// Hash returns with the tx hash calculated from the struct values.
func (tx Tx) Hash() (string, error) {
	encoded, err := tx.encodeForHash()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return "0x" + hex.EncodeToString(sum[:]), nil
}

func (tx Tx) encode() ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.Write([]byte{tx.GetTypeID()})
	if err != nil {
		return nil, err
	}

	amountBytes := reverseBytes(packBigInt(tx.Amount, amountExp, amountMantissa))
	if tx.Type == "Withdraw" {
		// L1->L2 withdraw should use full 16 bit for the amounts instead of packing
		// see: https://github.com/matter-labs/zksync/blob/b55ab5a7ec1fddd6c236980a54bc59babbe0076e/sdk/zksync.js/src/utils.ts#L694
		amountBytes = make([]byte, 16)
		tx.Amount.FillBytes(amountBytes)
	}

	for _, value := range []interface{}{
		uint32(tx.AccountID),
		tx.From.Bytes(),
		tx.To.Bytes(),
		uint16(tx.Token.ID),
		amountBytes,
		reverseBytes(packBigInt(tx.Fee, feeExp, feeMantissa)),
		uint32(tx.Nonce),
	} {
		err = binary.Write(buf, binary.BigEndian, value)
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (tx Tx) encodeForHash() ([]byte, error) {
	buf := new(bytes.Buffer)
	_, err := buf.Write([]byte{255 - tx.GetTypeID(), 1})
	if err != nil {
		return nil, err
	}
	amountBytes := reverseBytes(packBigInt(tx.Amount, amountExp, amountMantissa))
	if tx.Type == "Withdraw" {
		amountBytes = make([]byte, 16)
		tx.Amount.FillBytes(amountBytes)
	}
	for _, value := range []interface{}{
		uint32(tx.AccountID),
		tx.From.Bytes(),
		tx.To.Bytes(),
		uint32(tx.Token.ID),
		amountBytes,
		reverseBytes(packBigInt(tx.Fee, 5, 11)),
		uint32(tx.Nonce),
	} {
		err = binary.Write(buf, binary.BigEndian, value)
		if err != nil {
			return nil, err
		}
	}

	// bytes represent time range added in
	// https://github.com/matter-labs/zksync/commit/1c23f6e0
	m, err := hex.DecodeString("0000000000000000FFFFFFFFFFFFFFFF")
	if err != nil {
		return nil, err
	}
	return append(buf.Bytes(), m...), nil
}

func (t *Token) Format(value *big.Int) string {
	result := decimal.NewFromBigInt(value, -t.Decimals).String()
	// note: 1 should be represented as 1.0 in the base string of eth signature
	if !strings.Contains(result, ".") {
		result = result + ".0"
	}
	return result
}

func signHash(data []byte) common.Hash {
	msg := fmt.Sprintf(EtherSignMessage+"%d%s", len(data), data)
	return crypto.Keccak256Hash([]byte(msg))
}

func (c *ZkClient) privateKeySeed() ([]byte, error) {
	dataToSign := ZkSyncMessage
	if c.ChainID != 1 {
		dataToSign += fmt.Sprintf("\nChain ID: %d.", c.ChainID)
	}
	signature, err := c.signMessage(dataToSign, true)
	return signature, err
}

func (c *ZkClient) signMessage(message string, pad bool) ([]byte, error) {
	signHash := signHash([]byte(message))
	signature, err := crypto.Sign(signHash.Bytes(), c.privateKey)
	if pad {
		signature[len(signature)-1] = signature[len(signature)-1] + 27
	}
	return signature, err
}

func (c *ZkClient) zkPrivateKey() (*zkscrypto.PrivateKey, error) {
	seed, err := c.privateKeySeed()
	if err != nil {
		return nil, err
	}
	return zkscrypto.NewPrivateKey(seed)
}

func (c *ZkClient) ethereumSignature(tx Tx) ([]byte, error) {
	humanReadableTxInfo := fmt.Sprintf(
		"%s %s %s\n"+
			"To: %s\n"+
			"Nonce: %d\n"+
			"Fee: %s %s\n"+
			"Account Id: %d",
		tx.Type,
		tx.Token.Format(tx.Amount),
		tx.Token.Symbol,
		strings.ToLower(tx.To.String()),
		tx.Nonce,
		tx.Token.Format(tx.Fee),
		tx.Token.Symbol,
		tx.AccountID,
	)
	return c.signMessage(humanReadableTxInfo, true)

}

func (c *ZkClient) Transfer(ctx context.Context, to common.Address, amount *big.Int, fee *big.Int, token Token) (string, error) {
	err := c.fillAccountInfo(ctx)
	if err != nil {
		return "", err
	}
	nonce := int(c.accountInfo.Committed.Nonce)
	txs, err := c.CreateTransferTx(ctx, to, amount, fee, token, nonce)
	if err != nil {
		return "", err
	}
	return c.SubmitTransaction(ctx, txs)
}

func (c *ZkClient) SubmitTransaction(ctx context.Context, txs TxWithEthSignature) (string, error) {
	client, err := rpc.DialContext(ctx, c.rpcURL)
	if err != nil {
		return "", err
	}
	defer client.Close()

	hash := ""
	err = client.CallContext(ctx, &hash, "tx_submit", txs.Tx, txs.Signature)
	if err != nil {
		return "", err
	}

	// our Canonical transaction format is pure 0xHEX
	hash = strings.TrimPrefix(hash, syncTxPrefix)
	if !strings.HasPrefix(hash, "0x") {
		hash = "0x" + hash
	}
	return hash, err
}

func (c *ZkClient) CreateTransferTx(ctx context.Context, to common.Address, amount *big.Int, fee *big.Int, token Token, nonce int) (TxWithEthSignature, error) {
	return c.CreateTx(ctx, "Transfer", to, amount, fee, token, nonce)
}

func (c *ZkClient) CreateTx(ctx context.Context, txType string, to common.Address, amount *big.Int, fee *big.Int, token Token, nonce int) (TxWithEthSignature, error) {
	pk, err := c.zkPrivateKey()
	if err != nil {
		return TxWithEthSignature{}, err
	}
	err = c.fillAccountInfo(ctx)
	if err != nil {
		return TxWithEthSignature{}, err
	}

	publicKey, err := pk.PublicKey()
	if err != nil {
		return TxWithEthSignature{}, err
	}
	from := crypto.PubkeyToAddress(c.privateKey.PublicKey)

	tx := Tx{
		AccountID: int(c.accountInfo.ID),
		Nonce:     nonce,
		Type:      txType,
		From:      from,
		To:        to,
		Amount:    amount,
		Fee:       fee,
		Token:     token,
	}
	zkSignature, err := c.zkSignature(tx)
	if err != nil {
		return TxWithEthSignature{}, err
	}
	tx.Signature = TxSignature{
		Signature: hex.EncodeToString(zkSignature),
		PubKey:    publicKey.HexString(),
	}

	ethSignature, err := c.ethereumSignature(tx)
	if err != nil {
		return TxWithEthSignature{}, err
	}
	ethereumSignature := Signature{
		Type:      "EthereumSignature",
		Signature: "0x" + hex.EncodeToString(ethSignature),
	}
	return TxWithEthSignature{
		Signature: ethereumSignature,
		Tx:        tx,
	}, nil
}

func (c *ZkClient) zkSignature(tx Tx) ([]byte, error) {

	pk, err := c.zkPrivateKey()
	if err != nil {
		return nil, err
	}
	encoded, err := tx.encode()
	if err != nil {
		return nil, err
	}
	signature, err := pk.Sign(encoded)
	if err != nil {
		return nil, err
	}
	result, err := hex.DecodeString(signature.HexString())
	if err != nil {
		return nil, err
	}
	return result, nil
}

type AccountInfo struct {
	ID        int64
	Address   string
	Committed AccountState
	Verified  AccountState
}
type AccountState struct {
	Balances map[string]string
	Nonce    uint64
}

func (c *ZkClient) fillAccountInfo(ctx context.Context) error {
	if c.accountInfo != nil {
		return nil
	}
	return c.refreshAccountInfo(ctx)

}

func (c *ZkClient) refreshAccountInfo(ctx context.Context) error {
	client, err := rpc.DialContext(ctx, c.rpcURL)
	if err != nil {
		return err
	}
	defer client.Close()

	addr := crypto.PubkeyToAddress(c.privateKey.PublicKey)
	result := AccountInfo{}
	err = client.CallContext(ctx, &result, "account_info", addr)
	if err != nil {
		return err
	}
	c.accountInfo = &result
	return nil
}

func (c *ZkClient) GetBalance(ctx context.Context, instrument string) (*big.Int, error) {
	err := c.fillAccountInfo(ctx)
	if err != nil {
		return nil, err
	}
	balanceString := c.accountInfo.Committed.Balances[instrument]
	balance, ok := new(big.Int).SetString(balanceString, 10)
	if !ok {
		return big.NewInt(0), errs.New("Invalid balance string")
	}
	return balance, nil
}

func (c *ZkClient) GetFee(ctx context.Context, txType string, address string, instrument string) (*big.Int, error) {
	client, err := rpc.DialContext(ctx, c.rpcURL)
	if err != nil {
		return big.NewInt(0), err
	}
	defer client.Close()

	result := map[string]interface{}{}
	err = client.CallContext(ctx, &result, "get_tx_fee", txType, address, instrument)
	if err != nil {
		return big.NewInt(0), err
	}
	fee, _ := new(big.Int).SetString(result["totalFee"].(string), 10)
	return fee, nil
}

func (c *ZkClient) GetNonce(ctx context.Context) (uint64, error) {
	err := c.refreshAccountInfo(ctx)
	if err != nil {
		return 0, err
	}
	nonce := c.accountInfo.Committed.Nonce
	return nonce, nil
}

func (c *ZkClient) Address() (common.Address, error) {
	pk, err := c.zkPrivateKey()
	if err != nil {
		return common.Address{}, err
	}
	publicKey, err := pk.PublicKey()
	if err != nil {
		return common.Address{}, err
	}
	from := common.HexToAddress(publicKey.HexString())
	return from, nil
}

type TxStatus struct {
	Executed bool
	Success  bool
	Block    BlockStatus
}

type BlockStatus struct {
	Committed   bool
	Verified    bool
	BlockNumber int
}

func (c *ZkClient) TxStatus(ctx context.Context, txHash string) (TxStatus, error) {
	client, err := rpc.DialContext(ctx, c.rpcURL)
	if err != nil {
		return TxStatus{}, err
	}
	defer client.Close()
	result := TxStatus{}
	err = client.CallContext(ctx, &result, "tx_info", txHash)
	return result, err
}

func (c *ZkClient) GetToken(ctx context.Context, symbol string) (Token, error) {
	client, err := rpc.DialContext(ctx, c.rpcURL)
	if err != nil {
		return Token{}, err
	}
	defer client.Close()
	tokens := make(map[string]Token)
	err = client.CallContext(ctx, &tokens, "tokens")
	if err != nil {
		return Token{}, err
	}
	if token, found := tokens[symbol]; found {
		return token, nil
	} else {
		return Token{}, errs.New("Couldn't find token %symbol", symbol)
	}

}

func reverseBytes(bigInt []byte) []byte {
	reversed := make([]byte, len(bigInt))
	for i := 0; i < len(bigInt); i++ {
		reversed[i] = bigInt[len(bigInt)-1-i]
	}
	return reversed
}
