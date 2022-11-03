package eth

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"storj.io/crypto-batch-payment/pkg"

	"github.com/zeebo/errs/v2"
	"storj.io/crypto-batch-payment/pkg/contract"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
)

type EthPayer struct {
	log *zap.Logger

	client        *ethclient.Client
	contract      *contract.Token
	owner         common.Address
	gasTipCap     *big.Int
	maxGas        *big.Int
	signer        bind.SignerFn
	from          common.Address
	tokenDecimals int32
}

var (
	_ payer.Payer = &EthPayer{}

	// zero is a big int set to 0 for convenience.
	zero = big.NewInt(0)
)

func NewEthPayer(ctx context.Context,
	logger *zap.Logger,
	client *ethclient.Client,
	contractAddress common.Address,
	owner common.Address,
	key *ecdsa.PrivateKey,
	chainID *big.Int,
	gasTipCap *big.Int,
	maxGas *big.Int) (*EthPayer, error) {

	contract, err := contract.NewToken(contractAddress, &ignoreSend{
		ContractBackend: client,
	})
	if err != nil {
		return nil, errs.Wrap(err)
	}

	if gasTipCap == nil {
		suggestedGasTip, err := client.SuggestGasTipCap(ctx)
		if err != nil {
			return nil, errs.Wrap(err)
		}
		gasTipCap = suggestedGasTip
	}

	opts, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	decimals, err := contract.Decimals(nil)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	return &EthPayer{
		log:           logger,
		owner:         owner,
		gasTipCap:     gasTipCap,
		maxGas:        maxGas,
		client:        client,
		contract:      contract,
		signer:        opts.Signer,
		from:          opts.From,
		tokenDecimals: int32(decimals.Int64()),
	}, nil
}

func (e *EthPayer) NextNonce(ctx context.Context) (uint64, error) {
	return e.client.NonceAt(ctx, e.from, nil)
}

func (e *EthPayer) IsPreconditionMet(ctx context.Context) (bool, error) {
	lastBlock, err := e.client.BlockByNumber(ctx, nil)
	if err != nil {
		return false, errs.Wrap(err)
	}

	// max gas should be higher than the base fee + tip
	if e.maxGas.Cmp(new(big.Int).Add(lastBlock.BaseFee(), e.gasTipCap)) < 0 {
		e.log.Sugar().Warnf("base fee in last block is %s, together with the tip %s it's larger "+
			"than the max allowed gas price %s. "+
			"sleeping for 5 seconds", lastBlock.BaseFee(), e.gasTipCap, e.maxGas)
		return false, nil
	}
	return true, nil

}

func (e *EthPayer) GetTokenBalance(ctx context.Context) (*big.Int, error) {
	storjBalance, err := e.contract.BalanceOf(&bind.CallOpts{
		Pending: false,
		Context: ctx,
	}, e.owner)
	return storjBalance, errs.Wrap(err)
}

func (e *EthPayer) CreateRawTransaction(ctx context.Context, log *zap.Logger, payouts []*pipelinedb.Payout,
	nonce uint64, storjPrice decimal.Decimal) (tx payer.Transaction, from common.Address, err error) {

	var rawTx *types.Transaction
	if len(payouts) > 1 {
		err = errs.Errorf("multitransfer is not supported yet")
		return
	}
	payout := payouts[0]

	opts := &bind.TransactOpts{
		From:      e.from,
		Signer:    e.signer,
		Value:     zero,
		Nonce:     new(big.Int).SetUint64(nonce),
		GasTipCap: e.gasTipCap,
		GasFeeCap: e.maxGas,
		Context:   ctx,
	}

	storjTokens := storjtoken.FromUSD(payout.USD, storjPrice, e.tokenDecimals)
	var storjAllowance *big.Int
	if e.owner == opts.From {
		opts.GasLimit = contract.TokenTransferGasLimit
		rawTx, err = e.contract.Transfer(opts, payout.Payee, storjTokens)
	} else {
		// Check the STORJ allowance to make sure there is enough. Since the
		// contract does not support pending operations, the best we can do
		// is check the live balance.
		storjAllowance, err = e.contract.Allowance(&bind.CallOpts{
			Pending: false,
			Context: ctx,
		}, e.owner, opts.From)
		if err != nil {
			err = errs.Wrap(err)
			return
		}
		if storjAllowance.Cmp(storjTokens) < 0 {
			err = errs.Errorf("not enough STORJ allowance to cover transfer")
			return
		}

		opts.GasLimit = contract.TokenTransferFromGasLimit
		rawTx, err = e.contract.TransferFrom(opts, e.owner, payout.Payee, storjTokens)
	}
	if err != nil {
		err = errs.Wrap(err)
		return
	}

	// Grab the pending ETH balance for logging
	ethBalance, err := e.client.PendingBalanceAt(ctx, opts.From)
	if err != nil {
		err = errs.Wrap(err)
		return
	}

	fields := []zap.Field{
		zap.String("payee", payout.Payee.String()),
		zap.String("usd", payout.USD.String()),
		zap.String("pending-eth-balance", ethBalance.String()),
		zap.String("hash", rawTx.Hash().String()),
	}

	if e.owner != e.from {
		fields = append(fields,
			zap.String("spender", opts.From.String()),
			zap.String("spender-storj-allowance", storjAllowance.String()),
		)
	}
	log.With(fields...).Info("Transaction is created")

	return payer.Transaction{
		Hash:  rawTx.Hash().Hex(),
		Nonce: nonce,
		Raw:   rawTx,
	}, e.from, nil
}

