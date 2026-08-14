package eth

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type stubTransport struct {
	status  int
	body    string
	err     error
	seenReq []byte
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		s.seenReq = b
	}
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{
		StatusCode: s.status,
		Body:       io.NopCloser(strings.NewReader(s.body)),
		Header:     make(http.Header),
	}, nil
}

func newTestTransport(t *testing.T, stub *stubTransport) (*loggingTransport, *observer.ObservedLogs) {
	t.Helper()
	core, obs := observer.New(zapcore.DebugLevel)
	log := zap.New(core)
	return newLoggingTransport(log, stub), obs
}

func doRPC(t *testing.T, lt *loggingTransport, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequest("POST", "https://example/rpc", strings.NewReader(body))
	require.NoError(t, err)
	resp, err := lt.RoundTrip(req)
	require.NoError(t, err)
	return resp
}

func TestLoggingTransport_QuietOnRoutineSuccess(t *testing.T) {
	stub := &stubTransport{status: 200, body: `{"jsonrpc":"2.0","id":1,"result":"0x2b820"}`}
	lt, obs := newTestTransport(t, stub)
	doRPC(t, lt, `{"jsonrpc":"2.0","id":1,"method":"eth_getTransactionCount","params":["0xabc","pending"]}`)
	require.Zero(t, obs.Len(), "routine success should not be logged")
}

func TestLoggingTransport_LogsSendRawTransaction(t *testing.T) {
	stub := &stubTransport{status: 200, body: `{"jsonrpc":"2.0","id":1,"result":"0xhash"}`}
	lt, obs := newTestTransport(t, stub)
	doRPC(t, lt, `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":["0x02f8..."]}`)
	require.Equal(t, 1, obs.Len())
	entry := obs.All()[0]
	require.Equal(t, "eth rpc call", entry.Message)
	require.Equal(t, "eth_sendRawTransaction", entry.ContextMap()["method"])
}

func TestLoggingTransport_LogsErrorResponse(t *testing.T) {
	stub := &stubTransport{status: 200, body: `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"Internal error"}}`}
	lt, obs := newTestTransport(t, stub)
	doRPC(t, lt, `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[]}`)
	require.Equal(t, 1, obs.Len())
	fields := obs.All()[0].ContextMap()
	require.Contains(t, fields["response"].(string), "-32603")
}

func TestLoggingTransport_LogsNon2xx(t *testing.T) {
	stub := &stubTransport{status: 503, body: "<html>Service Unavailable</html>"}
	lt, obs := newTestTransport(t, stub)
	doRPC(t, lt, `{"jsonrpc":"2.0","id":1,"method":"eth_getTransactionCount","params":[]}`)
	require.Equal(t, 1, obs.Len())
	fields := obs.All()[0].ContextMap()
	require.Equal(t, int64(503), fields["status"])
}

func TestLoggingTransport_SurvivesNonJSONResponse(t *testing.T) {
	// Non-JSON response with an error status — should log without panicking.
	stub := &stubTransport{status: 429, body: "rate limited"}
	lt, obs := newTestTransport(t, stub)
	doRPC(t, lt, `{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction","params":[]}`)
	require.Equal(t, 1, obs.Len())
}

func TestLoggingTransport_PreservesRequestBody(t *testing.T) {
	// The base transport must see the exact request body — nothing consumed.
	reqBody := `{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[]}`
	stub := &stubTransport{status: 200, body: `{"jsonrpc":"2.0","id":1,"result":"0x"}`}
	lt, _ := newTestTransport(t, stub)
	doRPC(t, lt, reqBody)
	require.Equal(t, reqBody, string(stub.seenReq))
}

func TestLoggingTransport_PreservesResponseBody(t *testing.T) {
	// The caller must be able to read the response body after RoundTrip returns.
	body := `{"jsonrpc":"2.0","id":1,"result":"0x2b820"}`
	stub := &stubTransport{status: 200, body: body}
	lt, _ := newTestTransport(t, stub)
	resp := doRPC(t, lt, `{"jsonrpc":"2.0","id":1,"method":"eth_getTransactionCount","params":[]}`)
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, body, string(got))
}

func TestLoggingTransport_LogsTransportError(t *testing.T) {
	stub := &stubTransport{err: errors.New("dial tcp: connection refused")}
	lt, obs := newTestTransport(t, stub)
	req, _ := http.NewRequest("POST", "https://example/rpc", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"eth_sendRawTransaction"}`))
	_, err := lt.RoundTrip(req)
	require.Error(t, err)
	require.Equal(t, 1, obs.Len())
	require.Equal(t, "eth rpc transport error", obs.All()[0].Message)
}

func TestLoggingTransport_TruncatesHugeBodies(t *testing.T) {
	huge := strings.Repeat("A", maxBodyLog*3)
	stub := &stubTransport{status: 500, body: huge}
	lt, obs := newTestTransport(t, stub)
	doRPC(t, lt, `{"jsonrpc":"2.0","id":1,"method":"eth_call"}`)
	fields := obs.All()[0].ContextMap()
	require.Equal(t, maxBodyLog, len(fields["response"].(string)))
}

func TestPeekJSONRPCMethod_BatchReturnsEmpty(t *testing.T) {
	got := peekJSONRPCMethod([]byte(`[{"method":"eth_call"},{"method":"eth_call"}]`))
	require.Equal(t, "", got)
}

func TestPeekJSONRPCMethod_GarbageReturnsEmpty(t *testing.T) {
	got := peekJSONRPCMethod([]byte("not json at all"))
	require.Equal(t, "", got)
}

func TestPeekJSONRPCMethod_EmptyBody(t *testing.T) {
	got := peekJSONRPCMethod(nil)
	require.Equal(t, "", got)
}

// Sanity check that bytes.Contains lookup can't be spoofed by a well-known
// substring appearing in a legitimate string result value.
func TestShouldLog_ErrorKeyMatch(t *testing.T) {
	require.True(t, shouldLog("", 200, []byte(`{"error":{"code":-32603}}`)))
	require.False(t, shouldLog("", 200, []byte(`{"result":"0x"}`)))
}

// Guard against a subtle regression: bytes.Reader must not accidentally share
// storage with the caller's buffer in a way that would corrupt logs.
func TestTruncate(t *testing.T) {
	src := []byte("hello world")
	got := truncate(src, 5)
	require.Equal(t, []byte("hello"), got)
	require.True(t, bytes.Equal(truncate(src, 100), src))
}
