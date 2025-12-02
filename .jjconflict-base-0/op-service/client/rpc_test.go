package client_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func startTestJSONRPCServerWithDataField() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"error": map[string]interface{}{
				"code":    -38002,
				"message": "Invalid forkchoice state",
				"data":    "test error",
			},
			"id": "0",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	return httptest.NewServer(handler)
}

func startTestJSONRPCServerWithoutDataField() *httptest.Server {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"jsonrpc": "2.0",
			"error": map[string]interface{}{
				"code":    -38002,
				"message": "Invalid forkchoice state",
			},
			"id": "0",
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
	return httptest.NewServer(handler)
}

func TestBaseRPCClientCallContextJSONRPCError(t *testing.T) {
	server := startTestJSONRPCServerWithDataField()
	defer server.Close()
	rpcClient, err := rpc.DialHTTP(server.URL)
	require.NoError(t, err)
	client := client.NewBaseRPCClient(rpcClient)
	var result any
	err = client.CallContext(context.Background(), &result, "test_method")
	require.Contains(t, err.Error(), "Invalid forkchoice state", "Error should contain message field")
	require.Contains(t, err.Error(), "test error", "Error should contain data field")
}

func TestBaseRPCClientCallContextJSONRPCErrorNoData(t *testing.T) {
	server := startTestJSONRPCServerWithoutDataField()
	defer server.Close()
	rpcClient, err := rpc.DialHTTP(server.URL)
	require.NoError(t, err)
	client := client.NewBaseRPCClient(rpcClient)
	var result any
	err = client.CallContext(context.Background(), &result, "test_method")
	require.Exactly(t, err.Error(), "Invalid forkchoice state", "Error should exactly match the message field")
}
