package stack

import (
	"sync"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// mockComponent is a test component that implements Registrable.
type mockComponent struct {
	id   ComponentID
	name string
}

func (m *mockComponent) RegistryID() ComponentID {
	return m.id
}

func requireCompletesWithoutDeadlock(t *testing.T, fn func()) {
	t.Helper()

	done := make(chan struct{})
	go func() {
		fn()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("operation timed out (likely callback executed under lock)")
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "batcher1", chainID)
	component := &mockComponent{id: id, name: "test-batcher"}

	// Register
	r.Register(id, component)

	// Get
	got, ok := r.Get(id)
	require.True(t, ok)
	require.Equal(t, component, got)

	// Check Has
	require.True(t, r.Has(id))

	// Check non-existent
	otherId := NewComponentID(KindL2Batcher, "batcher2", chainID)
	_, ok = r.Get(otherId)
	require.False(t, ok)
	require.False(t, r.Has(otherId))
}

func TestRegistry_RegisterComponent(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "batcher1", chainID)
	component := &mockComponent{id: id, name: "test-batcher"}

	// Register using RegisterComponent
	r.RegisterComponent(component)

	// Get
	got, ok := r.Get(id)
	require.True(t, ok)
	require.Equal(t, component, got)
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "batcher1", chainID)
	component := &mockComponent{id: id, name: "test-batcher"}

	r.Register(id, component)
	require.True(t, r.Has(id))

	r.Unregister(id)
	require.False(t, r.Has(id))

	// Unregistering again should be a no-op
	r.Unregister(id)
	require.False(t, r.Has(id))
}

func TestRegistry_Replace(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "batcher1", chainID)
	component1 := &mockComponent{id: id, name: "original"}
	component2 := &mockComponent{id: id, name: "replacement"}

	r.Register(id, component1)
	r.Register(id, component2) // Replace

	got, ok := r.Get(id)
	require.True(t, ok)
	require.Equal(t, component2, got)

	// Should only have one entry
	require.Equal(t, 1, r.Len())

	// Should only be in indexes once
	ids := r.IDsByKind(KindL2Batcher)
	require.Len(t, ids, 1)
}

func TestRegistry_GetByKind(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	// Register multiple batchers
	batcher1 := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher1", chainID),
		name: "batcher1",
	}
	batcher2 := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher2", chainID),
		name: "batcher2",
	}
	// Register a proposer (different kind)
	proposer := &mockComponent{
		id:   NewComponentID(KindL2Proposer, "proposer1", chainID),
		name: "proposer1",
	}

	r.Register(batcher1.id, batcher1)
	r.Register(batcher2.id, batcher2)
	r.Register(proposer.id, proposer)

	// Get batchers
	batchers := r.GetByKind(KindL2Batcher)
	require.Len(t, batchers, 2)

	// Get proposers
	proposers := r.GetByKind(KindL2Proposer)
	require.Len(t, proposers, 1)

	// Get non-existent kind
	challengers := r.GetByKind(KindL2Challenger)
	require.Len(t, challengers, 0)
}

func TestRegistry_GetByChainID(t *testing.T) {
	r := NewRegistry()

	chainID1 := eth.ChainIDFromUInt64(420)
	chainID2 := eth.ChainIDFromUInt64(421)

	// Components on chain 420
	batcher1 := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher1", chainID1),
		name: "batcher1",
	}
	proposer1 := &mockComponent{
		id:   NewComponentID(KindL2Proposer, "proposer1", chainID1),
		name: "proposer1",
	}

	// Component on chain 421
	batcher2 := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher2", chainID2),
		name: "batcher2",
	}

	r.Register(batcher1.id, batcher1)
	r.Register(proposer1.id, proposer1)
	r.Register(batcher2.id, batcher2)

	// Get all on chain 420
	chain420 := r.GetByChainID(chainID1)
	require.Len(t, chain420, 2)

	// Get all on chain 421
	chain421 := r.GetByChainID(chainID2)
	require.Len(t, chain421, 1)

	// Non-existent chain
	chain999 := r.GetByChainID(eth.ChainIDFromUInt64(999))
	require.Len(t, chain999, 0)
}

func TestRegistry_KeyOnlyComponents(t *testing.T) {
	r := NewRegistry()

	// Key-only components (like Supervisor) don't have a ChainID
	supervisor := &mockComponent{
		id:   NewComponentIDKeyOnly(KindSupervisor, "supervisor1"),
		name: "supervisor1",
	}

	r.Register(supervisor.id, supervisor)

	// Should be findable by kind
	supervisors := r.GetByKind(KindSupervisor)
	require.Len(t, supervisors, 1)

	// Should not appear in any chain index
	// (GetByChainID with zero ChainID should not return it)
	byChain := r.GetByChainID(eth.ChainID{})
	require.Len(t, byChain, 0)
}

