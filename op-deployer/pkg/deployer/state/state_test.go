package state

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ethereum-optimism/optimism/op-chain-ops/addresses"
	"github.com/ethereum-optimism/optimism/op-service/ptr"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"
)

func TestState_PrestateJSONRoundTrip(t *testing.T) {
	chainID := common.HexToHash("0x01")
	selectedPrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	st := &State{
		Chains: []*ChainState{{
			ID:       chainID,
			Prestate: selectedPrestate,
		}},
	}

	data, err := json.Marshal(st)
	require.NoError(t, err)
	require.Contains(t, string(data), `"prestate":"`+selectedPrestate.Hex()+`"`)

	var roundTripped State
	require.NoError(t, json.Unmarshal(data, &roundTripped))
	chain, err := roundTripped.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, selectedPrestate, chain.Prestate)
}

func TestState_ContinuationJSONRoundTrip(t *testing.T) {
	chainID := common.HexToHash("0x01")
	st := &State{Chains: []*ChainState{{
		ID: chainID,
		Continuation: &ContinuationState{
			LiveValidated: true,
		},
	}}}

	data, err := json.Marshal(st)
	require.NoError(t, err)
	require.Contains(t, string(data), `"liveValidated":true`)

	var roundTripped State
	require.NoError(t, json.Unmarshal(data, &roundTripped))
	chain, err := roundTripped.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, st.Chains[0].Continuation, chain.Continuation)
}

func TestState_PrestateJSONOmitsZeroValue(t *testing.T) {
	selectedPrestate := common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111")
	tests := []struct {
		name             string
		selectedPrestate common.Hash
		wantSelected     bool
	}{
		{name: "unset"},
		{name: "set", selectedPrestate: selectedPrestate, wantSelected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &State{
				Chains: []*ChainState{{
					ID:       common.HexToHash("0x01"),
					Prestate: tt.selectedPrestate,
				}},
			}

			data, err := json.Marshal(st)
			require.NoError(t, err)
			require.Equal(t, tt.wantSelected, strings.Contains(string(data), `"prestate"`))
		})
	}
}

func TestState_StartingAnchorRootJSONRoundTrip(t *testing.T) {
	chainID := common.HexToHash("0x01")
	proposal := &StartingAnchorProposal{
		Root:             common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222"),
		L2SequenceNumber: 0,
	}
	st := &State{
		Chains: []*ChainState{{
			ID:                 chainID,
			StartingAnchorRoot: proposal,
		}},
	}

	data, err := json.Marshal(st)
	require.NoError(t, err)
	require.Contains(t, string(data), `"startingAnchorRoot":{"root":"`+proposal.Root.Hex()+`","l2SequenceNumber":"0x0"}`)

	var roundTripped State
	require.NoError(t, json.Unmarshal(data, &roundTripped))
	chain, err := roundTripped.Chain(chainID)
	require.NoError(t, err)
	require.Equal(t, proposal, chain.StartingAnchorRoot)
}

func TestStartingAnchorProposal_RejectsOversizedSequenceNumber(t *testing.T) {
	var proposal StartingAnchorProposal
	err := json.Unmarshal(
		[]byte(`{"l2SequenceNumber":"0x10000000000000000"}`),
		&proposal,
	)
	require.ErrorContains(t, err, "hex number > 64 bits")
}

func TestState_StartingAnchorRootJSONBackwardCompatibility(t *testing.T) {
	chainID := common.HexToHash("0x01")

	t.Run("nil omitted", func(t *testing.T) {
		st := &State{
			Chains: []*ChainState{{
				ID: chainID,
			}},
		}

		data, err := json.Marshal(st)
		require.NoError(t, err)
		require.NotContains(t, string(data), `"startingAnchorRoot"`)
	})

	t.Run("legacy JSON decodes to nil", func(t *testing.T) {
		data := []byte(`{"opChainDeployments":[{"id":"` + chainID.Hex() + `"}]}`)

		var st State
		require.NoError(t, json.Unmarshal(data, &st))
		chain, err := st.Chain(chainID)
		require.NoError(t, err)
		require.Nil(t, chain.StartingAnchorRoot)
	})
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
		require.NoError(t, (&State{PreparedDeployment: &PreparedDeployment{Deployer: deployer, OPCM: opcm}}).CheckL1PredictInputs(deployer, opcm))
	})

	t.Run("mismatched deployer fails", func(t *testing.T) {
		err := (&State{PreparedDeployment: &PreparedDeployment{Deployer: other, OPCM: opcm}}).CheckL1PredictInputs(deployer, opcm)
		require.ErrorContains(t, err, "deployer address mismatch")
	})

	t.Run("mismatched opcm fails", func(t *testing.T) {
		err := (&State{PreparedDeployment: &PreparedDeployment{Deployer: deployer, OPCM: otherOPCM}}).CheckL1PredictInputs(deployer, opcm)
		require.ErrorContains(t, err, "opcm address mismatch")
	})
}

func TestState_CheckNotPrepared(t *testing.T) {
	t.Run("non-prepared state can be applied", func(t *testing.T) {
		require.NoError(t, (&State{}).CheckNotPrepared())
	})

	t.Run("prepared state cannot be applied", func(t *testing.T) {
		err := (&State{PreparedDeployment: new(PreparedDeployment)}).CheckNotPrepared()
		require.ErrorContains(t, err, "cannot be applied")
	})
}

