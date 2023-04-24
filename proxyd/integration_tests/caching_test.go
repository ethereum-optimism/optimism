package integration_tests

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis"
	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

func TestCaching(t *testing.T) {
	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	hdlr := NewBatchRPCResponseRouter()
	hdlr.SetRoute("eth_chainId", "999", "0x420")
	hdlr.SetRoute("net_version", "999", "0x1234")
	hdlr.SetRoute("eth_blockNumber", "999", "0x64")
	hdlr.SetRoute("eth_getBlockByNumber", "999", "dummy_block")
	hdlr.SetRoute("eth_call", "999", "dummy_call")

	// mock LVC requests
	hdlr.SetFallbackRoute("eth_blockNumber", "0x64")
	hdlr.SetFallbackRoute("eth_gasPrice", "0x420")

	backend := NewMockBackend(hdlr)
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))
	require.NoError(t, os.Setenv("REDIS_URL", fmt.Sprintf("redis://127.0.0.1:%s", redis.Port())))
	config := ReadConfig("caching")
	client := NewProxydClient("http://127.0.0.1:8545")
	_, shutdown, err := proxyd.Start(config)
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
		hdlr.SetFallbackRoute("eth_blockNumber", "0x100")
		time.Sleep(1500 * time.Millisecond)
		resRaw, _, err := client.SendRPC("eth_blockNumber", nil)
		require.NoError(t, err)
		RequireEqualJSON(t, []byte("{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":\"0x100\"}"), resRaw)
		backend.Reset()
	})

	t.Run("nil responses should not be cached", func(t *testing.T) {
		hdlr.SetRoute("eth_getBlockByNumber", "999", nil)
		resRaw, _, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x123"})
		require.NoError(t, err)
		resCache, _, err := client.SendRPC("eth_getBlockByNumber", []interface{}{"0x123"})
		require.NoError(t, err)
		RequireEqualJSON(t, []byte("{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":null}"), resRaw)
		RequireEqualJSON(t, resRaw, resCache)
		require.Equal(t, 2, countRequests(backend, "eth_getBlockByNumber"))
	})
}

func TestBatchCaching(t *testing.T) {
	redis, err := miniredis.Run()
	require.NoError(t, err)
	defer redis.Close()

	hdlr := NewBatchRPCResponseRouter()
	hdlr.SetRoute("eth_chainId", "1", "0x420")
	hdlr.SetRoute("net_version", "1", "0x1234")
	hdlr.SetRoute("eth_call", "1", "dummy_call")

	// mock LVC requests
	hdlr.SetFallbackRoute("eth_blockNumber", "0x64")
	hdlr.SetFallbackRoute("eth_gasPrice", "0x420")

	backend := NewMockBackend(hdlr)
	defer backend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", backend.URL()))
	require.NoError(t, os.Setenv("REDIS_URL", fmt.Sprintf("redis://127.0.0.1:%s", redis.Port())))

	config := ReadConfig("caching")
	client := NewProxydClient("http://127.0.0.1:8545")
	_, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	// allow time for the block number fetcher to fire
	time.Sleep(1500 * time.Millisecond)

	goodChainIdResponse := "{\"jsonrpc\": \"2.0\", \"result\": \"0x420\", \"id\": 1}"
	goodNetVersionResponse := "{\"jsonrpc\": \"2.0\", \"result\": \"0x1234\", \"id\": 1}"
	goodEthCallResponse := "{\"jsonrpc\": \"2.0\", \"result\": \"dummy_call\", \"id\": 1}"

	res, _, err := client.SendBatchRPC(
		NewRPCReq("1", "eth_chainId", nil),
		NewRPCReq("1", "net_version", nil),
	)
	require.NoError(t, err)
	RequireEqualJSON(t, []byte(asArray(goodChainIdResponse, goodNetVersionResponse)), res)
	require.Equal(t, 1, countRequests(backend, "eth_chainId"))
	require.Equal(t, 1, countRequests(backend, "net_version"))

	backend.Reset()
	res, _, err = client.SendBatchRPC(
		NewRPCReq("1", "eth_chainId", nil),
		NewRPCReq("1", "eth_call", []interface{}{`{"to":"0x1234"}`, "pending"}),
		NewRPCReq("1", "net_version", nil),
	)
	require.NoError(t, err)
	RequireEqualJSON(t, []byte(asArray(goodChainIdResponse, goodEthCallResponse, goodNetVersionResponse)), res)
	require.Equal(t, 0, countRequests(backend, "eth_chainId"))
	require.Equal(t, 0, countRequests(backend, "net_version"))
	require.Equal(t, 1, countRequests(backend, "eth_call"))
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