func (e *EthPayer) SendTransaction(ctx context.Context, t payer.Transaction) error {
	switch tx := t.Raw.(type) {
	case *types.Transaction:
		return errs.Wrap(e.client.SendTransaction(ctx, tx))
	default:
		return errs.Errorf("EthPayer doesn't support transaction %v", t.Raw)
	}
}

func (e *EthPayer) EstimatedGasFee(ctx context.Context) (*big.Int, error) {
	lastBlock, err := e.client.BlockByNumber(ctx, nil)
	if err != nil {
		return nil, errs.Wrap(err)
	}

	baseGasFee := lastBlock.BaseFee()
	return baseGasFee, nil
}

func (e *EthPayer) CheckNonceGroup(ctx context.Context, nonceGroup *pipelinedb.NonceGroup, checkOnly bool) (pipelinedb.TxState, []*pipelinedb.TxStatus, error) {
	log := e.log.With(zap.Uint64("nonce", nonceGroup.Nonce), zap.Int64("payout-group-id", nonceGroup.PayoutGroupID))

	log.Debug("Checking nonce group",
		zap.Int("txs", len(nonceGroup.Txs)),
	)

	transactions, err := e.getNonceGroupTxStatus(ctx, nonceGroup)
	if err != nil {
		return "", transactions.all, err
	}

	if len(transactions.other) > 1 {
		// There are two or more transactions for this nonce still being
		// reported by the node. Continuing to process transactions for this
		// group in this state would be dangerous. Do nothing until the node
		// decides which transaction to keep. This is highly unlikely and
		// should be transient, although we've certainly had our surprises
		// around geth and parity behavior in the past.
		log.Warn("Multiple transactions for nonce present", zap.Int("count", len(transactions.other)))
		return pipelinedb.TxPending, transactions.all, nil
	}

	if len(transactions.other) == 0 {
		if checkOnly {
			log.Warn("All transactions dropped but previous nonce groups have failed; aborting",
				zap.Int("dropped", len(transactions.dropped)))
			return "", transactions.all, errs.Errorf("re-send aborted due to previous nonce group failure")
		}
		e.log.Warn("All transactions dropped; sending another", zap.Int("dropped", len(transactions.dropped)))
		return pipelinedb.TxDropped, transactions.all, nil
	}

	status := transactions.other[0]
	switch status.State {
	case pipelinedb.TxPending:
		// The transaction is still pending. Nothing to do but wait.
		return pipelinedb.TxPending, transactions.all, nil
	case pipelinedb.TxFailed:
		// The transaction has failed. This should fail the pipeline since
		// continuing could rack up gas costs trying to push transactions
		// through when there isn't enough STORJ to cover the transfers.
		// The payout group will be picked up again when the pipeline is
		// restarted after the operator has had a chance to rectify the
		// situation.
		log.Warn("Transaction failed", zap.String("hash", status.Hash))
		return pipelinedb.TxFailed, transactions.all, nil
	case pipelinedb.TxConfirmed:
		log.Info("Transaction confirmed", zap.String("hash", status.Hash))
		return pipelinedb.TxConfirmed, transactions.all, nil
	case pipelinedb.TxDropped:
		// Dropped transactions should have already been filtered out already.
		fallthrough
	default:
		return "", transactions.all, errs.Errorf("unexpected state %q for tx %s", status.State, status.Hash)
	}
}

type nonceGroupTransactions struct {
	all     []*pipelinedb.TxStatus
	dropped []*pipelinedb.TxStatus
	other   []*pipelinedb.TxStatus
}

