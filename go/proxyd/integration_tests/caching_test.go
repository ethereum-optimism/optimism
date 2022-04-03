package integration_tests

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/ethereum-optimism/optimism/go/proxyd"
	"github.com/stretchr/testify/require"
)

func TestCaching(t *testing.T) {
	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	hdlr := NewRPCResponseHandler(map[string]interface{}{
		"eth_chainId":          "0x420",
		"net_version":          "0x1234",
		"eth_blockNumber":      "0x64",
		"eth_getBlockByNumber": "dummy_block",
		"eth_call":             "dummy_call",
	})
	backend := NewMockBackend(hdlr)
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))
	require.NoError(t, os.Setenv("REDIS_URL", fmt.Sprintf("redis://127.0.0.1:%s", redis.Port())))
	config := ReadConfig("caching")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	// allow time for the block number fetcher to fire
	time.Sleep(1500 * time.Millisecond)

	tests := []struct {
		method       string
		params       []interface{}
		response     string
		backendCalls int
	}{
		{
			"eth_chainId",
			nil,
			"{\"jsonrpc\": \"2.0\", \"result\": \"0x420\", \"id\": 999}",
			1,
		},
		{
			"net_version",
			nil,
			"{\"jsonrpc\": \"2.0\", \"result\": \"0x1234\", \"id\": 999}",
			1,
		},
		{
			"eth_getBlockByNumber",
			[]interface{}{
				"0x1",
				true,
			},
			"{\"jsonrpc\": \"2.0\", \"result\": \"dummy_block\", \"id\": 999}",
			1,
		},
		{
			"eth_call",
			[]interface{}{
				struct {
					To string `json:"to"`
				}{
					"0x1234",
				},
				"0x60",
			},
			"{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":\"dummy_call\"}",
			1,
		},
		{
			"eth_blockNumber",
			nil,
			"{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":\"0x64\"}",
			0,
		},
		{
			"eth_call",
			[]interface{}{
				struct {
					To string `json:"to"`
				}{
					"0x1234",
				},
				"latest",
			},
			"{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":\"dummy_call\"}",
			2,
		},
		{
			"eth_call",
			[]interface{}{
				struct {
					To string `json:"to"`
				}{
					"0x1234",
				},
				"pending",
			},
			"{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":\"dummy_call\"}",
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			resRaw, _, err := client.SendRPC(tt.method, tt.params)
			require.NoError(t, err)
			resCache, _, err := client.SendRPC(tt.method, tt.params)
			require.NoError(t, err)
			RequireEqualJSON(t, []byte(tt.response), resCache)
			RequireEqualJSON(t, resRaw, resCache)
			require.Equal(t, tt.backendCalls, countRequests(backend, tt.method))
			backend.Reset()
		})
	}

	t.Run("block numbers update", func(t *testing.T) {
		hdlr.SetResponse("eth_blockNumber", "0x100")
		time.Sleep(1500 * time.Millisecond)
		resRaw, _, err := client.SendRPC("eth_blockNumber", nil)
		require.NoError(t, err)
		RequireEqualJSON(t, []byte("{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":\"0x100\"}"), resRaw)
		backend.Reset()
	})

	t.Run("nil responses should not be cached", func(t *testing.T) {
		hdlr.SetResponse("eth_getBlockByNumber", nil)
		resRaw, _, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x123"})
		require.NoError(t, err)
		resCache, _, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x123"})
		require.NoError(t, err)
		RequireEqualJSON(t, []byte("{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":null}"), resRaw)
		RequireEqualJSON(t, resRaw, resCache)
		require.Equal(t, 2, countRequests(backend, "eth_getBlockByNumber"))
	})
}

func countRequests(backend *MockBackend, name string) int {
	var count int
	for _, req := range backend.Requests() {
		if bytes.Contains(req.Body, []byte(name)) {
			count++
		}
	}
	return count
}
