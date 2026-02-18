package stack

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

func TestComponentID_KeyAndChain(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "mynode", chainID)

	require.Equal(t, KindL2Batcher, id.Kind())
	require.Equal(t, "mynode", id.Key())
	require.Equal(t, chainID, id.ChainID())
	require.Equal(t, IDShapeKeyAndChain, id.Shape())
	require.Equal(t, "L2Batcher-mynode-420", id.String())
}

func TestComponentID_ChainOnly(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(1)
	id := NewComponentIDChainOnly(KindL1Network, chainID)

	require.Equal(t, KindL1Network, id.Kind())
	require.Equal(t, "", id.Key())
	require.Equal(t, chainID, id.ChainID())
	require.Equal(t, IDShapeChainOnly, id.Shape())
	require.Equal(t, "L1Network-1", id.String())
}

func TestComponentID_KeyOnly(t *testing.T) {
	id := NewComponentIDKeyOnly(KindSupervisor, "mysupervisor")

	require.Equal(t, KindSupervisor, id.Kind())
	require.Equal(t, "mysupervisor", id.Key())
	require.Equal(t, eth.ChainID{}, id.ChainID())
	require.Equal(t, IDShapeKeyOnly, id.Shape())
	require.Equal(t, "Supervisor-mysupervisor", id.String())
}

func TestComponentID_MarshalRoundTrip_KeyAndChain(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)
	original := NewComponentID(KindL2Batcher, "mynode", chainID)

	data, err := original.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "L2Batcher-mynode-420", string(data))

	var parsed ComponentID
	parsed.kind = KindL2Batcher // Must set kind before unmarshal
	err = parsed.UnmarshalText(data)
	require.NoError(t, err)
	require.Equal(t, original, parsed)
}

func TestComponentID_MarshalRoundTrip_ChainOnly(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(1)
	original := NewComponentIDChainOnly(KindL1Network, chainID)

	data, err := original.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "L1Network-1", string(data))

	var parsed ComponentID
	parsed.kind = KindL1Network
	err = parsed.UnmarshalText(data)
	require.NoError(t, err)
	require.Equal(t, original, parsed)
}

func TestComponentID_MarshalRoundTrip_KeyOnly(t *testing.T) {
	original := NewComponentIDKeyOnly(KindSupervisor, "mysupervisor")

	data, err := original.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "Supervisor-mysupervisor", string(data))

	var parsed ComponentID
	parsed.kind = KindSupervisor
	err = parsed.UnmarshalText(data)
	require.NoError(t, err)
	require.Equal(t, original, parsed)
}

func TestComponentID_UnmarshalKindMismatch(t *testing.T) {
	var id ComponentID
	id.kind = KindL2Batcher
	err := id.UnmarshalText([]byte("L2ELNode-mynode-420"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected kind")
}

func TestID_TypeSafety(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)

	// Create two different ID types with same key and chainID
	batcherID := NewL2BatcherID2("mynode", chainID)
	elNodeID := NewL2ELNodeID2("mynode", chainID)

	// They should have different kinds
	require.Equal(t, KindL2Batcher, batcherID.Kind())
	require.Equal(t, KindL2ELNode, elNodeID.Kind())

	// Their string representations should be different
	require.Equal(t, "L2Batcher-mynode-420", batcherID.String())
	require.Equal(t, "L2ELNode-mynode-420", elNodeID.String())

	// The underlying ComponentIDs should be different due to kind
	require.NotEqual(t, batcherID.AsComponentID(), elNodeID.AsComponentID())
}

func TestID_MarshalRoundTrip(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)
	original := NewL2BatcherID2("mynode", chainID)

	data, err := original.MarshalText()
	require.NoError(t, err)
	require.Equal(t, "L2Batcher-mynode-420", string(data))

	// Unmarshal into a zero value - Kind() should still work
	var parsed L2BatcherID2
	require.Equal(t, KindL2Batcher, parsed.Kind()) // Works on zero value!

	err = parsed.UnmarshalText(data)
	require.NoError(t, err)
	require.Equal(t, original, parsed)
}

