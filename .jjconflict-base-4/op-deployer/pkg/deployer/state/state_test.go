package state

import (
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestBlockRef_Deserialize(t *testing.T) {
	tests := []struct {
		name                 string
		input                string
		expected             L1BlockRefJSON
		expectedErrSubString string
	}{
		{
			name:  "typical block",
			input: `{"hash":"0xd84d7e6e3de812c7e0305d52971dc7488acaa2b2611ecc5e222e6bfc350d1940","parentHash":"0xbfbf7e85c93e031b97fad589175d509631672c62f76c4b12280614cce4031ff9","number":"0x727172","timestamp":"0x67884564"}`,
			expected: L1BlockRefJSON{
				Hash:       common.HexToHash("0xd84d7e6e3de812c7e0305d52971dc7488acaa2b2611ecc5e222e6bfc350d1940"),
				ParentHash: common.HexToHash("0xbfbf7e85c93e031b97fad589175d509631672c62f76c4b12280614cce4031ff9"),
				Number:     7500146,
				Time:       1736983908,
			},
		},
		{
			name:                 "non-hex number",
			input:                `{"hash":"0xd84d7e6e3de812c7e0305d52971dc7488acaa2b2611ecc5e222e6bfc350d1940","parentHash":"0xbfbf7e85c93e031b97fad589175d509631672c62f76c4b12280614cce4031ff9","number":1234,"timestamp":2345}`,
			expectedErrSubString: "cannot unmarshal non-string",
		},
		{
			name:                 "negative number",
			input:                `{"hash":"0xd84d7e6e3de812c7e0305d52971dc7488acaa2b2611ecc5e222e6bfc350d1940","parentHash":"0xbfbf7e85c93e031b97fad589175d509631672c62f76c4b12280614cce4031ff9","number":-1234,"timestamp":2345}`,
			expectedErrSubString: "cannot unmarshal non-string",
		},
		{
			name:                 "invalid number",
			input:                `{"hash":"0xd84d7e6e3de812c7e0305d52971dc7488acaa2b2611ecc5e222e6bfc350d1940","parentHash":"0xbfbf7e85c93e031b97fad589175d509631672c62f76c4b12280614cce4031ff9","number":"foo","timestamp":"bar"}`,
			expectedErrSubString: "cannot unmarshal hex string without 0x",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			var blockRef L1BlockRefJSON
			err := json.Unmarshal([]byte(test.input), &blockRef)
			if test.expectedErrSubString != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrSubString)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.expected, blockRef)
			}
		})
	}
}

func TestBlockRef_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		input    L1BlockRefJSON
		expected string
	}{
		{
			name: "typical block",
			input: L1BlockRefJSON{
				Hash:       common.HexToHash("0xd84d7e6e3de812c7e0305d52971dc7488acaa2b2611ecc5e222e6bfc350d1940"),
				ParentHash: common.HexToHash("0xbfbf7e85c93e031b97fad589175d509631672c62f76c4b12280614cce4031ff9"),
				Number:     7500146,
				Time:       1736983908,
			},
			expected: `{"hash":"0xd84d7e6e3de812c7e0305d52971dc7488acaa2b2611ecc5e222e6bfc350d1940","number":"0x727172","parentHash":"0xbfbf7e85c93e031b97fad589175d509631672c62f76c4b12280614cce4031ff9","timestamp":"0x67884564"}`,
		},
		{
			name: "zero values",
			input: L1BlockRefJSON{
				Hash:       common.Hash{},
				ParentHash: common.Hash{},
				Number:     0,
				Time:       0,
			},
			expected: `{"hash":"0x0000000000000000000000000000000000000000000000000000000000000000","number":"0x0","parentHash":"0x0000000000000000000000000000000000000000000000000000000000000000","timestamp":"0x0"}`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			data, err := json.Marshal(test.input)
			require.NoError(t, err)
			require.JSONEq(t, test.expected, string(data))
		})
	}
}
