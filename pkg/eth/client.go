package eth

import (
	"context"
	"net/http"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/zeebo/errs/v2"
	"go.uber.org/zap"
)

// Dial connects to a JSON-RPC endpoint over HTTP(S) and installs a logging
// HTTP transport so that interesting exchanges (eth_sendRawTransaction, error
// responses, non-2xx statuses) are recorded to log for later diagnosis.
//
// If log is nil, no logging is performed.
func Dial(ctx context.Context, nodeAddress string, log *zap.Logger) (*ethclient.Client, error) {
	if log == nil {
		log = zap.NewNop()
	}
	httpClient := &http.Client{
		Transport: newLoggingTransport(log, http.DefaultTransport),
	}
	rpcClient, err := rpc.DialOptions(ctx, nodeAddress, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, errs.Wrap(err)
	}
	return ethclient.NewClient(rpcClient), nil
}
