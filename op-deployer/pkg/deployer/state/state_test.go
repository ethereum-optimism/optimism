package state

import (
	"encoding/json"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestState_SetChainPrestateCreatesChain(t *testing.T) {
	chainID := common.HexToHash("0x01")
	prestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	st := &State{}

	st.SetChainPrestate(chainID, prestate)

	require.Len(t, st.Chains, 1)
	require.Equal(t, &ChainState{
		ID:       chainID,
		Prestate: prestate,
	}, st.Chains[0])
}

func TestState_SetChainPrestateUpdatesExistingChain(t *testing.T) {
	chainID := common.HexToHash("0x01")
	oldPrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	newPrestate := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	contracts := addresses.OpChainContracts{
		OpChainCoreContracts: addresses.OpChainCoreContracts{
			SystemConfigProxy: common.HexToAddress("0x100"),
		},
	}
	games := []AdditionalDisputeGameState{{
		GameType:    1,
		GameAddress: common.HexToAddress("0x200"),
	}}
	startBlock := &L1BlockRefJSON{
		Hash:   common.HexToHash("0x03"),
		Number: 4,
	}
	existing := &ChainState{
		ID:                     chainID,
		OpChainContracts:       contracts,
		Prestate:               oldPrestate,
		AdditionalDisputeGames: games,
		StartBlock:             startBlock,
	}
	st := &State{
		Chains: []*ChainState{existing},
	}

	st.SetChainPrestate(chainID, newPrestate)

	require.Len(t, st.Chains, 1)
	require.Same(t, existing, st.Chains[0])
	require.Equal(t, newPrestate, st.Chains[0].Prestate)
	require.Equal(t, contracts, st.Chains[0].OpChainContracts)
	require.Equal(t, games, st.Chains[0].AdditionalDisputeGames)
	require.Same(t, startBlock, st.Chains[0].StartBlock)
}

func TestState_PrestateJSONRoundTrip(t *testing.T) {
	chainID := common.HexToHash("0x01")
	prestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	st := &State{
		Chains: []*ChainState{{
			ID:       chainID,
			Prestate: prestate,
		}},
	}

	data, err := json.Marshal(st)
	require.NoError(t, err)
	require.Contains(t, string(data), `"prestate":"`+prestate.Hex()+`"`)

	var roundTripped State
	require.NoError(t, json.Unmarshal(data, &roundTripped))
	chain, err := roundTripped.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, prestate, chain.Prestate)
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