func TestID_UnmarshalKindMismatch(t *testing.T) {
	// Try to unmarshal an L2ELNode ID into an L2Batcher ID
	var batcherID L2BatcherID2
	err := batcherID.UnmarshalText([]byte("L2ELNode-mynode-420"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "unexpected kind")
}

func TestID_ChainOnlyTypes(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(1)
	networkID := NewL1NetworkID2(chainID)

	require.Equal(t, KindL1Network, networkID.Kind())
	require.Equal(t, chainID, networkID.ChainID())
	require.Equal(t, "L1Network-1", networkID.String())

	data, err := networkID.MarshalText()
	require.NoError(t, err)

	var parsed L1NetworkID2
	err = parsed.UnmarshalText(data)
	require.NoError(t, err)
	require.Equal(t, networkID, parsed)
}

func TestID_KeyOnlyTypes(t *testing.T) {
	supervisorID := NewSupervisorID2("mysupervisor")

	require.Equal(t, KindSupervisor, supervisorID.Kind())
	require.Equal(t, "mysupervisor", supervisorID.Key())
	require.Equal(t, "Supervisor-mysupervisor", supervisorID.String())

	data, err := supervisorID.MarshalText()
	require.NoError(t, err)

	var parsed SupervisorID2
	err = parsed.UnmarshalText(data)
	require.NoError(t, err)
	require.Equal(t, supervisorID, parsed)
}

func TestID_Sorting(t *testing.T) {
	chainID1 := eth.ChainIDFromUInt64(100)
	chainID2 := eth.ChainIDFromUInt64(200)

	ids := []L2BatcherID2{
		NewL2BatcherID2("charlie", chainID1),
		NewL2BatcherID2("alice", chainID1),
		NewL2BatcherID2("alice", chainID2),
		NewL2BatcherID2("bob", chainID1),
	}

	sorted := SortIDs(ids)

	// Should be sorted by key first, then by chainID
	require.Equal(t, "alice", sorted[0].Key())
	require.Equal(t, chainID1, sorted[0].ChainID())
	require.Equal(t, "alice", sorted[1].Key())
	require.Equal(t, chainID2, sorted[1].ChainID())
	require.Equal(t, "bob", sorted[2].Key())
	require.Equal(t, "charlie", sorted[3].Key())
}

func TestID_MapKey(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)

	// IDs should work as map keys
	m := make(map[L2BatcherID2]string)

	id1 := NewL2BatcherID2("node1", chainID)
	id2 := NewL2BatcherID2("node2", chainID)

	m[id1] = "value1"
	m[id2] = "value2"

	require.Equal(t, "value1", m[id1])
	require.Equal(t, "value2", m[id2])

	// Same key+chainID should retrieve same value
	id1Copy := NewL2BatcherID2("node1", chainID)
	require.Equal(t, "value1", m[id1Copy])
}

func TestAllIDTypes(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)

	// Test all ID constructors and their kinds
	tests := []struct {
		name     string
		id       interface{ Kind() ComponentKind }
		expected ComponentKind
	}{
		{"L1ELNode", NewL1ELNodeID2("node", chainID), KindL1ELNode},
		{"L1CLNode", NewL1CLNodeID2("node", chainID), KindL1CLNode},
		{"L1Network", NewL1NetworkID2(chainID), KindL1Network},
		{"L2ELNode", NewL2ELNodeID2("node", chainID), KindL2ELNode},
		{"L2CLNode", NewL2CLNodeID2("node", chainID), KindL2CLNode},
		{"L2Network", NewL2NetworkID2(chainID), KindL2Network},
		{"L2Batcher", NewL2BatcherID2("node", chainID), KindL2Batcher},
		{"L2Proposer", NewL2ProposerID2("node", chainID), KindL2Proposer},
		{"L2Challenger", NewL2ChallengerID2("node", chainID), KindL2Challenger},
		{"RollupBoostNode", NewRollupBoostNodeID2("node", chainID), KindRollupBoostNode},
		{"OPRBuilderNode", NewOPRBuilderNodeID2("node", chainID), KindOPRBuilderNode},
		{"Faucet", NewFaucetID2("node", chainID), KindFaucet},
		{"SyncTester", NewSyncTesterID2("node", chainID), KindSyncTester},
		{"Supervisor", NewSupervisorID2("node"), KindSupervisor},
		{"Conductor", NewConductorID2("node"), KindConductor},
		{"Cluster", NewClusterID2("node"), KindCluster},
		{"Superchain", NewSuperchainID2("node"), KindSuperchain},
		{"TestSequencer", NewTestSequencerID2("node"), KindTestSequencer},
		{"FlashblocksClient", NewFlashblocksClientID2("node", chainID), KindFlashblocksClient},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, tt.id.Kind())
		})
	}
}