func TestState_PreparedSerialization(t *testing.T) {
	t.Run("omitted when absent", func(t *testing.T) {
		b, err := json.Marshal(&State{})
		require.NoError(t, err)
		require.NotContains(t, string(b), "preparedDeployment")
	})

	t.Run("round-trips when set", func(t *testing.T) {
		prepared := &PreparedDeployment{Deployer: common.Address{0x01}, OPCM: common.Address{0x02}}
		b, err := json.Marshal(&State{PreparedDeployment: prepared})
		require.NoError(t, err)
		require.Contains(t, string(b), `"preparedDeployment"`)

		var got State
		require.NoError(t, json.Unmarshal(b, &got))
		require.Equal(t, prepared, got.PreparedDeployment)
	})

	t.Run("absent field defaults to nil", func(t *testing.T) {
		var got State
		require.NoError(t, json.Unmarshal([]byte(`{"version":1}`), &got))
		require.Nil(t, got.PreparedDeployment)
	})
}

func TestChainState_GenesisTimeSerialization(t *testing.T) {
	t.Run("omitted when unset for backward compatibility", func(t *testing.T) {
		b, err := json.Marshal(&ChainState{})
		require.NoError(t, err)
		require.NotContains(t, string(b), "genesisTime")
	})

	t.Run("round trip when set", func(t *testing.T) {
		genesisTime := hexutil.Uint64(1_750_000_000)
		b, err := json.Marshal(&ChainState{GenesisTime: &genesisTime})
		require.NoError(t, err)
		require.Contains(t, string(b), `"genesisTime":"0x684ee180"`) // 1_750_000_000 in hex

		var got ChainState
		require.NoError(t, json.Unmarshal(b, &got))
		require.NotNil(t, got.GenesisTime)
		require.Equal(t, genesisTime, *got.GenesisTime)
	})

	t.Run("absent field defaults to nil", func(t *testing.T) {
		var got ChainState
		require.NoError(t, json.Unmarshal([]byte(`{"startBlock":null}`), &got))
		require.Nil(t, got.GenesisTime)
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
	s.Chains[0].StartBlock = &L1BlockRefJSON{Hash: common.HexToHash("0xfeed")}
	prestate := common.HexToHash("0x1234")
	s.Chains[0].Prestate = prestate
	startingAnchorRoot := &StartingAnchorProposal{
		Root:             common.HexToHash("0x5678"),
		L2SequenceNumber: 9,
	}
	s.Chains[0].StartingAnchorRoot = startingAnchorRoot
	s.SetChainContracts(chainA, contractsWith("0xa2"), true)
	require.Len(t, s.Chains, 2)

	got, err := s.Chain(chainA)
	require.NoError(t, err)
	require.Equal(t, common.HexToAddress("0xa2"), got.SystemConfigProxy)
	require.NotNil(t, got.Deployed)
	require.True(t, *got.Deployed)
	require.NotNil(t, got.StartBlock, "other fields must be preserved on update")
	require.Equal(t, common.HexToHash("0xfeed"), got.StartBlock.Hash)
	require.Equal(t, prestate, got.Prestate, "prestate must be preserved on update")
	require.Equal(t, startingAnchorRoot, got.StartingAnchorRoot, "starting anchor root must be preserved on update")
}

func TestState_PinChainAnchor(t *testing.T) {
	id := common.HexToHash("0x0a")
	anchor := &L1BlockRefJSON{Hash: common.HexToHash("0xa11c"), Number: 100, Time: 5000}

	t.Run("creates an entry marked as not deployed for an unknown chain", func(t *testing.T) {
		s := &State{}
		s.PinChainAnchor(id, anchor, 5600)
		require.Len(t, s.Chains, 1)
		got := s.Chains[0]
		require.Equal(t, id, got.ID)
		require.Equal(t, anchor, got.StartBlock)
		require.NotNil(t, got.GenesisTime)
		require.EqualValues(t, 5600, *got.GenesisTime)
		require.NotNil(t, got.Deployed)
		require.False(t, *got.Deployed, "an entry created at pin time must not read as deployed")
		require.False(t, s.IsChainDeployed(id))
	})

	t.Run("updates an existing entry in place, preserving other fields", func(t *testing.T) {
		// Simulates a different stage pinning the dry-run predicted addresses.
		var contracts addresses.OpChainContracts
		contracts.SystemConfigProxy = common.HexToAddress("0xbeef")
		s := &State{}
		s.SetChainContracts(id, contracts, false)

		s.PinChainAnchor(id, anchor, 5600)
		require.Len(t, s.Chains, 1)
		got := s.Chains[0]
		require.Equal(t, anchor, got.StartBlock)
		require.EqualValues(t, 5600, *got.GenesisTime)
		require.Equal(t, common.HexToAddress("0xbeef"), got.SystemConfigProxy, "contracts must be preserved")
		require.NotNil(t, got.Deployed)
		require.False(t, *got.Deployed, "the deployed flag must not be touched on update")
	})
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
