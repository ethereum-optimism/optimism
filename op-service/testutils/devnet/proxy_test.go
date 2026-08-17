package devnet

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

// TestRetryProxy_SetsContentType checks that proxied requests carry the JSON-RPC content type.
// Some clients (Nethermind, for one) reject requests without it.
func TestRetryProxy_SetsContentType(t *testing.T) {
	var gotContentType string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		if gotContentType != "application/json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer upstream.Close()

	prox := NewRetryProxy(testlog.Logger(t, log.LevelDebug), upstream.URL)
	prox.maxRetries = 1 // fail fast instead of backing off through every retry
	require.NoError(t, prox.Start())
	defer func() {
		require.NoError(t, prox.Stop())
	}()

	res, err := http.Post(prox.Endpoint(), "application/json", bytes.NewReader(
		[]byte(`{"jsonrpc":"2.0","id":1,"method":"eth_blockNumber"}`),
	))
	require.NoError(t, err)
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode, "proxy returned: %s", body)
	require.Equal(t, "application/json", gotContentType)
	require.JSONEq(t, `{"jsonrpc":"2.0","id":1,"result":"0x1"}`, string(body))
}
