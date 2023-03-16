package zksync2

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs"
	"github.com/zksync-sdk/zksync2-go"
	"github.com/zksync-sdk/zksync2-go/contracts/ERC20"
	"go.uber.org/zap"
	"math/big"
	"storj.io/crypto-batch-payment/pkg/contract"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
	"strings"
)

type Payer struct {
	log              *zap.Logger
	maxFee           *big.Int
	wallet           *zksync2.Wallet
	ethereumProvider interface{}
	zk               *zksync2.DefaultProvider
	signer           *zksync2.DefaultEthSigner
	contractAddress  common.Address
	erc20abi         abi.ABI
	decimals         int32
}

func NewPayer(
	logger *zap.Logger,
	contractAddress common.Address,
	url string,
	key *ecdsa.PrivateKey,
	chainID int,
	maxFee *big.Int) (*Payer, error) {

	ethereumSigner, err := zksync2.NewEthSignerFromRawPrivateKey(key.D.Bytes(), int64(chainID))
	if err != nil {
		return nil, errs.Wrap(err)
	}

	zkSyncProvider, err := zksync2.NewDefaultProvider("https://zksync2-testnet.zksync.dev")
	if err != nil {
		return nil, errs.Wrap(err)
	}

	wallet, err := zksync2.NewWallet(ethereumSigner, zkSyncProvider)

	erc20abi, err := abi.JSON(strings.NewReader(ERC20.ERC20MetaData.ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to load erc20abi: %w", err)
	}

	p := &Payer{
		wallet:          wallet,
		log:             logger,
		zk:              zkSyncProvider,
		signer:          ethereumSigner,
		contractAddress: contractAddress,
		erc20abi:        erc20abi,
	}
	p.decimals, err = p.GetTokenDecimals(context.Background())
	return p, nil

}

func (p *Payer) NextNonce(ctx context.Context) (uint64, error) {
	nonce, err := p.wallet.GetNonce()
	if err != nil {
		return 0, errs.Wrap(err)
	}
	return nonce.Uint64(), nil
}

func (p *Payer) IsPreconditionMet(ctx context.Context) (bool, error) {
	return true, nil
}

func (p *Payer) GetTokenBalance(ctx context.Context) (*big.Int, error) {
	return p.wallet.GetBalanceOf(p.signer.GetAddress(), &zksync2.Token{
		L2Address: p.contractAddress,
	}, zksync2.BlockNumberCommitted)
}

func (p *Payer) GetTokenDecimals(ctx context.Context) (int32, error) {
	tokenContract, err := contract.NewToken(p.contractAddress, p.zk.GetClient())
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
	from = p.signer.GetAddress()

	if len(payouts) > 1 {
		err = errs.New("multitransfer is not supported yet")
		return
	}
	payout := payouts[0]

	tokenAmount := storjtoken.FromUSD(payout.USD, storjPrice, p.decimals)

	packedData, err := p.erc20abi.Pack("transfer", payout.Payee, tokenAmount)
	if err != nil {
		return tx, from, errs.Wrap(err)
	}

	zkTx := zksync2.CreateFunctionCallTransaction(
		from,
		payouts[0].Payee,
		big.NewInt(0),
		big.NewInt(0),
		big.NewInt(0),
		packedData,
		nil, nil,
	)

	gas, err := p.zk.EstimateGas(zkTx)
	if err != nil {
		return tx, from, errs.Wrap(err)
	}

	gasPrice, err := p.zk.GetGasPrice()
	if err != nil {
		return tx, from, errs.Wrap(err)
	}

	chainId, err := p.zk.ChainID(ctx)
	if err != nil {
		return tx, from, errs.Wrap(err)
	}

	data := zksync2.NewTransaction712(
		chainId,
		big.NewInt(int64(nonce)),
		gas,
		zkTx.To,
		zkTx.Value.ToInt(),
		zkTx.Data,
		big.NewInt(100000000), // TODO: Estimate correct one
		gasPrice,
		zkTx.From,
		zkTx.Eip712Meta,
	)

	domain := p.signer.GetDomain()

	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			data.GetEIP712Type():   data.GetEIP712Types(),
			domain.GetEIP712Type(): domain.GetEIP712Types(),
		},
		PrimaryType: data.GetEIP712Type(),
		Domain:      domain.GetEIP712Domain(),
		Message:     data.GetEIP712Message(),
	}

	hashTypedData, err := p.signer.HashTypedData(typedData)
	if err != nil {
		return
	}

	signature, err := p.signer.SignTypedData(domain, data)
	if err != nil {
		return tx, from, errs.Wrap(err)
	}

	rawTx, err := data.RLPValues(signature)
	if err != nil {
		return tx, from, errs.Wrap(err)
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

func (p *Payer) SendTransaction(ctx context.Context, tx payer.Transaction) error {
	hash, err := p.zk.SendRawTransaction(tx.Raw.([]byte))
	if hash != tx.Hash {
		return errs.New("Transaction hash mismatch (pre-calculated, saved in db=%s, returned from the chain=%s)", hash, tx.Hash)
	}
	return err
}

func (p *Payer) CheckNonceGroup(ctx context.Context, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	//TODO implement me
	panic("implement me")
}

func (p *Payer) PrintEstimate(ctx context.Context, remaining int64) error {
	return nil
}

var (
	_ payer.Payer = &Payer{}
)
