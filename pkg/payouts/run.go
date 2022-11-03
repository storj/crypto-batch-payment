package payouts

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"math/big"
	"os"
	"path/filepath"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/zeebo/errs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"storj.io/crypto-batch-payment/pkg/payer"
	"storj.io/crypto-batch-payment/pkg/pipelinedb"

	"storj.io/crypto-batch-payment/pkg/coinmarketcap"
	"storj.io/crypto-batch-payment/pkg/eth"
	"storj.io/crypto-batch-payment/pkg/pipeline"
	"storj.io/crypto-batch-payment/pkg/storjtoken"
	"storj.io/crypto-batch-payment/pkg/zksync"
)

type Config struct {
	DataDir string

	Name string

	Spender *ecdsa.PrivateKey

	ChainID big.Int

	Owner common.Address

	Quoter coinmarketcap.Quoter

	EstimateCacheExpiry time.Duration

	MaxGas big.Int
	MaxFee *big.Int

	GasTipCap *big.Int

	ContractAddress common.Address

	PipelineLimit int

	TxDelay time.Duration

	Drain bool

	NodeType pipeline.NodeType

	PromptConfirm func(label string) error

	NodeAddress string

	PayerType PayerType
}

func Run(ctx context.Context, config Config) error {
	runDir := filepath.Join(config.DataDir, config.Name)
	dbPath := dbPathFromDir(runDir)

	db, err := pipelinedb.OpenDB(ctx, dbPath, false)
	if err != nil {
		return err
	}
	defer db.Close()

	stats, err := db.Stats(ctx)
	if err != nil {
		return err
	}

	storjQuote, err := config.Quoter.GetQuote(ctx, coinmarketcap.STORJ)
	if err != nil {
		return err
	}

	log, err := openLog(runDir)
	if err != nil {
		return err
	}

	var paymentPayer payer.Payer
	switch config.PayerType {
	case Eth, Polygon:
		client, err := ethclient.Dial(config.NodeAddress)
		if err != nil {
			return errs.New("Failed to dial node %q: %v\n", config.NodeAddress, err)
		}
		defer client.Close()

		paymentPayer, err = eth.NewEthPayer(ctx,
			log,
			client,
			config.ContractAddress,
			config.Owner,
			config.Spender,
			&config.ChainID,
			config.GasTipCap,
			&config.MaxGas)
		if err != nil {
			return err
		}
	case ZkSync:
		paymentPayer, err = zksync.NewPayer(
			ctx,
			log,
			config.NodeAddress,
			config.Spender,
			int(config.ChainID.Int64()),
			false,
			config.MaxFee)
		if err != nil {
			return err
		}
	case ZkWithdraw:
		paymentPayer, err = zksync.NewPayer(
			ctx,
			log,
			config.NodeAddress,
			config.Spender,
			int(config.ChainID.Int64()),
			true,
			config.MaxFee)
		if err != nil {
			return err
		}
	case Sim:
		paymentPayer, err = payer.NewSimPayer(log)
		if err != nil {
			return err
		}
	default:
		return errs.New("unsupported payer type: %v", config.PayerType)
	}

	decimals, err := paymentPayer.GetTokenDecimals(ctx)
	if err != nil {
		return err
	}

	balance, err := paymentPayer.GetTokenBalance(ctx)
	if err != nil {
		return err
	}

	estimatedSTORJ := storjtoken.FromUSD(stats.PendingUSD, storjQuote.Price, decimals)

	fmt.Printf("**PAYMENT TYPE**............: %s\n", config.PayerType)
	fmt.Printf("Current STORJ Price.........: $%s\n", storjQuote.Price.String())
	fmt.Println()
	fmt.Printf("Total Payees................: %d\n", stats.Payees)
	fmt.Printf("Total Payouts...............: %d\n", stats.TotalPayouts)
	fmt.Printf("Total Payout Groups.........: %d\n", stats.TotalPayoutGroups)
	fmt.Printf("Total USD...................: $%s\n", stats.TotalUSD.String())
	fmt.Println()
	fmt.Printf("Pending Payouts.............: %d\n", stats.PendingPayouts)
	fmt.Printf("Pending Payout Groups.......: %d\n", stats.PendingPayoutGroups)
	fmt.Printf("Pending USD.................: $%s\n", stats.PendingUSD.String())
	fmt.Printf("Pending in STORJ ~ .........: %s\n", storjtoken.Pretty(estimatedSTORJ, decimals))
	fmt.Printf("Current STORJ balance ......: %s\n", storjtoken.Pretty(balance, decimals))
	fmt.Println()
	fmt.Printf("Total Transactions..........: %d\n", stats.TotalTransactions)
	fmt.Printf("Pending Transactions........: %d\n", stats.PendingTransactions)
	fmt.Printf("Failed Transactions.........: %d\n", stats.FailedTransactions)
	fmt.Printf("Confirmed Transactions......: %d\n", stats.ConfirmedTransactions)
	fmt.Printf("Dropped Transactions........: %d\n", stats.DroppedTransactions)

	err = paymentPayer.PrintEstimate(ctx, stats.PendingPayoutGroups)
	if err != nil {
		return err
	}
	fmt.Println()
	fmt.Println("Node Type:", config.NodeType)

	if config.Drain {
		if err := config.PromptConfirm("Drain"); err != nil {
			return err
		}
	} else {
		if err := config.PromptConfirm("Proceed"); err != nil {
			return err
		}
	}

	p, err := pipeline.NewPipeline(paymentPayer, pipeline.PipelineConfig{
		Log:      log,
		Spender:  config.Spender,
		Owner:    config.Owner,
		Quoter:   config.Quoter,
		DB:       db,
		Limit:    config.PipelineLimit,
		Drain:    config.Drain,
		NodeType: config.NodeType,
		TxDelay:  config.TxDelay,
	})
	if err != nil {
		return err
	}

	if err := p.ProcessPayouts(ctx); err != nil {
		return err
	}

	return nil
}

