package integration_tests

import (
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/go/proxyd"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestDaisychainBadConfig ensures that the daisychain
// fails when there is bad config
func TestDaisychainBadConfig(t *testing.T) {
	hdlr := NewRPCResponseHandler(map[string]interface{}{
		"eth_chainId": "0x420",
	})
	backend := NewMockBackend(hdlr)
	defer backend.Close()

	config := ReadConfig("daisychain_bad")
	shutdown, err := proxyd.StartDaisyChain(config)
	require.Error(t, err)
	defer shutdown()
}

// TestDaisychainRequests tests various RPC requests
func TestDaisychainRequests(t *testing.T) {
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
	config := ReadConfig("daisychain")
	client := NewProxydClient("http://127.0.0.1:8545")
	shutdown, err := proxyd.StartDaisyChain(config)
	require.NoError(t, err)
	defer shutdown()

	tests := []struct {
		method       string
		params       []interface{}
		response     string
		backendCalls int
	}{
		{
			"eth_chainId",
			nil,
			`{"jsonrpc": "2.0", "result": "0x420", "id": 999}`,
			1,
		},
		{
			"net_version",
			nil,
			`{"jsonrpc": "2.0", "result": "0x1234", "id": 999}`,
			1,
		},
		{
			"eth_getBlockByNumber",
			[]interface{}{
				"0x1",
				true,
			},
			`{"jsonrpc": "2.0", "result": "dummy_block", "id": 999}`,
			1,
		},
		{
			"eth_call",
			[]interface{}{
				common.HexToAddress("0x646dB8ffC21e7ddc2B6327448dd9Fa560Df41087"),
				proxyd.TransactionArgs{},
				"0x60",
			},
			`{"id":999,"jsonrpc":"2.0","result":"dummy_call"}`,
			1,
		},
		{
			"eth_blockNumber",
			nil,
			`{"id":999,"jsonrpc":"2.0","result":"0x64"}`,
			0,
		},
		{
			"eth_call",
			[]interface{}{
				common.HexToAddress("0x646dB8ffC21e7ddc2B6327448dd9Fa560Df41087"),
				proxyd.TransactionArgs{},
				"latest",
			},
			`{"id":999,"jsonrpc":"2.0","result":"dummy_call"}`,
			2,
		},
		{
			"eth_call",
			[]interface{}{
				common.HexToAddress("0x646dB8ffC21e7ddc2B6327448dd9Fa560Df41087"),
				proxyd.TransactionArgs{},
				"latest",
			},
			`{"id":999,"jsonrpc":"2.0","result":"dummy_call"}`,
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			res, _, err := client.SendRPC(tt.method, tt.params)
			require.NoError(t, err)
			require.NoError(t, err)
			RequireEqualJSON(t, []byte(tt.response), res)
			backend.Reset()
		})
	}
}
