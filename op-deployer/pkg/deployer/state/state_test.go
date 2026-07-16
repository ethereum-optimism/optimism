package state

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestState_SetChainPrestateUnknownChain(t *testing.T) {
	chainID := common.HexToHash("0x01")
	prestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	st := &State{}

	err := st.SetChainPrestate(chainID, prestate)

	require.ErrorContains(t, err, "chain not found")
	require.Empty(t, st.Chains)
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

	require.NoError(t, st.SetChainPrestate(chainID, newPrestate))

	require.Len(t, st.Chains, 1)
	require.Same(t, existing, st.Chains[0])
	require.Equal(t, newPrestate, st.Chains[0].Prestate)
	require.Equal(t, contracts, st.Chains[0].OpChainContracts)
	require.Equal(t, games, st.Chains[0].AdditionalDisputeGames)
	require.Same(t, startBlock, st.Chains[0].StartBlock)
}

func TestState_SetChainCannonFallbackPrestateUnknownChain(t *testing.T) {
	chainID := common.HexToHash("0x01")
	prestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	st := &State{}

	err := st.SetChainCannonFallbackPrestate(chainID, prestate)

	require.ErrorContains(t, err, "chain not found")
	require.Empty(t, st.Chains)
}

func TestState_SetChainCannonFallbackPrestateUpdatesExistingChain(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selectedPrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	oldFallbackPrestate := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	newFallbackPrestate := common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333")
	contracts := addresses.OpChainContracts{
		OpChainCoreContracts: addresses.OpChainCoreContracts{
			SystemConfigProxy: common.HexToAddress("0x100"),
		},
	}
	deployed := ptr.New(true)
	games := []AdditionalDisputeGameState{{
		GameType:    1,
		GameAddress: common.HexToAddress("0x200"),
	}}
	allocs := &GzipData[foundry.ForgeAllocs]{}
	startBlock := &L1BlockRefJSON{
		Hash:   common.HexToHash("0x04"),
		Number: 5,
	}
	existing := &ChainState{
		ID:                     chainID,
		OpChainContracts:       contracts,
		Deployed:               deployed,
		Prestate:               selectedPrestate,
		CannonFallbackPrestate: oldFallbackPrestate,
		AdditionalDisputeGames: games,
		Allocs:                 allocs,
		StartBlock:             startBlock,
	}
	st := &State{
		Chains: []*ChainState{existing},
	}

	require.NoError(t, st.SetChainCannonFallbackPrestate(chainID, newFallbackPrestate))

	require.Len(t, st.Chains, 1)
	require.Same(t, existing, st.Chains[0])
	require.Equal(t, contracts, st.Chains[0].OpChainContracts)
	require.Same(t, deployed, st.Chains[0].Deployed)
	require.Equal(t, selectedPrestate, st.Chains[0].Prestate)
	require.Equal(t, newFallbackPrestate, st.Chains[0].CannonFallbackPrestate)
	require.Equal(t, games, st.Chains[0].AdditionalDisputeGames)
	require.Same(t, allocs, st.Chains[0].Allocs)
	require.Same(t, startBlock, st.Chains[0].StartBlock)

	require.NoError(t, st.SetChainCannonFallbackPrestate(chainID, common.Hash{}))
	require.Zero(t, st.Chains[0].CannonFallbackPrestate)
	require.Equal(t, selectedPrestate, st.Chains[0].Prestate)
}

func TestState_PrestateJSONRoundTrip(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selectedPrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	fallbackPrestate := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	st := &State{
		Chains: []*ChainState{{
			ID:                     chainID,
			Prestate:               selectedPrestate,
			CannonFallbackPrestate: fallbackPrestate,
		}},
	}

	data, err := json.Marshal(st)
	require.NoError(t, err)
	require.Contains(t, string(data), `"prestate":"`+selectedPrestate.Hex()+`"`)
	require.Contains(t, string(data), `"cannonFallbackPrestate":"`+fallbackPrestate.Hex()+`"`)

	var roundTripped State
	require.NoError(t, json.Unmarshal(data, &roundTripped))
	chain, err := roundTripped.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, selectedPrestate, chain.Prestate)
	require.Equal(t, fallbackPrestate, chain.CannonFallbackPrestate)
}

