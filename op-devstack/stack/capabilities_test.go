package stack

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// Mock implementations for testing capabilities

type mockELNode struct {
	chainID eth.ChainID
}

func (m *mockELNode) T() devtest.T                      { return nil }
func (m *mockELNode) Logger() log.Logger                { return nil }
func (m *mockELNode) Label(key string) string           { return "" }
func (m *mockELNode) SetLabel(key, value string)        {}
func (m *mockELNode) ChainID() eth.ChainID              { return m.chainID }
func (m *mockELNode) EthClient() apis.EthClient         { return nil }
func (m *mockELNode) TransactionTimeout() time.Duration { return 0 }

type mockL2ELNode struct {
	mockELNode
	id L2ELNodeID
}

func (m *mockL2ELNode) ID() L2ELNodeID                    { return m.id }
func (m *mockL2ELNode) L2EthClient() apis.L2EthClient     { return nil }
func (m *mockL2ELNode) L2EngineClient() apis.EngineClient { return nil }
func (m *mockL2ELNode) RegistryID() ComponentID           { return ConvertL2ELNodeID(m.id).ComponentID }

var _ L2ELNode = (*mockL2ELNode)(nil)
var _ L2ELCapable = (*mockL2ELNode)(nil)
var _ Registrable = (*mockL2ELNode)(nil)

type mockRollupBoostNode struct {
	mockELNode
	id RollupBoostNodeID
}

func (m *mockRollupBoostNode) ID() RollupBoostNodeID               { return m.id }
func (m *mockRollupBoostNode) L2EthClient() apis.L2EthClient       { return nil }
func (m *mockRollupBoostNode) L2EngineClient() apis.EngineClient   { return nil }
func (m *mockRollupBoostNode) FlashblocksClient() *client.WSClient { return nil }
func (m *mockRollupBoostNode) RegistryID() ComponentID {
	return ConvertRollupBoostNodeID(m.id).ComponentID
}

var _ RollupBoostNode = (*mockRollupBoostNode)(nil)
var _ L2ELCapable = (*mockRollupBoostNode)(nil)
var _ Registrable = (*mockRollupBoostNode)(nil)

type mockOPRBuilderNode struct {
	mockELNode
	id OPRBuilderNodeID
}

func (m *mockOPRBuilderNode) ID() OPRBuilderNodeID                { return m.id }
func (m *mockOPRBuilderNode) L2EthClient() apis.L2EthClient       { return nil }
func (m *mockOPRBuilderNode) L2EngineClient() apis.EngineClient   { return nil }
func (m *mockOPRBuilderNode) FlashblocksClient() *client.WSClient { return nil }
func (m *mockOPRBuilderNode) RegistryID() ComponentID {
	return ConvertOPRBuilderNodeID(m.id).ComponentID
}

var _ OPRBuilderNode = (*mockOPRBuilderNode)(nil)
var _ L2ELCapable = (*mockOPRBuilderNode)(nil)
var _ Registrable = (*mockOPRBuilderNode)(nil)

func TestL2ELCapableKinds(t *testing.T) {
	kinds := L2ELCapableKinds()
	require.Len(t, kinds, 3)
	require.Contains(t, kinds, KindL2ELNode)
	require.Contains(t, kinds, KindRollupBoostNode)
	require.Contains(t, kinds, KindOPRBuilderNode)
}

func TestRegistryFindByCapability(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	// Register different L2 EL-capable nodes
	l2el := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewL2ELNodeID("sequencer", chainID),
	}
	rollupBoost := &mockRollupBoostNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewRollupBoostNodeID("boost", chainID),
	}
	oprBuilder := &mockOPRBuilderNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewOPRBuilderNodeID("builder", chainID),
	}

	r.RegisterComponent(l2el)
	r.RegisterComponent(rollupBoost)
	r.RegisterComponent(oprBuilder)

	// Also register a non-L2EL component
	r.Register(NewComponentID(KindL2Batcher, "batcher", chainID), "not-l2el-capable")

	// Find all L2ELCapable
	capable := RegistryFindByCapability[L2ELCapable](r)
	require.Len(t, capable, 3)
}

func TestRegistryFindByCapabilityOnChain(t *testing.T) {
	r := NewRegistry()

	chainID1 := eth.ChainIDFromUInt64(420)
	chainID2 := eth.ChainIDFromUInt64(421)

	// Nodes on chain 420
	l2el1 := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID1},
		id:         NewL2ELNodeID("sequencer", chainID1),
	}
	rollupBoost1 := &mockRollupBoostNode{
		mockELNode: mockELNode{chainID: chainID1},
		id:         NewRollupBoostNodeID("boost", chainID1),
	}

	// Node on chain 421
	l2el2 := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID2},
		id:         NewL2ELNodeID("sequencer", chainID2),
	}

	r.RegisterComponent(l2el1)
	r.RegisterComponent(rollupBoost1)
	r.RegisterComponent(l2el2)

	// Find on chain 420
	chain420 := RegistryFindByCapabilityOnChain[L2ELCapable](r, chainID1)
	require.Len(t, chain420, 2)

	// Find on chain 421
	chain421 := RegistryFindByCapabilityOnChain[L2ELCapable](r, chainID2)
	require.Len(t, chain421, 1)
}

