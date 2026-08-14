package eth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// maxBodyLog caps how many bytes of a request or response body get logged.
// Full JSON-RPC payloads for our use are well under this; the cap is there so
// a huge response (eth_getLogs, debug traces) can't blow up the log.
const maxBodyLog = 16 * 1024

// loggingTransport is an http.RoundTripper that logs the JSON-RPC request and
// response when the exchange is interesting: any eth_sendRawTransaction call,
// any response that carries a JSON-RPC error object, or any non-2xx HTTP
// status. Routine polling (eth_getTransactionCount, eth_call, etc.) is not
// logged.
type loggingTransport struct {
	log  *zap.Logger
	base http.RoundTripper
}

func newLoggingTransport(log *zap.Logger, base http.RoundTripper) *loggingTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &loggingTransport{log: log, base: base}
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var reqBody []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		reqBody = b
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(reqBody)), nil
		}
	}

	method := peekJSONRPCMethod(reqBody)

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		t.log.Warn("eth rpc transport error",
			zap.String("method", method),
			zap.ByteString("request", truncate(reqBody, maxBodyLog)),
			zap.Error(err),
		)
		return nil, err
	}

	respBody, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	if shouldLog(method, resp.StatusCode, respBody) {
		t.log.Info("eth rpc call",
			zap.String("method", method),
			zap.Int("status", resp.StatusCode),
			zap.ByteString("request", truncate(reqBody, maxBodyLog)),
			zap.ByteString("response", truncate(respBody, maxBodyLog)),
		)
	}

	if readErr != nil {
		return resp, readErr
	}
	return resp, nil
}

func shouldLog(method string, status int, respBody []byte) bool {
	if status != 0 && (status < 200 || status >= 300) {
		return true
	}
	if method == "eth_sendRawTransaction" {
		return true
	}
	if bytes.Contains(respBody, []byte(`"error":`)) {
		return true
	}
	return false
}

// peekJSONRPCMethod extracts the JSON-RPC "method" field from a single-request
// body. Batch requests and unparseable bodies return an empty string.
func peekJSONRPCMethod(body []byte) string {
	trimmed := bytes.TrimLeft(body, " \t\r\n")
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return ""
	}
	var head struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(body, &head); err != nil {
		return ""
	}
	return head.Method
}

func truncate(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[:n]
}