func openLog(dataDir string) (*zap.Logger, error) {
	// Ensure a logs directory exists
	logsDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, errs.Wrap(err)
	}

	// Name the log based on the current timestamp to millisecond precision
	logName := time.Now().UTC().Format("2006.01.02.15.04.05.000Z") + ".json"

	// Convert to an absolute path for the file URI passed to zap
	logsPath, err := filepath.Abs(filepath.Join(logsDir, logName))
	if err != nil {
		return nil, errs.Wrap(err)
	}

	// Send info to stderr
	stderrEncoder := zap.NewDevelopmentEncoderConfig()
	stderrEncoder.EncodeLevel = zapcore.CapitalColorLevelEncoder
	stderrLog, err := (zap.Config{
		Level:         zap.NewAtomicLevelAt(zap.InfoLevel),
		Encoding:      "console",
		EncoderConfig: stderrEncoder,
		OutputPaths:   []string{"stderr"},
	}).Build()
	if err != nil {
		return nil, errs.Wrap(err)
	}

	// Send debug to file as JSON
	fileEncoder := zap.NewProductionEncoderConfig()
	fileEncoder.EncodeTime = zapcore.ISO8601TimeEncoder
	fileLog, err := (zap.Config{
		Level:         zap.NewAtomicLevelAt(zap.DebugLevel),
		Encoding:      "json",
		EncoderConfig: fileEncoder,
		OutputPaths:   []string{"file://" + logsPath},
	}).Build()
	if err != nil {
		return nil, errs.Wrap(err)
	}

	log := zap.New(zapcore.NewTee(stderrLog.Core(), fileLog.Core()))

	// Overwrite the latest symlink
	if err := os.Symlink(logName, filepath.Join(logsDir, ".latest")); err != nil {
		return nil, errs.Wrap(err)
	}
	if err := os.Rename(filepath.Join(logsDir, ".latest"), filepath.Join(logsDir, "latest")); err != nil {
		return nil, errs.Wrap(err)
	}

	return log, nil
}
