package integration_tests

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

const (
	batchTimeoutResponse = `{"error":{"code":-32015,"message":"gateway timeout"},"id":null,"jsonrpc":"2.0"}`
)

func TestBatchTimeout(t *testing.T) {
	slowBackend := NewMockBackend(nil)
	defer slowBackend.Close()

	require.NoError(t, os.Setenv("SLOW_BACKEND_RPC_URL", slowBackend.URL()))

	config := ReadConfig("batch_timeout")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	slowBackend.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// check the config. The sleep duration should be at least double the server.timeout_seconds config to prevent flakes
		time.Sleep(time.Second * 2)
		BatchedResponseHandler(200, goodResponse)(w, r)
	}))
	res, statusCode, err := client.SendBatchRPC(
		NewRPCReq("1", "eth_chainId", nil),
		NewRPCReq("1", "eth_chainId", nil),
	)
	require.NoError(t, err)
	require.Equal(t, 504, statusCode)
	RequireEqualJSON(t, []byte(batchTimeoutResponse), res)
	require.Equal(t, 1, len(slowBackend.Requests()))
}
