package proxyd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRPCResJSON(t *testing.T) {
	tests := []struct {
		name string
		in   *RPCRes
		out  string
	}{
		{
			"string result",
			&RPCRes{
				JSONRPC: JSONRPCVersion,
				Result:  "foobar",
				ID:      []byte("123"),
			},
			`{"jsonrpc":"2.0","result":"foobar","id":123}`,
		},
		{
			"object result",
			&RPCRes{
				JSONRPC: JSONRPCVersion,
				Result: struct {
					Str string `json:"str"`
				}{
					"test",
				},
				ID: []byte("123"),
			},
			`{"jsonrpc":"2.0","result":{"str":"test"},"id":123}`,
		},
		{
			"nil result",
			&RPCRes{
				JSONRPC: JSONRPCVersion,
				Result:  nil,
				ID:      []byte("123"),
			},
			`{"jsonrpc":"2.0","result":null,"id":123}`,
		},
		{
			"error result without data",
			&RPCRes{
				JSONRPC: JSONRPCVersion,
				Error: &RPCErr{
					Code:    1234,
					Message: "test err",
				},
				ID: []byte("123"),
			},
			`{"jsonrpc":"2.0","error":{"code":1234,"message":"test err"},"id":123}`,
		},
		{
			"error result with data",
			&RPCRes{
				JSONRPC: JSONRPCVersion,
				Error: &RPCErr{
					Code:    1234,
					Message: "test err",
					Data:    "revert",
				},
				ID: []byte("123"),
			},
			`{"jsonrpc":"2.0","error":{"code":1234,"message":"test err","data":"revert"},"id":123}`,
		},
		{
			"string ID",
			&RPCRes{
				JSONRPC: JSONRPCVersion,
				Result:  "foobar",
				ID:      []byte("\"123\""),
			},
			`{"jsonrpc":"2.0","result":"foobar","id":"123"}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := json.Marshal(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.out, string(out))
		})
	}
}
