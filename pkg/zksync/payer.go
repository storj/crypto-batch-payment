package zksync

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethereum/go-ethereum/common"
	"github.com/shopspring/decimal"
	"github.com/zeebo/errs/v2"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
)

type Payer struct {
	client   *ZkClient
	token    Token
	withdraw bool // set to true to pay on l1 (from l2!)
	maxFee   *big.Int
}

var (
	_ payer.Payer = &Payer{}
)

func NewPayer(
	ctx context.Context,
	url string,
	key *ecdsa.PrivateKey,
	chainID int,
	withdraw bool,
	maxFee *big.Int) (*Payer, error) {
	client, err := NewZkClient(key, url)
	if err != nil {
		return nil, err
	}
	client.ChainID = chainID

	token, err := client.GetToken(ctx, "STORJ")
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return &Payer{
		client:   &client,
		token:    token,
		withdraw: withdraw,
		maxFee:   maxFee,
	}, nil
}

func (z Payer) NextNonce(ctx context.Context) (uint64, error) {
	return z.client.GetNonce(ctx)
}

func (z Payer) CheckPreconditions(ctx context.Context) ([]string, error) {
	return nil, nil
}

func (z Payer) GetTokenBalance(ctx context.Context) (*big.Int, error) {
	return z.client.GetBalance(ctx, "STORJ")
}

func (z Payer) CreateRawTransaction(ctx context.Context, log *zap.Logger, payouts []*pipelinedb.Payout, nonce uint64, storjPrice decimal.Decimal) (tx payer.Transaction, from common.Address, err error) {
	if len(payouts) > 1 {
		err = errs.Errorf("multitransfer is not supported yet")
		return
	}

	address, err := z.client.Address()
	if err != nil {
		return payer.Transaction{}, address, err
	}

	payout := payouts[0]

	var fee *big.Int
	for {
		fee, err = z.client.GetFee(ctx, z.TxTypeString(), payout.Payee.String(), "STORJ")
		if err != nil {
			return payer.Transaction{}, address, err
		}
		if z.maxFee == nil || z.maxFee.Cmp(fee) >= 0 {
			break
		}
		log.Sugar().Infof("current fee (%s) greater than max fee (%s), sleeping", fee.String(), z.maxFee.String())
		time.Sleep(5 * time.Second)
	}

	storjTokens := storjtoken.FromUSD(payout.USD, storjPrice, z.token.Decimals)
	storjTokens = closestPackableAmount(storjTokens, amountExp, amountMantissa)

	txs, err := z.client.CreateTx(ctx, z.TxTypeString(), payout.Payee, storjTokens, fee, z.token, int(nonce))
	if err != nil {
		return payer.Transaction{}, address, err
	}

	hash, err := txs.Tx.Hash()
	if err != nil {
		return payer.Transaction{}, address, err
	}
	return payer.Transaction{
		Raw:   txs,
		Nonce: uint64(txs.Tx.Nonce),
		Hash:  hash,
	}, address, nil

}

func (z Payer) SendTransaction(ctx context.Context, log *zap.Logger, t payer.Transaction) error {
	switch tx := t.Raw.(type) {
	case TxWithEthSignature:
		txHash, err := z.client.SubmitTransaction(ctx, tx)
		if err != nil {
			return errs.Errorf("Transaction %s is failed: %+v", t.Hash, err)
		}
		if txHash != t.Hash {
			return errs.Errorf("Transaction hash calculated wrongly %s!=%s", txHash, t.Hash)
		}
		return errs.Wrap(err)
	default:
		return errs.Errorf("ZkSyncPayer doesn't support transaction %v", t.Raw)
	}
}

func (z Payer) CheckNonceGroup(ctx context.Context, log *zap.Logger, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	if len(nonceGroup.Txs) != 1 {
		return pipelinedb.TxFailed, nil, errs.Errorf("noncegroup should have only 1 transaction, not %d", len(nonceGroup.Txs))
	}
	tx := nonceGroup.Txs[0]
	hash := tx.Hash

	// zksync api doesn't accept TX in pure hex format
	if !strings.HasPrefix(hash, "0x") && !strings.HasPrefix(hash, syncTxPrefix) {
		hash = "0x" + hash
	}
	status, err := z.client.TxStatus(ctx, hash)
	if err != nil {
		return pipelinedb.TxFailed, nil, err
	}
	txStatus := TxStateFromZkStatus(status)
	return txStatus, []*pipelinedb.TxStatus{
		{
			Hash:    tx.Hash,
			State:   txStatus,
			Receipt: nil,
		},
	}, nil
}

func TxStateFromZkStatus(status TxStatus) pipelinedb.TxState {
	if !status.Executed {
		return pipelinedb.TxPending
	} else if !status.Success {
		return pipelinedb.TxFailed
	} else if status.Executed {
		return pipelinedb.TxConfirmed
	}
	return pipelinedb.TxPending
}

func (z Payer) PrintEstimate(ctx context.Context, remaining int64) error {
	if z.withdraw {
		fmt.Println("Payment type................: L2->L1 (!)")
	} else {
		fmt.Println("Payment type................: L2->L2 (normal zksync)")
	}

	// use generated address to get fee for cold addresses
	rawKey, err := crypto.GenerateKey()
	if err != nil {
		return errs.Wrap(err)
	}
	address := crypto.PubkeyToAddress(rawKey.PublicKey).Hex()

	fee, err := z.client.GetFee(ctx, z.TxTypeString(), address, "STORJ")
	if err != nil {
		return errs.Wrap(err)
	}

	fmt.Printf("Current fee / tx ...........: %s\n", storjtoken.Pretty(fee, z.token.Decimals))
	remainingFee := new(big.Int).Mul(fee, big.NewInt(remaining))
	fmt.Printf("Remaining tx fee ~ .........: %s\n", storjtoken.Pretty(remainingFee, z.token.Decimals))

	return nil
}

func (z Payer) GetTokenDecimals(ctx context.Context) (int32, error) {
	return z.token.Decimals, nil
}

func (z Payer) TxTypeString() string {
	if z.withdraw {
		return "Withdraw"
	}
	return "Transfer"
}