func TestState_PrestateJSONOmitsZeroValue(t *testing.T) {
	selectedPrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	fallbackPrestate := common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222")
	tests := []struct {
		name               string
		selectedPrestate   common.Hash
		fallbackPrestate   common.Hash
		wantSelected       bool
		wantCannonFallback bool
	}{
		{name: "both unset"},
		{name: "selected unset", fallbackPrestate: fallbackPrestate, wantCannonFallback: true},
		{name: "fallback unset", selectedPrestate: selectedPrestate, wantSelected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &State{
				Chains: []*ChainState{{
					ID:                     common.HexToHash("0x01"),
					Prestate:               tt.selectedPrestate,
					CannonFallbackPrestate: tt.fallbackPrestate,
				}},
			}

			data, err := json.Marshal(st)
			require.NoError(t, err)
			require.Equal(t, tt.wantSelected, strings.Contains(string(data), `"prestate"`))
			require.Equal(t, tt.wantCannonFallback, strings.Contains(string(data), `"cannonFallbackPrestate"`))
		})
	}
}

func TestState_PrestateLegacyJSONDefaultsFallbackToZero(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selectedPrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	data := []byte(`{"opChainDeployments":[{"id":"` + chainID.Hex() + `","prestate":"` + selectedPrestate.Hex() + `"}]}`)

	var st State
	require.NoError(t, json.Unmarshal(data, &st))
	chain, err := st.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, selectedPrestate, chain.Prestate)
	require.Zero(t, chain.CannonFallbackPrestate)
}

func TestState_EnsureCreate2Salt(t *testing.T) {
	t.Run("generates a salt when unset", func(t *testing.T) {
		s := &State{}
		require.NoError(t, s.EnsureCreate2Salt())
		require.NotEqual(t, common.Hash{}, s.Create2Salt, "salt should be randomised from zero")
	})

	t.Run("preserves an existing salt", func(t *testing.T) {
		existing := common.HexToHash("0x00000000000000000000000000000000000000000000000000000000000000aa")
		s := &State{Create2Salt: existing}
		require.NoError(t, s.EnsureCreate2Salt())
		require.Equal(t, existing, s.Create2Salt, "existing salt must not be regenerated")
	})

	t.Run("is idempotent across calls", func(t *testing.T) {
		s := &State{}
		require.NoError(t, s.EnsureCreate2Salt())
		first := s.Create2Salt
		require.NoError(t, s.EnsureCreate2Salt())
		require.Equal(t, first, s.Create2Salt, "second call must not change the salt")
	})
}

func TestState_CheckL1PredictInputs(t *testing.T) {
	deployer := common.HexToAddress("0x1111000000000000000000000000000000000001")
	other := common.HexToAddress("0x2222000000000000000000000000000000000002")
	opcm := common.HexToAddress("0x3333000000000000000000000000000000000003")
	otherOPCM := common.HexToAddress("0x4444000000000000000000000000000000000004")

	t.Run("unpinned state accepts any inputs", func(t *testing.T) {
		require.NoError(t, (&State{}).CheckL1PredictInputs(deployer, opcm))
	})

	t.Run("matching deployer and opcm succeeds", func(t *testing.T) {
		pinnedSender := deployer
		pinnedOPCM := opcm
		require.NoError(t, (&State{L1PredictSenderAddress: &pinnedSender, L1PredictOPCMAddress: &pinnedOPCM}).CheckL1PredictInputs(deployer, opcm))
	})

	t.Run("mismatched deployer fails", func(t *testing.T) {
		pinned := other
		err := (&State{L1PredictSenderAddress: &pinned}).CheckL1PredictInputs(deployer, opcm)
		require.ErrorContains(t, err, "deployer address mismatch")
	})

	t.Run("mismatched opcm fails", func(t *testing.T) {
		pinned := otherOPCM
		err := (&State{L1PredictOPCMAddress: &pinned}).CheckL1PredictInputs(deployer, opcm)
		require.ErrorContains(t, err, "opcm address mismatch")
	})
}

