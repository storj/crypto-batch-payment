package zksyncera

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
	"github.com/zksync-sdk/zksync2-go/accounts"
	"github.com/zksync-sdk/zksync2-go/clients"
	"github.com/zksync-sdk/zksync2-go/contracts/erc20"
	zktypes "github.com/zksync-sdk/zksync2-go/types"
	"github.com/zksync-sdk/zksync2-go/utils"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/contract"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
)

type Payer struct {
	wallet           *accounts.Wallet
	zk               clients.Client
	signer           *accounts.BaseSigner
	contractAddress  common.Address
	erc20abi         abi.ABI
	decimals         int32
	paymasterAddress *common.Address
	paymasterPayload []byte
}

func NewPayer(
	contractAddress common.Address,
	url string,
	key *ecdsa.PrivateKey,
	chainID int,
	paymasterAddress *common.Address,
	paymasterPayload []byte,
	maxFee *big.Int) (*Payer, error) {

	ethSigner, err := accounts.NewBaseSignerFromRawPrivateKey(key.D.Bytes(), int64(chainID))
	if err != nil {
		return nil, errs.Wrap(err)
	}

	zkClients, err := clients.Dial(url)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	signer := accounts.Signer(ethSigner)
	wallet, err := accounts.NewWalletFromSigner(&signer, &zkClients, nil)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	erc20abi, err := abi.JSON(strings.NewReader(erc20.IERC20MetaData.ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to load erc20abi: %w", err)
	}

	p := &Payer{
		wallet:           wallet,
		zk:               zkClients,
		signer:           ethSigner,
		contractAddress:  contractAddress,
		erc20abi:         erc20abi,
		paymasterAddress: paymasterAddress,
		paymasterPayload: paymasterPayload,
	}
	p.decimals, err = p.GetTokenDecimals(context.Background())
	return p, errs.Wrap(err)

}

func (p *Payer) String() string {
	return payer.ZkSyncEra.String()
}

func (p *Payer) NextNonce(ctx context.Context) (uint64, error) {
	nonce, err := p.wallet.Nonce(ctx, nil)
	if err != nil {
		return 0, errs.Wrap(err)
	}
	return nonce, nil
}

func (p *Payer) CheckPreconditions(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (p *Payer) GetTokenBalance(ctx context.Context) (*big.Int, error) {
	return p.wallet.Balance(ctx, p.contractAddress, nil)
}

func (p *Payer) GetTokenDecimals(ctx context.Context) (int32, error) {
	tokenContract, err := contract.NewToken(p.contractAddress, p.zk)
	if err != nil {
		return 0, fmt.Errorf("failed to load ERC20: %w", err)
	}
	decimals, err := tokenContract.Decimals(&bind.CallOpts{})
	if err != nil {
		return 0, fmt.Errorf("failed to load ERC20: %w", err)
	}
	return int32(decimals.Int64()), nil

}

func (p *Payer) CreateRawTransaction(ctx context.Context, log *zap.Logger, payouts []*pipelinedb.Payout, nonce uint64, storjPrice decimal.Decimal) (tx payer.Transaction, from common.Address, err error) {
	from = p.signer.Address()

	if len(payouts) > 1 {
		return payer.Transaction{}, common.Address{}, errs.New("multitransfer is not supported yet")
	}
	payout := payouts[0]

	tokenAmount := storjtoken.FromUSD(payout.USD, storjPrice, p.decimals)

	packedData, err := p.erc20abi.Pack("transfer", payout.Payee, tokenAmount)
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	gasPrice, err := p.zk.SuggestGasPrice(ctx)
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	chainID, err := p.zk.ChainID(ctx)
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	callMsg := zktypes.CallMsg{
		CallMsg: ethereum.CallMsg{
			From:      from,
			To:        &p.contractAddress,
			Gas:       0,             // estimated below
			GasTipCap: big.NewInt(0), // TODO: Estimate correct one
			GasFeeCap: gasPrice,
			Value:     nil,
			Data:      packedData,
		},
		Meta: &zktypes.Eip712Meta{
			GasPerPubdata: utils.NewBig(utils.DefaultGasPerPubdataLimit.Int64()),
		},
	}

	if p.paymasterAddress != nil {
		callMsg.Meta.PaymasterParams = &zktypes.PaymasterParams{
			Paymaster:      *p.paymasterAddress,
			PaymasterInput: p.paymasterPayload,
		}
	}

	gas, err := p.zk.EstimateGasL2(ctx, callMsg)
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	data := &zktypes.Transaction712{
		Nonce:     big.NewInt(int64(nonce)),
		GasTipCap: callMsg.GasTipCap,
		GasFeeCap: callMsg.GasFeeCap,
		Gas:       new(big.Int).SetUint64(gas),
		To:        callMsg.To,
		Value:     callMsg.Value,
		Data:      callMsg.Data,
		ChainID:   chainID,
		From:      &callMsg.From,
		Meta:      callMsg.Meta,
	}

	domain := p.signer.Domain()

	message, err := data.EIP712Message()
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			data.EIP712Type():   data.EIP712Types(),
			domain.EIP712Type(): domain.EIP712Types(),
		},
		PrimaryType: data.EIP712Type(),
		Domain:      domain.EIP712Domain(),
		Message:     message,
	}

	hashTypedData, err := p.signer.HashTypedData(typedData)
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	signature, err := p.signer.SignTypedData(domain, data)
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	rawTx, err := data.RLPValues(signature)
	if err != nil {
		return payer.Transaction{}, common.Address{}, errs.Wrap(err)
	}

	hash := common.Hash{}
	copy(hash[:], crypto.Keccak256(
		append(hashTypedData, crypto.Keccak256(signature)...),
	))

	return payer.Transaction{
		Hash:  hash.String(),
		Nonce: nonce,
		Raw:   rawTx,
	}, from, nil
}