func TestFindL2ELCapable(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	l2el := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewL2ELNodeID("sequencer", chainID),
	}
	rollupBoost := &mockRollupBoostNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewRollupBoostNodeID("boost", chainID),
	}

	r.RegisterComponent(l2el)
	r.RegisterComponent(rollupBoost)

	capable := FindL2ELCapable(r)
	require.Len(t, capable, 2)
}

func TestFindL2ELCapableOnChain(t *testing.T) {
	r := NewRegistry()

	chainID1 := eth.ChainIDFromUInt64(420)
	chainID2 := eth.ChainIDFromUInt64(421)

	l2el1 := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID1},
		id:         NewL2ELNodeID("sequencer", chainID1),
	}
	l2el2 := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID2},
		id:         NewL2ELNodeID("sequencer", chainID2),
	}

	r.RegisterComponent(l2el1)
	r.RegisterComponent(l2el2)

	chain420 := FindL2ELCapableOnChain(r, chainID1)
	require.Len(t, chain420, 1)
	require.Equal(t, chainID1, chain420[0].ChainID())
}

func TestFindL2ELCapableByKey(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	// Register a RollupBoostNode with key "sequencer"
	rollupBoost := &mockRollupBoostNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewRollupBoostNodeID("sequencer", chainID),
	}
	r.RegisterComponent(rollupBoost)

	// Should find it by key, even though it's not an L2ELNode
	found, ok := FindL2ELCapableByKey(r, "sequencer", chainID)
	require.True(t, ok)
	require.NotNil(t, found)
	require.Equal(t, chainID, found.ChainID())

	// Should not find non-existent key
	_, ok = FindL2ELCapableByKey(r, "nonexistent", chainID)
	require.False(t, ok)
}

func TestFindL2ELCapableByKey_PrefersL2ELNode(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	// Register both L2ELNode and RollupBoostNode with same key
	l2el := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewL2ELNodeID("sequencer", chainID),
	}
	rollupBoost := &mockRollupBoostNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewRollupBoostNodeID("sequencer", chainID),
	}

	r.RegisterComponent(l2el)
	r.RegisterComponent(rollupBoost)

	// Should find L2ELNode first (it's first in L2ELCapableKinds)
	found, ok := FindL2ELCapableByKey(r, "sequencer", chainID)
	require.True(t, ok)
	// Verify it's the L2ELNode by checking it's the right mock type
	_, isL2EL := found.(*mockL2ELNode)
	require.True(t, isL2EL, "expected to find L2ELNode first")
}

func TestRegistryFindByKindsTyped(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	l2el := &mockL2ELNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewL2ELNodeID("sequencer", chainID),
	}
	rollupBoost := &mockRollupBoostNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewRollupBoostNodeID("boost", chainID),
	}

	r.RegisterComponent(l2el)
	r.RegisterComponent(rollupBoost)

	// Find only L2ELNode kind
	l2els := RegistryFindByKindsTyped[L2ELCapable](r, []ComponentKind{KindL2ELNode})
	require.Len(t, l2els, 1)

	// Find both kinds
	both := RegistryFindByKindsTyped[L2ELCapable](r, []ComponentKind{KindL2ELNode, KindRollupBoostNode})
	require.Len(t, both, 2)
}

// TestPolymorphicLookupScenario demonstrates the polymorphic lookup use case
// that Phase 3 is designed to solve.
func TestPolymorphicLookupScenario(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	// Scenario: A test wants to find an L2 EL node by key "sequencer"
	// The actual node could be L2ELNode, RollupBoostNode, or OPRBuilderNode
	// depending on the test configuration.

	// Configuration 1: Using RollupBoost
	rollupBoost := &mockRollupBoostNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewRollupBoostNodeID("sequencer", chainID),
	}
	r.RegisterComponent(rollupBoost)

	// The polymorphic lookup finds the sequencer regardless of its concrete type
	sequencer, ok := FindL2ELCapableByKey(r, "sequencer", chainID)
	require.True(t, ok)
	require.NotNil(t, sequencer)

	// Can use it as L2ELCapable
	require.Equal(t, chainID, sequencer.ChainID())
	// Could call sequencer.L2EthClient(), sequencer.L2EngineClient(), etc.

	// Clear and try with OPRBuilder
	r.Clear()

	oprBuilder := &mockOPRBuilderNode{
		mockELNode: mockELNode{chainID: chainID},
		id:         NewOPRBuilderNodeID("sequencer", chainID),
	}
	r.RegisterComponent(oprBuilder)

	// Same lookup code works
	sequencer, ok = FindL2ELCapableByKey(r, "sequencer", chainID)
	require.True(t, ok)
	require.NotNil(t, sequencer)
	require.Equal(t, chainID, sequencer.ChainID())
}
