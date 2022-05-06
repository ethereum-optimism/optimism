package integration_tests

import (
	"os"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/proxyd"
	"github.com/stretchr/testify/require"
)

const (
	notWhitelistedResponse        = `{"jsonrpc":"2.0","error":{"code":-32001,"message":"rpc method is not whitelisted"},"id":999}`
	parseErrResponse              = `{"jsonrpc":"2.0","error":{"code":-32700,"message":"parse error"},"id":null}`
	invalidJSONRPCVersionResponse = `{"error":{"code":-32601,"message":"invalid JSON-RPC version"},"id":null,"jsonrpc":"2.0"}`
	invalidIDResponse             = `{"error":{"code":-32601,"message":"invalid ID"},"id":null,"jsonrpc":"2.0"}`
	invalidMethodResponse         = `{"error":{"code":-32601,"message":"no method specified"},"id":null,"jsonrpc":"2.0"}`
	invalidBatchLenResponse       = `{"error":{"code":-32601,"message":"must specify at least one batch call"},"id":null,"jsonrpc":"2.0"}`
)

func TestSingleRPCValidation(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("whitelist")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	tests := []struct {
		name string
		body string
		res  string
		code int
	}{
		{
			"body not JSON",
			"this ain't an RPC call",
			parseErrResponse,
			400,
		},
		{
			"body not RPC",
			"{\"not\": \"rpc\"}",
			invalidJSONRPCVersionResponse,
			400,
		},
		{
			"body missing RPC ID",
			"{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23]}",
			invalidIDResponse,
			400,
		},
		{
			"body has array ID",
			"{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": []}",
			invalidIDResponse,
			400,
		},
		{
			"body has object ID",
			"{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": {}}",
			invalidIDResponse,
			400,
		},
		{
			"bad method",
			"{\"jsonrpc\": \"2.0\", \"method\": 7, \"params\": [42, 23], \"id\": 1}",
			parseErrResponse,
			400,
		},
		{
			"bad JSON-RPC",
			"{\"jsonrpc\": \"1.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": 1}",
			invalidJSONRPCVersionResponse,
			400,
		},
		{
			"omitted method",
			"{\"jsonrpc\": \"2.0\", \"params\": [42, 23], \"id\": 1}",
			invalidMethodResponse,
			400,
		},
		{
			"not whitelisted method",
			"{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": 999}",
			notWhitelistedResponse,
			403,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, code, err := client.SendRequest([]byte(tt.body))
			require.NoError(t, err)
			RequireEqualJSON(t, []byte(tt.res), res)
			require.Equal(t, tt.code, code)
			require.Equal(t, 0, len(goodBackend.Requests()))
		})
	}
}

func TestBatchRPCValidation(t *testing.T) {
	goodBackend := NewMockBackend(BatchedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

	require.NoError(t, os.Setenv("GOOD_BACKEND_RPC_URL", goodBackend.URL()))

	config := ReadConfig("whitelist")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.Start(config)
	require.NoError(t, err)
	defer shutdown()

	tests := []struct {
		name     string
		body     string
		res      string
		code     int
		reqCount int
	}{
		{
			"empty batch",
			"[]",
			invalidBatchLenResponse,
			400,
			0,
		},
		{
			"bad json",
			"[{,]",
			parseErrResponse,
			400,
			0,
		},
		{
			"not object in batch",
			"[123]",
			asArray(parseErrResponse),
			200,
			0,
		},
		{
			"body not RPC",
			"[{\"not\": \"rpc\"}]",
			asArray(invalidJSONRPCVersionResponse),
			200,
			0,
		},
		{
			"body missing RPC ID",
			"[{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23]}]",
			asArray(invalidIDResponse),
			200,
			0,
		},
		{
			"body has array ID",
			"[{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": []}]",
			asArray(invalidIDResponse),
			200,
			0,
		},
		{
			"body has object ID",
			"[{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": {}}]",
			asArray(invalidIDResponse),
			200,
			0,
		},
		// this happens because we can't deserialize the method into a non
		// string value, and it blows up the parsing for the whole request.
		{
			"bad method",
			"[{\"error\":{\"code\":-32600,\"message\":\"invalid request\"},\"id\":null,\"jsonrpc\":\"2.0\"}]",
			asArray(invalidMethodResponse),
			200,
			0,
		},
		{
			"bad JSON-RPC",
			"[{\"jsonrpc\": \"1.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": 1}]",
			asArray(invalidJSONRPCVersionResponse),
			200,
			0,
		},
		{
			"omitted method",
			"[{\"jsonrpc\": \"2.0\", \"params\": [42, 23], \"id\": 1}]",
			asArray(invalidMethodResponse),
			200,
			0,
		},
		{
			"not whitelisted method",
			"[{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": 999}]",
			asArray(notWhitelistedResponse),
			200,
			0,
		},
		{
			"mixed",
			asArray(
				"{\"jsonrpc\": \"2.0\", \"method\": \"subtract\", \"params\": [42, 23], \"id\": 999}",
				"{\"jsonrpc\": \"2.0\", \"method\": \"eth_chainId\", \"params\": [], \"id\": 123}",
				"123",
			),
			asArray(
				notWhitelistedResponse,
				goodResponse,
				parseErrResponse,
			),
			200,
			1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, code, err := client.SendRequest([]byte(tt.body))
			require.NoError(t, err)
			RequireEqualJSON(t, []byte(tt.res), res)
			require.Equal(t, tt.code, code)
			require.Equal(t, tt.reqCount, len(goodBackend.Requests()))
		})
	}
}

func asArray(in ...string) string {
	return "[" + strings.Join(in, ",") + "]"
}
