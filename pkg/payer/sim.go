package payer

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/zeebo/errs"
	"go.uber.org/zap"

	"storj.io/crypto-batch-payment/pkg/contract"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
)

var (
	_ Payer   = &SimPayer{}
	_ Auditor = &SimAuditor{}
)

type SimPayer struct {
	from common.Address
}

func (s *SimPayer) PrintEstimate(ctx context.Context, remaining int64) error {
	fmt.Println("SIMULATING MODE, NO REAL PAYMENT WILL HAPPEN")
	return nil
}

func NewSimPayer() (*SimPayer, error) {
	hash := make([]byte, 20)
	_, err := rand.Read(hash)
	if err != nil {
		return nil, err
	}
	return &SimPayer{
		from: common.BytesToAddress(hash),
	}, nil
}

func (s *SimPayer) String() string {
	return Sim.String()
}

func (s *SimPayer) ChainID() int {
	return 1337
}

func (s *SimPayer) Decimals() int32 {
	return 8
}

func (s *SimPayer) NextNonce(ctx context.Context) (uint64, error) {
	return uint64(0), nil
}

func (s *SimPayer) GetGasInfo(context.Context) (GasInfo, error) {
	return GasInfo{
		GasLimit:  contract.TokenTransferGasLimit,
		GasFeeCap: big.NewInt(1_000_000_000), // 1 gwei
		GasTipCap: big.NewInt(1_000_000),     // .001 gwei
	}, nil
}

func (s *SimPayer) GetETHBalance(ctx context.Context) (*big.Int, error) {
	balance, _ := new(big.Int).SetString("1000000000000000000", 0)
	return balance, nil
}

func (s *SimPayer) GetTokenBalance(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1000000000000), nil
}

func (s *SimPayer) CreateRawTransaction(ctx context.Context, log *zap.Logger, params TransactionParams) (tx Transaction, from common.Address, err error) {
	gasInfo, err := s.GetGasInfo(ctx)
	if err != nil {
		return Transaction{}, common.Address{}, err
	}

	hash := make([]byte, 32)
	_, err = rand.Read(hash)
	if err != nil {
		return Transaction{}, common.Address{}, err
	}
	txHash := common.BytesToHash(hash).String()
	return Transaction{
		Hash:               txHash,
		Nonce:              params.Nonce,
		EstimatedGasLimit:  gasInfo.GasLimit,
		EstimatedGasFeeCap: gasInfo.GasFeeCap,
		Raw: map[string]interface{}{
			"hash": txHash,
		},
	}, s.from, nil
}

func (s *SimPayer) SendTransaction(ctx context.Context, log *zap.Logger, tx Transaction) error {
	log.Info("Sending transaction")
	return nil
}

func (s *SimPayer) CheckNonceGroup(ctx context.Context, log *zap.Logger, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	if len(nonceGroup.Txs) != 1 {
		return pipelinedb.TxFailed, nil, errs.New("noncegroup should have only 1 transaction, not %d", len(nonceGroup.Txs))
	}
	tx := nonceGroup.Txs[0]
	return pipelinedb.TxConfirmed, []*pipelinedb.TxStatus{
		{
			Hash:    tx.Hash,
			State:   pipelinedb.TxConfirmed,
			Receipt: nil,
		},
	}, nil
}

func (s *SimPayer) CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	return pipelinedb.TxConfirmed, nil
}

func (s *SimPayer) CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	return pipelinedb.TxConfirmed, nil
}

type SimAuditor struct{}

// NewSimAuditor creates a simulated auditor.
func NewSimAuditor() SimAuditor {
	return SimAuditor{}
}

func (s SimAuditor) CheckTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	return pipelinedb.TxConfirmed, nil
}

func (s SimAuditor) CheckConfirmedTransactionState(ctx context.Context, hash string) (pipelinedb.TxState, error) {
	return pipelinedb.TxConfirmed, nil
}