func TestRegistry_ChainOnlyComponents(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(1)

	// Chain-only components (like L1Network) don't have a key
	network := &mockComponent{
		id:   NewComponentIDChainOnly(KindL1Network, chainID),
		name: "mainnet",
	}

	r.Register(network.id, network)

	// Should be findable by kind
	networks := r.GetByKind(KindL1Network)
	require.Len(t, networks, 1)

	// Should be findable by chain
	byChain := r.GetByChainID(chainID)
	require.Len(t, byChain, 1)
}

func TestRegistry_IDsByKind(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id1 := NewComponentID(KindL2Batcher, "batcher1", chainID)
	id2 := NewComponentID(KindL2Batcher, "batcher2", chainID)

	r.Register(id1, &mockComponent{id: id1})
	r.Register(id2, &mockComponent{id: id2})

	ids := r.IDsByKind(KindL2Batcher)
	require.Len(t, ids, 2)
	require.Contains(t, ids, id1)
	require.Contains(t, ids, id2)
}

func TestRegistry_AllAndLen(t *testing.T) {
	r := NewRegistry()

	require.Equal(t, 0, r.Len())
	require.Len(t, r.All(), 0)
	require.Len(t, r.AllIDs(), 0)

	chainID := eth.ChainIDFromUInt64(420)
	id1 := NewComponentID(KindL2Batcher, "batcher1", chainID)
	id2 := NewComponentID(KindL2Proposer, "proposer1", chainID)

	r.Register(id1, &mockComponent{id: id1})
	r.Register(id2, &mockComponent{id: id2})

	require.Equal(t, 2, r.Len())
	require.Len(t, r.All(), 2)
	require.Len(t, r.AllIDs(), 2)
}

func TestRegistry_Range(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id1 := NewComponentID(KindL2Batcher, "batcher1", chainID)
	id2 := NewComponentID(KindL2Batcher, "batcher2", chainID)

	r.Register(id1, &mockComponent{id: id1, name: "b1"})
	r.Register(id2, &mockComponent{id: id2, name: "b2"})

	// Collect all
	var collected []ComponentID
	r.Range(func(id ComponentID, component any) bool {
		collected = append(collected, id)
		return true
	})
	require.Len(t, collected, 2)

	// Early termination
	collected = nil
	r.Range(func(id ComponentID, component any) bool {
		collected = append(collected, id)
		return false // stop after first
	})
	require.Len(t, collected, 1)
}

func TestRegistry_RangeByKind(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	batcher := NewComponentID(KindL2Batcher, "batcher1", chainID)
	proposer := NewComponentID(KindL2Proposer, "proposer1", chainID)

	r.Register(batcher, &mockComponent{id: batcher})
	r.Register(proposer, &mockComponent{id: proposer})

	var collected []ComponentID
	r.RangeByKind(KindL2Batcher, func(id ComponentID, component any) bool {
		collected = append(collected, id)
		return true
	})
	require.Len(t, collected, 1)
	require.Equal(t, batcher, collected[0])
}

func TestRegistry_RangeByChainID(t *testing.T) {
	r := NewRegistry()

	chainID1 := eth.ChainIDFromUInt64(420)
	chainID2 := eth.ChainIDFromUInt64(421)

	batcher1 := NewComponentID(KindL2Batcher, "batcher1", chainID1)
	batcher2 := NewComponentID(KindL2Batcher, "batcher2", chainID2)

	r.Register(batcher1, &mockComponent{id: batcher1})
	r.Register(batcher2, &mockComponent{id: batcher2})

	var collected []ComponentID
	r.RangeByChainID(chainID1, func(id ComponentID, component any) bool {
		collected = append(collected, id)
		return true
	})
	require.Len(t, collected, 1)
	require.Equal(t, batcher1, collected[0])

	// Test early termination
	collected = nil
	r.RangeByChainID(chainID1, func(id ComponentID, component any) bool {
		collected = append(collected, id)
		return false // stop immediately
	})
	require.Len(t, collected, 1)
}

func TestRegistry_Range_CallbackCanMutateRegistry(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "batcher1", chainID)
	r.Register(id, &mockComponent{id: id})

	requireCompletesWithoutDeadlock(t, func() {
		r.Range(func(id ComponentID, component any) bool {
			r.Clear()
			return false
		})
	})

	require.Equal(t, 0, r.Len())
}

func TestRegistry_RangeByKind_CallbackCanMutateRegistry(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	oldID := NewComponentID(KindL2Batcher, "batcher1", chainID)
	newID := NewComponentID(KindL2Batcher, "batcher2", chainID)
	r.Register(oldID, &mockComponent{id: oldID})

	requireCompletesWithoutDeadlock(t, func() {
		r.RangeByKind(KindL2Batcher, func(id ComponentID, component any) bool {
			r.Unregister(oldID)
			r.Register(newID, &mockComponent{id: newID})
			return false
		})
	})

	require.False(t, r.Has(oldID))
	require.True(t, r.Has(newID))
}