// TestSerializationCompatibility verifies that the new ID system produces
// the same serialization format as the old system.
func TestSerializationCompatibility(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)

	// These formats must match the old ID system exactly
	tests := []struct {
		name     string
		id       interface{ MarshalText() ([]byte, error) }
		expected string
	}{
		{"L2Batcher", NewL2BatcherID2("mynode", chainID), "L2Batcher-mynode-420"},
		{"L2ELNode", NewL2ELNodeID2("mynode", chainID), "L2ELNode-mynode-420"},
		{"L1Network", NewL1NetworkID2(eth.ChainIDFromUInt64(1)), "L1Network-1"},
		{"Supervisor", NewSupervisorID2("mysupervisor"), "Supervisor-mysupervisor"},
		{"RollupBoostNode", NewRollupBoostNodeID2("boost", chainID), "RollupBoostNode-boost-420"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.id.MarshalText()
			require.NoError(t, err)
			require.Equal(t, tt.expected, string(data))
		})
	}
}

// TestConversionHelpers verifies that conversion between old and new ID systems
// preserves all data correctly.
func TestConversionHelpers(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)

	t.Run("L2BatcherID", func(t *testing.T) {
		old := NewL2BatcherID("mynode", chainID)
		new := ConvertL2BatcherID(old)
		require.Equal(t, KindL2Batcher, new.Kind())
		require.Equal(t, "mynode", new.Key())
		require.Equal(t, chainID, new.ChainID())
		require.Equal(t, old.String(), new.String())
	})

	t.Run("L2ELNodeID", func(t *testing.T) {
		old := NewL2ELNodeID("mynode", chainID)
		new := ConvertL2ELNodeID(old)
		require.Equal(t, KindL2ELNode, new.Kind())
		require.Equal(t, "mynode", new.Key())
		require.Equal(t, chainID, new.ChainID())
		require.Equal(t, old.String(), new.String())
	})

	t.Run("L1NetworkID", func(t *testing.T) {
		old := L1NetworkID(eth.ChainIDFromUInt64(1))
		new := ConvertL1NetworkID(old)
		require.Equal(t, KindL1Network, new.Kind())
		require.Equal(t, eth.ChainIDFromUInt64(1), new.ChainID())
		require.Equal(t, old.String(), new.String())
	})

	t.Run("SupervisorID", func(t *testing.T) {
		old := SupervisorID("mysupervisor")
		new := ConvertSupervisorID(old)
		require.Equal(t, KindSupervisor, new.Kind())
		require.Equal(t, "mysupervisor", new.Key())
		require.Equal(t, old.String(), new.String())
	})

	t.Run("RollupBoostNodeID", func(t *testing.T) {
		old := NewRollupBoostNodeID("boost", chainID)
		new := ConvertRollupBoostNodeID(old)
		require.Equal(t, KindRollupBoostNode, new.Kind())
		require.Equal(t, "boost", new.Key())
		require.Equal(t, chainID, new.ChainID())
		require.Equal(t, old.String(), new.String())
	})

	t.Run("FlashblocksWSClientID", func(t *testing.T) {
		old := NewFlashblocksWSClientID("fbclient", chainID)
		new := ConvertFlashblocksClientID(old)
		require.Equal(t, KindFlashblocksClient, new.Kind())
		require.Equal(t, "fbclient", new.Key())
		require.Equal(t, chainID, new.ChainID())
		require.Equal(t, old.String(), new.String())
	})
}