func (e *EthPayer) getNonceGroupTxStatus(ctx context.Context, nonceGroup *pipelinedb.NonceGroup) (txs nonceGroupTransactions, err error) {
	var all, dropped, other []*pipelinedb.TxStatus
	counts := struct {
		pending   int
		dropped   int
		failed    int
		confirmed int
	}{}

	for _, tx := range nonceGroup.Txs {
		status, err := e.getTransactionStatus(ctx, tx.Hash)
		if err != nil {
			e.log.Error("Unable to get transaction status",
				zap.String("hash", tx.Hash),
				zap.Error(err),
			)
			return nonceGroupTransactions{}, err
		}

		e.log.Debug("Transaction status",
			zap.Uint64("nonce", tx.Nonce),
			zap.String("hash", tx.Hash),
			zap.String("state", string(status.State)),
		)

		// gather stats
		switch status.State {
		case pipelinedb.TxPending:
			counts.pending++
		case pipelinedb.TxDropped:
			counts.dropped++
		case pipelinedb.TxFailed:
			counts.failed++
		case pipelinedb.TxConfirmed:
			counts.confirmed++
		default:
			return nonceGroupTransactions{}, errs.Errorf("unexpected state %q for tx %s", status.State, status.Hash)
		}

		all = append(all, status)
		if status.State == pipelinedb.TxDropped {
			dropped = append(dropped, status)
		} else {
			other = append(other, status)
		}
	}

	allGood := true
	fields := []zap.Field{
		zap.Uint64("nonce", nonceGroup.Nonce),
		zap.Int("txs", len(nonceGroup.Txs)),
	}
	if counts.pending > 0 {
		fields = append(fields, zap.Int("pending", counts.pending))
	}
	if counts.dropped > 0 {
		allGood = false
		fields = append(fields, zap.Int("dropped", counts.dropped))
	}
	if counts.failed > 0 {
		allGood = false
		fields = append(fields, zap.Int("failed", counts.failed))
	}
	if counts.confirmed > 0 {
		fields = append(fields, zap.Int("confirmed", counts.confirmed))
	}

	if allGood {
		e.log.Debug("Nonce group status", fields...)
	} else {
		e.log.Warn("Nonce group status", fields...)
	}

	return nonceGroupTransactions{
		all:     all,
		dropped: dropped,
		other:   other,
	}, nil
}

func (e *EthPayer) getTransactionStatus(ctx context.Context, hashString string) (*pipelinedb.TxStatus, error) {
	hash, err := batchpayment.HashFromString(hashString)
	if err != nil {
		return nil, err
	}
	state, _, receipt, err := GetTransactionInfo(ctx, e.client, hash)
	if err != nil {
		return nil, err
	}

	return &pipelinedb.TxStatus{
		Hash:    hash.String(),
		Receipt: receipt,
		State:   state,
	}, nil
}

func (e *EthPayer) PrintEstimate(ctx context.Context, remaining int64) error {

	// TODO: revisit with multitransfer by estimating payout group size, etc.
	var gasPerTx *big.Int
	if e.owner == e.from {
		// transfer rough cost estimate
		gasPerTx = big.NewInt(contract.TokenTransferGasLimit)
	} else {
		// transfer-from rough cost estimate
		gasPerTx = big.NewInt(contract.TokenTransferFromGasLimit)
	}
	gasFee, err := e.EstimatedGasFee(ctx)
	if err != nil {
		return err
	}
	estimatedGasLeft := new(big.Int).Mul(big.NewInt(remaining), gasPerTx)
	estimatedGasCost := new(big.Int).Mul(estimatedGasLeft, gasFee)

	fmt.Printf("Estimated Gas Per Tx........: %s\n", gasPerTx)
	fmt.Printf("Estimated Gas Remaining.....: %s\n", estimatedGasLeft)
	fmt.Printf("Current base fee    ........: %s\n", batchpayment.PrettyETH(gasFee))
	fmt.Printf("Remaining Gas Cost..........: %s\n", batchpayment.PrettyETH(estimatedGasCost))
	return nil
}

func (e *EthPayer) GetTokenDecimals(ctx context.Context) (int32, error) {
	return e.tokenDecimals, nil
}

// ignoreSend wraps a contract backend to not actually send the transaction.
// It is used with the token contract to prepare and sign transactions but not
// actually send them.
type ignoreSend struct {
	bind.ContractBackend
}

func (p *ignoreSend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	return nil
}
