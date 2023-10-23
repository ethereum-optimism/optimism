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
	/* cacheable */
	hdlr.SetRoute("eth_chainId", "999", "0x420")
	hdlr.SetRoute("net_version", "999", "0x1234")
	hdlr.SetRoute("eth_getBlockTransactionCountByHash", "999", "eth_getBlockTransactionCountByHash")
	hdlr.SetRoute("eth_getBlockByHash", "999", "eth_getBlockByHash")
	hdlr.SetRoute("eth_getTransactionByHash", "999", "eth_getTransactionByHash")
	hdlr.SetRoute("eth_getTransactionByBlockHashAndIndex", "999", "eth_getTransactionByBlockHashAndIndex")
	hdlr.SetRoute("eth_getUncleByBlockHashAndIndex", "999", "eth_getUncleByBlockHashAndIndex")
	hdlr.SetRoute("eth_getTransactionReceipt", "999", "eth_getTransactionReceipt")
	hdlr.SetRoute("debug_getRawReceipts", "999", "debug_getRawReceipts")
	/* not cacheable */
	hdlr.SetRoute("eth_getBlockByNumber", "999", "eth_getBlockByNumber")
	hdlr.SetRoute("eth_blockNumber", "999", "eth_blockNumber")
	hdlr.SetRoute("eth_call", "999", "eth_call")

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
		/* cacheable */
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
			"eth_getBlockTransactionCountByHash",
			[]interface{}{"0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238"},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_getBlockTransactionCountByHash\", \"id\": 999}",
			1,
		},
		{
			"eth_getBlockByHash",
			[]interface{}{"0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b", "false"},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_getBlockByHash\", \"id\": 999}",
			1,
		},
		{
			"eth_getTransactionByBlockHashAndIndex",
			[]interface{}{"0xe670ec64341771606e55d6b4ca35a1a6b75ee3d5145a99d05921026d1527331", "0x55"},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_getTransactionByBlockHashAndIndex\", \"id\": 999}",
			1,
		},
		{
			"eth_getUncleByBlockHashAndIndex",
			[]interface{}{"0xb903239f8543d04b5dc1ba6579132b143087c68db1b2168786408fcbce568238", "0x90"},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_getUncleByBlockHashAndIndex\", \"id\": 999}",
			1,
		},
		/* not cacheable */
		{
			"eth_getBlockByNumber",
			[]interface{}{
				"0x1",
				true,
			},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_getBlockByNumber\", \"id\": 999}",
			2,
		},
		{
			"eth_getTransactionReceipt",
			[]interface{}{"0x85d995eba9763907fdf35cd2034144dd9d53ce32cbec21349d4b12823c6860c5"},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_getTransactionReceipt\", \"id\": 999}",
			2,
		},
		{
			"eth_getTransactionByHash",
			[]interface{}{"0x88df016429689c079f3b2f6ad39fa052532c56795b733da78a91ebe6a713944b"},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_getTransactionByHash\", \"id\": 999}",
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
				"0x60",
			},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_call\", \"id\": 999}",
			2,
		},
		{
			"eth_blockNumber",
			nil,
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_blockNumber\", \"id\": 999}",
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
				"latest",
			},
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_call\", \"id\": 999}",
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
			"{\"jsonrpc\": \"2.0\", \"result\": \"eth_call\", \"id\": 999}",
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

	t.Run("nil responses should not be cached", func(t *testing.T) {
		hdlr.SetRoute("eth_getBlockByHash", "999", nil)
		resRaw, _, err := client.SendRPC("eth_getBlockByHash", []interface{}{"0x123"})
		require.NoError(t, err)
		resCache, _, err := client.SendRPC("eth_getBlockByHash", []interface{}{"0x123"})
		require.NoError(t, err)
		RequireEqualJSON(t, []byte("{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":null}"), resRaw)
		RequireEqualJSON(t, resRaw, resCache)
		require.Equal(t, 2, countRequests(backend, "eth_getBlockByHash"))
	})

	t.Run("debug_getRawReceipts with 0 receipts should not be cached", func(t *testing.T) {
		backend.Reset()
		hdlr.SetRoute("debug_getRawReceipts", "999", []string{})
		resRaw, _, err := client.SendRPC("debug_getRawReceipts", []interface{}{"0x88420081ab9c6d50dc57af36b541c6b8a7b3e9c0d837b0414512c4c5883560ff"})
		require.NoError(t, err)
		resCache, _, err := client.SendRPC("debug_getRawReceipts", []interface{}{"0x88420081ab9c6d50dc57af36b541c6b8a7b3e9c0d837b0414512c4c5883560ff"})
		require.NoError(t, err)
		RequireEqualJSON(t, []byte("{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":[]}"), resRaw)
		RequireEqualJSON(t, resRaw, resCache)
		require.Equal(t, 2, countRequests(backend, "debug_getRawReceipts"))
	})

	t.Run("debug_getRawReceipts with more than 0 receipts should be cached", func(t *testing.T) {
		backend.Reset()
		hdlr.SetRoute("debug_getRawReceipts", "999", []string{"a"})
		resRaw, _, err := client.SendRPC("debug_getRawReceipts", []interface{}{"0x88420081ab9c6d50dc57af36b541c6b8a7b3e9c0d837b0414512c4c5883560bb"})
		require.NoError(t, err)
		resCache, _, err := client.SendRPC("debug_getRawReceipts", []interface{}{"0x88420081ab9c6d50dc57af36b541c6b8a7b3e9c0d837b0414512c4c5883560bb"})
		require.NoError(t, err)
		RequireEqualJSON(t, []byte("{\"id\":999,\"jsonrpc\":\"2.0\",\"result\":[\"a\"]}"), resRaw)
		RequireEqualJSON(t, resRaw, resCache)
		require.Equal(t, 1, countRequests(backend, "debug_getRawReceipts"))
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
	hdlr.SetRoute("eth_getBlockByHash", "1", "eth_getBlockByHash")

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
	goodEthGetBlockByHash := "{\"jsonrpc\": \"2.0\", \"result\": \"eth_getBlockByHash\", \"id\": 1}"

	res, _, err := client.SendBatchRPC(
		NewRPCReq("1", "eth_chainId", nil),
		NewRPCReq("1", "net_version", nil),
		NewRPCReq("1", "eth_getBlockByHash", []interface{}{"0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b", "false"}),
	)
	require.NoError(t, err)
	RequireEqualJSON(t, []byte(asArray(goodChainIdResponse, goodNetVersionResponse, goodEthGetBlockByHash)), res)
	require.Equal(t, 1, countRequests(backend, "eth_chainId"))
	require.Equal(t, 1, countRequests(backend, "net_version"))
	require.Equal(t, 1, countRequests(backend, "eth_getBlockByHash"))

	backend.Reset()
	res, _, err = client.SendBatchRPC(
		NewRPCReq("1", "eth_chainId", nil),
		NewRPCReq("1", "eth_call", []interface{}{`{"to":"0x1234"}`, "pending"}),
		NewRPCReq("1", "net_version", nil),
		NewRPCReq("1", "eth_getBlockByHash", []interface{}{"0xc6ef2fc5426d6ad6fd9e2a26abeab0aa2411b7ab17f30a99d3cb96aed1d1055b", "false"}),
	)
	require.NoError(t, err)
	RequireEqualJSON(t, []byte(asArray(goodChainIdResponse, goodEthCallResponse, goodNetVersionResponse, goodEthGetBlockByHash)), res)
	require.Equal(t, 0, countRequests(backend, "eth_chainId"))
	require.Equal(t, 0, countRequests(backend, "net_version"))
	require.Equal(t, 0, countRequests(backend, "eth_getBlockByHash"))
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
