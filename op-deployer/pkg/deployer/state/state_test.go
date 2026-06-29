package state

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestState_SetChainContracts(t *testing.T) {
	chainA := common.HexToHash("0x0a")
	chainB := common.HexToHash("0x0b")

	contractsWith := func(systemConfig string) addresses.OpChainContracts {
		var c addresses.OpChainContracts
		c.SystemConfigProxy = common.HexToAddress(systemConfig)
		return c
	}

	s := &State{}

	// A new chain is appended, we mark a  predicted entry as not-deployed.
	s.SetChainContracts(chainA, contractsWith("0xa1"), false)
	require.Len(t, s.Chains, 1)
	require.Equal(t, chainA, s.Chains[0].ID)
	require.Equal(t, common.HexToAddress("0xa1"), s.Chains[0].SystemConfigProxy)
	require.False(t, s.Chains[0].Deployed)

	// A different chain is also appended.
	s.SetChainContracts(chainB, contractsWith("0xb1"), false)
	require.Len(t, s.Chains, 2)

	// Updating an existing chain in the state replaces it in place, preserves other
	// fields set by other stages, and can flip the deployed flag.
	s.Chains[0].StartBlock = &L1BlockRefJSON{Hash: common.HexToHash("0xdead")}
	s.SetChainContracts(chainA, contractsWith("0xa2"), true)
	require.Len(t, s.Chains, 2)

	got, err := s.Chain(chainA)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress("0xa2"), got.SystemConfigProxy)
	require.True(t, got.Deployed)
	require.NotNil(t, got.StartBlock, "other fields must be preserved on update")
	require.Equal(t, common.HexToHash("0xdead"), got.StartBlock.Hash)
}

func TestState_SetChainPrestate(t *testing.T) {
	chainA := common.HexToHash("0x0a")
	prestate := common.HexToHash("0xabc123")

	s := &State{}

	// Setting on an absent chain appends a new entry.
	s.SetChainPrestate(chainA, prestate)
	require.Len(t, s.Chains, 1)
	require.Equal(t, prestate, s.Chains[0].Prestate)

	// Setting on an existing chain updates in place and preserves other fields.
	var contracts addresses.OpChainContracts
	contracts.SystemConfigProxy = common.HexToAddress("0xa1")
	s.SetChainContracts(chainA, contracts, false)
	newPrestate := common.HexToHash("0xdef456")
	s.SetChainPrestate(chainA, newPrestate)
	require.Len(t, s.Chains, 1)
	require.Equal(t, newPrestate, s.Chains[0].Prestate)
	require.Equal(t, common.HexToAddress("0xa1"), s.Chains[0].SystemConfigProxy, "other fields must be preserved")

	// SetChainContracts must preserve a previously set prestate.
	s.SetChainContracts(chainA, contracts, true)
	require.Equal(t, newPrestate, s.Chains[0].Prestate, "prestate must survive a contracts update")
}

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
