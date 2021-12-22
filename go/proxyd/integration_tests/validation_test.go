package integration_tests

import (
	"github.com/ethereum-optimism/optimism/go/proxyd"
	"github.com/stretchr/testify/require"
	"testing"
)

const (
	notWhitelistedResponse        = "{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32001,\"message\":\"rpc method is not whitelisted\"},\"id\":999}"
	parseErrResponse              = "{\"jsonrpc\":\"2.0\",\"error\":{\"code\":-32700,\"message\":\"parse error\"},\"id\":null}"
	invalidJSONRPCVersionResponse = "{\"error\":{\"code\":-32601,\"message\":\"invalid JSON-RPC version\"},\"id\":null,\"jsonrpc\":\"2.0\"}"
	invalidIDResponse             = "{\"error\":{\"code\":-32601,\"message\":\"invalid ID\"},\"id\":null,\"jsonrpc\":\"2.0\"}"
	invalidMethodResponse         = "{\"error\":{\"code\":-32601,\"message\":\"no method specified\"},\"id\":null,\"jsonrpc\":\"2.0\"}"
)

func TestBadRPC(t *testing.T) {
	goodBackend := NewMockBackend(CannedResponseHandler(200, goodResponse))
	defer goodBackend.Close()

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
			require.Equal(t, 0, len(goodBackend.Requests))
		})
	}
}