func TestRegistry_RangeByChainID_CallbackCanMutateRegistry(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	oldID := NewComponentID(KindL2Batcher, "batcher1", chainID)
	newID := NewComponentID(KindL2Batcher, "batcher2", chainID)
	r.Register(oldID, &mockComponent{id: oldID})

	requireCompletesWithoutDeadlock(t, func() {
		r.RangeByChainID(chainID, func(id ComponentID, component any) bool {
			r.Unregister(oldID)
			r.Register(newID, &mockComponent{id: newID})
			return false
		})
	})

	require.False(t, r.Has(oldID))
	require.True(t, r.Has(newID))
}

func TestRegistry_Clear(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "batcher1", chainID)
	r.Register(id, &mockComponent{id: id})

	require.Equal(t, 1, r.Len())

	r.Clear()

	require.Equal(t, 0, r.Len())
	require.False(t, r.Has(id))
	require.Len(t, r.GetByKind(KindL2Batcher), 0)
	require.Len(t, r.GetByChainID(chainID), 0)
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	var wg sync.WaitGroup
	numGoroutines := 100

	// Concurrent writes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := NewComponentID(KindL2Batcher, string(rune('a'+i%26)), chainID)
			r.Register(id, &mockComponent{id: id})
		}(i)
	}

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = r.GetByKind(KindL2Batcher)
			_ = r.GetByChainID(chainID)
			_ = r.Len()
		}()
	}

	wg.Wait()

	// Should have some components (exact count depends on key collisions)
	require.Greater(t, r.Len(), 0)
}

// Tests for type-safe generic accessor functions

func TestRegistryGet_TypeSafe(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewL2BatcherID2("batcher1", chainID)
	component := &mockComponent{id: id.ComponentID, name: "test-batcher"}

	RegistryRegister(r, id, component)

	// Type-safe get
	got, ok := RegistryGet[*mockComponent](r, id)
	require.True(t, ok)
	require.Equal(t, component, got)

	// Wrong type should fail
	gotStr, ok := RegistryGet[string](r, id)
	require.False(t, ok)
	require.Equal(t, "", gotStr)
}

func TestRegistryGetByKind_TypeSafe(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	batcher1 := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher1", chainID),
		name: "batcher1",
	}
	batcher2 := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher2", chainID),
		name: "batcher2",
	}

	r.Register(batcher1.id, batcher1)
	r.Register(batcher2.id, batcher2)

	// Type-safe get by kind
	batchers := RegistryGetByKind[*mockComponent](r, KindL2Batcher)
	require.Len(t, batchers, 2)

	// Wrong type returns empty
	wrongType := RegistryGetByKind[string](r, KindL2Batcher)
	require.Len(t, wrongType, 0)
}

func TestRegistryGetByChainID_TypeSafe(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	batcher := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher1", chainID),
		name: "batcher1",
	}
	proposer := &mockComponent{
		id:   NewComponentID(KindL2Proposer, "proposer1", chainID),
		name: "proposer1",
	}

	r.Register(batcher.id, batcher)
	r.Register(proposer.id, proposer)

	// Get all mockComponents on chain
	components := RegistryGetByChainID[*mockComponent](r, chainID)
	require.Len(t, components, 2)
}

func TestRegistryRange_TypeSafe(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	batcher := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher1", chainID),
		name: "batcher1",
	}
	r.Register(batcher.id, batcher)

	// Also register a non-mockComponent
	r.Register(NewComponentID(KindL2Proposer, "other", chainID), "not a mockComponent")

	var collected []*mockComponent
	RegistryRange(r, func(id ComponentID, component *mockComponent) bool {
		collected = append(collected, component)
		return true
	})

	// Should only collect mockComponents
	require.Len(t, collected, 1)
	require.Equal(t, batcher, collected[0])
}

func TestRegistryRangeByKind_TypeSafe(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)

	batcher := &mockComponent{
		id:   NewComponentID(KindL2Batcher, "batcher1", chainID),
		name: "batcher1",
	}
	proposer := &mockComponent{
		id:   NewComponentID(KindL2Proposer, "proposer1", chainID),
		name: "proposer1",
	}

	r.Register(batcher.id, batcher)
	r.Register(proposer.id, proposer)

	var collected []*mockComponent
	RegistryRangeByKind(r, KindL2Batcher, func(id ComponentID, component *mockComponent) bool {
		collected = append(collected, component)
		return true
	})

	require.Len(t, collected, 1)
	require.Equal(t, batcher, collected[0])
}

func TestRegistry_UnregisterUpdatesIndexes(t *testing.T) {
	r := NewRegistry()

	chainID := eth.ChainIDFromUInt64(420)
	id := NewComponentID(KindL2Batcher, "batcher1", chainID)
	r.Register(id, &mockComponent{id: id})

	// Verify indexes before unregister
	require.Len(t, r.IDsByKind(KindL2Batcher), 1)
	require.Len(t, r.IDsByChainID(chainID), 1)

	r.Unregister(id)

	// Indexes should be updated
	require.Len(t, r.IDsByKind(KindL2Batcher), 0)
	require.Len(t, r.IDsByChainID(chainID), 0)
}
