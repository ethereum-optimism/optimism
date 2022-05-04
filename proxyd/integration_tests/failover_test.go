package integration_tests

import (
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

const (
	goodResponse       = `{"jsonrpc": "2.0", "result": "hello", "id": 999}`
	noBackendsResponse = `{"error":{"code":-32011,"message":"no backends available for method"},"id":999,"jsonrpc":"2.0"}`
)

func TestFailover(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()
	badBackend := NewMockBackend(nil)
	defer badBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))
	require.NoError(t, os.Setenv("BAD_BACKEND_RPC_URL", badBackend.URL()))

	config := ReadConfig("failover")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	tests := []struct {
		name    string
		handler http.Handler
	}{
		{
			"backend responds 200 with non-JSON response",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
				_, _ = w.Write([]byte("this data is not JSON!"))
			}),
		},
		{
			"backend responds with no body",
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(200)
			}),
		},
	}
	codes := []int{
		300,
		301,
		302,
		401,
		403,
		429,
		500,
		503,
	}
	for _, code := range codes {
		tests = append(tests, struct {
			name    string
			handler http.Handler
		}{
			fmt.Sprintf("backend %d", code),
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}),
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			badBackend.SetHandler(tt.handler)
			res, statusCode, err := client.SendRPC("eth_chainId", nil)
			require.NoError(t, err)
			require.Equal(t, 200, statusCode)
			RequireEqualJSON(t, []byte(goodResponse), res)
			require.Equal(t, 1, len(badBackend.Requests()))
			require.Equal(t, 1, len(goodBackend.Requests()))
			badBackend.Reset()
			goodBackend.Reset()
		})
	}

	t.Run("backend times out and falls back to another", func(t *testing.T) {
		badBackend.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			_, _ = w.Write([]byte("[{}]"))
		}))
		res, statusCode, err := client.SendRPC("eth_chainId", nil)
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)
		RequireEqualJSON(t, []byte(goodResponse), res)
		require.Equal(t, 1, len(badBackend.Requests()))
		require.Equal(t, 1, len(goodBackend.Requests()))
		goodBackend.Reset()
		badBackend.Reset()
	})

	t.Run("works with a batch request", func(t *testing.T) {
		goodBackend.SetHandler(BatchedResponseHandler(200, goodResponse, goodResponse))
		badBackend.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
		}))
		res, statusCode, err := client.SendBatchRPC(
			NewRPCReq("1", "eth_chainId", nil),
			NewRPCReq("2", "eth_chainId", nil),
		)
		require.NoError(t, err)
		require.Equal(t, 200, statusCode)
		RequireEqualJSON(t, []byte(asArray(goodResponse, goodResponse)), res)
		require.Equal(t, 1, len(badBackend.Requests()))
		require.Equal(t, 1, len(goodBackend.Requests()))
		goodBackend.Reset()
		badBackend.Reset()
	})
}

func TestRetries(t *testing.T) {
	backend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))
	config := ReadConfig("retries")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	attempts := int32(0)
	backend.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		incremented := atomic.AddInt32(&attempts, 1)
		if incremented != 2 {
			w.WriteHeader(500)
			return
		}
		BatchedResponseHandler(200, goodResponse)(w, r)
	}))

	// test case where request eventually succeeds
	res, statusCode, err := client.SendRPC("eth_chainId", nil)
	require.NoError(t, err)
	require.Equal(t, 200, statusCode)
	RequireEqualJSON(t, []byte(goodResponse), res)
	require.Equal(t, 2, len(backend.Requests()))

	// test case where it does not
	backend.Reset()
	attempts = -10
	res, statusCode, err = client.SendRPC("eth_chainId", nil)
	require.NoError(t, err)
	require.Equal(t, 503, statusCode)
	RequireEqualJSON(t, []byte(noBackendsResponse), res)
	require.Equal(t, 4, len(backend.Requests()))
}

func TestOutOfServiceInterval(t *testing.T) {
	okHandler := BatchedResponseHandler(200, goodResponse)
	goodBackend := NewMockBackend(okHandler)
	defer goodBackend.Close()
	badBackend := NewMockBackend(nil)
	defer badBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))
	require.NoError(t, os.Setenv("BAD_BACKEND_RPC_URL", badBackend.URL()))

	config := ReadConfig("out_of_service_interval")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	badBackend.SetHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))

	res, statusCode, err := client.SendRPC("eth_chainId", nil)
	require.NoError(t, err)
	require.Equal(t, 200, statusCode)
	RequireEqualJSON(t, []byte(goodResponse), res)
	require.Equal(t, 2, len(badBackend.Requests()))
	require.Equal(t, 1, len(goodBackend.Requests()))

	res, statusCode, err = client.SendRPC("eth_chainId", nil)
	require.NoError(t, err)
	require.Equal(t, 200, statusCode)
	RequireEqualJSON(t, []byte(goodResponse), res)
	require.Equal(t, 2, len(badBackend.Requests()))
	require.Equal(t, 2, len(goodBackend.Requests()))

	_, statusCode, err = client.SendBatchRPC(
		NewRPCReq("1", "eth_chainId", nil),
		NewRPCReq("1", "eth_chainId", nil),
	)
	require.NoError(t, err)
	require.Equal(t, 200, statusCode)
	require.Equal(t, 2, len(badBackend.Requests()))
	require.Equal(t, 4, len(goodBackend.Requests()))

	time.Sleep(time.Second)
	badBackend.SetHandler(okHandler)

	res, statusCode, err = client.SendRPC("eth_chainId", nil)
	require.NoError(t, err)
	require.Equal(t, 200, statusCode)
	RequireEqualJSON(t, []byte(goodResponse), res)
	require.Equal(t, 3, len(badBackend.Requests()))
	require.Equal(t, 4, len(goodBackend.Requests()))
}

func TestBatchWithPartialFailover(t *testing.T) {
	config := ReadConfig("failover")
	config.Server.MaxUpstreamBatchSize = 2

	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse, goodResponse))
	defer goodBackend.Close()
	badBackend := NewMockBackend(SingleResponseHandler(200, "this data is not JSON!"))
	defer badBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))
	require.NoError(t, os.Setenv("BAD_BACKEND_RPC_URL", badBackend.URL()))

	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	res, statusCode, err := client.SendBatchRPC(
		NewRPCReq("1", "eth_chainId", nil),
		NewRPCReq("2", "eth_chainId", nil),
		NewRPCReq("3", "eth_chainId", nil),
		NewRPCReq("4", "eth_chainId", nil),
	)
	require.NoError(t, err)
	require.Equal(t, 200, statusCode)
	RequireEqualJSON(t, []byte(asArray(goodResponse, goodResponse, goodResponse, goodResponse)), res)
	require.Equal(t, 2, len(badBackend.Requests()))
	require.Equal(t, 2, len(goodBackend.Requests()))
}