func TestState_CheckNotPrepared(t *testing.T) {
	t.Run("non-prepared state can be applied", func(t *testing.T) {
		require.NoError(t, (&State{}).CheckNotPrepared())
	})

	t.Run("prepared state cannot be applied", func(t *testing.T) {
		err := (&State{Prepared: true}).CheckNotPrepared()
		require.ErrorContains(t, err, "cannot be applied")
	})
}

func TestState_PreparedSerialization(t *testing.T) {
	t.Run("omitted when false for backward compatibility", func(t *testing.T) {
		b, err := json.Marshal(&State{})
		require.NoError(t, err)
		require.NotContains(t, string(b), "prepared")
	})

	t.Run("round-trips when set", func(t *testing.T) {
		b, err := json.Marshal(&State{Prepared: true})
		require.NoError(t, err)
		require.Contains(t, string(b), `"prepared":true`)

		var got State
		require.NoError(t, json.Unmarshal(b, &got))
		require.True(t, got.Prepared)
	})

	t.Run("absent field defaults to not prepared", func(t *testing.T) {
		var got State
		require.NoError(t, json.Unmarshal([]byte(`{"version":1}`), &got))
		require.False(t, got.Prepared)
	})
}

func TestState_IsChainDeployed(t *testing.T) {
	id := common.HexToHash("0x0a")
	other := common.HexToHash("0x0b")

	tests := []struct {
		name   string
		chains []*ChainState
		want   bool
	}{
		{"unknown chain", nil, false},
		{"deployed", []*ChainState{{ID: id, Deployed: ptr.New(true)}}, true},
		{"not yet deployed", []*ChainState{{ID: id, Deployed: ptr.New(false)}}, false},
		{"legacy pipeline with nil flag", []*ChainState{{ID: id, Deployed: nil}}, true},
		{"other id only", []*ChainState{{ID: other, Deployed: ptr.New(true)}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, (&State{Chains: tt.chains}).IsChainDeployed(id))
		})
	}
}

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
	require.NotNil(t, s.Chains[0].Deployed)
	require.False(t, *s.Chains[0].Deployed)

	// A different chain is also appended.
	s.SetChainContracts(chainB, contractsWith("0xb1"), false)
	require.Len(t, s.Chains, 2)

	// Updating an existing chain in the state replaces it in place, preserves other
	// fields set by other stages, and can flip the deployed flag.
	s.Chains[0].StartBlock = &L1BlockRefJSON{Hash: common.HexToHash("0xdead")}
	prestate := common.HexToHash("0x1234")
	fallbackPrestate := common.HexToHash("0x5678")
	s.Chains[0].Prestate = prestate
	s.Chains[0].CannonFallbackPrestate = fallbackPrestate
	s.SetChainContracts(chainA, contractsWith("0xa2"), true)
	require.Len(t, s.Chains, 2)

	got, err := s.Chain(chainA)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress("0xa2"), got.SystemConfigProxy)
	require.NotNil(t, got.Deployed)
	require.True(t, *got.Deployed)
	require.NotNil(t, got.StartBlock, "other fields must be preserved on update")
	require.Equal(t, common.HexToHash("0xdead"), got.StartBlock.Hash)
	require.Equal(t, prestate, got.Prestate, "prestate must be preserved on update")
	require.Equal(t, fallbackPrestate, got.CannonFallbackPrestate, "Cannon fallback prestate must be preserved on update")
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