func (p *Payer) SendTransaction(ctx context.Context, log *zap.Logger, tx payer.Transaction) error {
	hash, err := p.zk.SendRawTransaction(ctx, tx.Raw.([]byte))
	if err != nil {
		return err
	}
	if hash.String() != tx.Hash {
		return errs.New("Transaction hash mismatch (pre-calculated, saved in db=%s, returned from the chain=%s)", tx.Hash, hash)
	}
	return nil
}

func (p *Payer) CheckNonceGroup(ctx context.Context, log *zap.Logger, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	if len(nonceGroup.Txs) != 1 {
		return pipelinedb.TxFailed, nil, errs.New("ZkSyncEra payer supports only one transaction per nonce group")
	}

	txHash := common.HexToHash(nonceGroup.Txs[0].Hash)
	zkReceipt, err := p.zk.TransactionReceipt(ctx, txHash)
	if err != nil {
		return pipelinedb.TxDropped, []*pipelinedb.TxStatus{}, errs.Wrap(err)
	}

	status := pipelinedb.TxConfirmed
	switch {
	case zkReceipt == nil:
		status = pipelinedb.TxPending
	case zkReceipt.Status != types.ReceiptStatusSuccessful:
		status = pipelinedb.TxFailed
	}

	var receipt *types.Receipt
	if zkReceipt != nil {
		receipt = &zkReceipt.Receipt
	}

	return status, []*pipelinedb.TxStatus{
		{
			Hash:    nonceGroup.Txs[0].Hash,
			State:   status,
			Receipt: receipt,
		},
	}, err

}

func (p *Payer) PrintEstimate(ctx context.Context, remaining int64) error {
	if p.paymasterAddress != nil {
		fmt.Printf("Paymaster address...........: %s\n", p.paymasterAddress)
		fmt.Printf("Paymaster payload...........: %s\n", common.Bytes2Hex(p.paymasterPayload))
	}
	return nil
}

var (
	_ payer.Payer = &Payer{}
)
