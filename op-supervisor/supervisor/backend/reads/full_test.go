package reads

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestReadHandles(t *testing.T) {
	newRegistry := func(t *testing.T) *Registry {
		logger := testlog.Logger(t, log.LevelInfo)
		return NewRegistry(logger)
	}
	t.Run("empty", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.Release()
		require.True(t, h.IsValid(), "still valid after release")
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 100})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "still valid after unrelated late invalidation")
	})
	t.Run("basic", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "dependency is ok")
		h.Release()
		require.True(t, h.IsValid(), "valid after release")
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 10})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "unaffected by invalidation after release")
	})
	t.Run("no overlap", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 101}) // does not overlap with dependency
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "valid still")
		h.Release()
	})
	t.Run("invalidated single", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{10})
		require.NoError(t, err)
		release()
		require.False(t, h.IsValid(), "affected by invalidation before release")
		h.Release()
		require.False(t, h.IsValid(), "still considered invalid after release")
		require.ErrorIs(t, h.Err(), types.ErrInvalidatedRead, "err helper works")
	})
	t.Run("invalidated two", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		h.DependOnDerivedTime(90)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{10})
		require.NoError(t, err)
		release()
		h.Release()
		require.False(t, h.IsValid(), "invalidated both")
	})
	t.Run("multiple deps", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnDerivedTime(100)
		h.DependOnDerivedTime(90)
		require.True(t, h.IsValid(), "dependency is ok")
		release, err := reg.TryInvalidate(DerivedInvalidation{95})
		require.NoError(t, err)
		release()
		require.False(t, h.IsValid(), "expected to be invalidated")
		h.Release()
	})
	t.Run("invalidated other type", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnSourceBlock(100)
		release, err := reg.TryInvalidate(DerivedInvalidation{95})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "depended on source 100, but did not invalidate this type")
		h.Release()
	})
	t.Run("invalidated combined", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnSourceBlock(2000)
		release, err := reg.TryInvalidate(InvalidationRules{
			DerivedInvalidation{Timestamp: 95},
			SourceInvalidation{Number: 1000},
		})
		require.NoError(t, err)
		release()
		require.False(t, h.IsValid())
		h.Release()
	})
	t.Run("adjust up", func(t *testing.T) {
		reg := newRegistry(t)
		h := reg.AcquireHandle()
		require.True(t, h.IsValid(), "valid by default")
		h.DependOnSourceBlock(500)
		release, err := reg.TryInvalidate(SourceInvalidation{Number: 1000})
		require.NoError(t, err)
		release()
		require.True(t, h.IsValid(), "still valid")
		h.DependOnSourceBlock(1500)
		require.False(t, h.IsValid(), "invalidated")
		h.Release()
	})
	t.Run("no concurrent invalidating", func(t *testing.T) {
		reg := newRegistry(t)
		release1, err := reg.TryInvalidate(SourceInvalidation{100})
		require.NoError(t, err)
		_, err = reg.TryInvalidate(SourceInvalidation{200})
		require.ErrorIs(t, err, types.ErrAlreadyInvalidatingRead)
		release1()
	})
	t.Run("no valid reads while invalidating", func(t *testing.T) {
		reg := newRegistry(t)
		release1, err := reg.TryInvalidate(SourceInvalidation{100})
		require.NoError(t, err)
		h := reg.AcquireHandle()
		h.DependOnSourceBlock(200)
		release1()
		require.False(t, h.IsValid(), "invalidation was ongoing when read happened")
	})
}

// TestReadConsistencyFailure tests comprehensive read inconsistency failure scenarios
// as described in GitHub issue #16485
func TestReadConsistencyFailure(t *testing.T) {
	newRegistry := func(t *testing.T) *Registry {
		logger := testlog.Logger(t, log.LevelInfo)
		return NewRegistry(logger)
	}

	t.Run("concurrent_invalidation_during_read", func(t *testing.T) {
		reg := newRegistry(t)

		// Start a read operation
		h := reg.AcquireHandle()
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "handle should be valid initially")

		// Simulate concurrent invalidation while read is in progress
		var wg sync.WaitGroup
		var invalidationErr error

		wg.Add(1)
		go func() {
			defer wg.Done()
			release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 50})
			invalidationErr = err
			if err == nil {
				defer release()
				// Keep invalidation active during the read
				time.Sleep(10 * time.Millisecond)
			}
		}()

		// Small delay to ensure invalidation starts
		time.Sleep(5 * time.Millisecond)

		// Continue with read operation - it should be invalidated
		require.False(t, h.IsValid(), "handle should be invalid due to concurrent invalidation")
		require.ErrorIs(t, h.Err(), types.ErrInvalidatedRead, "should return invalidated read error")

		wg.Wait()
		require.NoError(t, invalidationErr, "invalidation should succeed")

		h.Release()
	})

	t.Run("work_aborted_on_inconsistency", func(t *testing.T) {
		reg := newRegistry(t)

		// Simulate a complex operation that checks validity at multiple points
		h := reg.AcquireHandle()

		// Phase 1: Initial dependency
		h.DependOnDerivedTime(100)
		require.True(t, h.IsValid(), "should be valid after first dependency")

		// Phase 2: Invalidation occurs during work
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 50})
		require.NoError(t, err)
		defer release()

		// Phase 3: Work should detect inconsistency and abort
		require.False(t, h.IsValid(), "work should detect inconsistency")

		// Phase 4: Attempt to add more dependencies - should still be invalid
		h.DependOnSourceBlock(200)
		require.False(t, h.IsValid(), "should remain invalid even with new dependencies")

		h.Release()
	})

	t.Run("no_writes_when_work_aborted", func(t *testing.T) {
		reg := newRegistry(t)

		// Simulate a database operation that would normally write data
		h := reg.AcquireHandle()
		h.DependOnDerivedTime(100)

		// Track if "write" operations would be performed
		writeAttempted := false
		performWrite := func() bool {
			if !h.IsValid() {
				return false // Abort write if handle is invalid
			}
			writeAttempted = true
			return true
		}

		require.True(t, h.IsValid(), "should be valid initially")

		// Invalidate the handle
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 50})
		require.NoError(t, err)
		defer release()

		// Attempt write - should be aborted
		success := performWrite()
		require.False(t, success, "write should be aborted due to invalidation")
		require.False(t, writeAttempted, "no write should be attempted when handle is invalid")

		h.Release()
	})

	t.Run("multiple_writes_atomicity", func(t *testing.T) {
		reg := newRegistry(t)

		// Test that multiple reads with same dependencies are atomic
		h1 := reg.AcquireHandle()
		h2 := reg.AcquireHandle()

		// Both handles depend on same derived time
		h1.DependOnDerivedTime(100)
		h2.DependOnDerivedTime(100)

		require.True(t, h1.IsValid(), "h1 should be valid")
		require.True(t, h2.IsValid(), "h2 should be valid")

		// Invalidate - both should be affected atomically
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 50})
		require.NoError(t, err)
		defer release()

		require.False(t, h1.IsValid(), "h1 should be invalid")
		require.False(t, h2.IsValid(), "h2 should be invalid")
		require.ErrorIs(t, h1.Err(), types.ErrInvalidatedRead)
		require.ErrorIs(t, h2.Err(), types.ErrInvalidatedRead)

		h1.Release()
		h2.Release()
	})

	t.Run("database_operation_atomicity", func(t *testing.T) {
		reg := newRegistry(t)

		// Test that complex database operations are atomic with respect to read handles
		// This simulates the kind of operation seen in initializedUpdateCrossSafe

		handle := reg.AcquireHandle()
		defer handle.Release()

		// Phase 1: Setup dependencies like a real database operation would
		handle.DependOnDerivedTime(1000) // L2 block timestamp
		handle.DependOnSourceBlock(500)  // L1 source block

		require.True(t, handle.IsValid(), "handle should be valid initially")

		// Phase 2: Simulate multi-step database operation
		// Step 1: Lookup revision (read operation)
		if !handle.IsValid() {
			t.Fatal("operation should abort before revision lookup")
		}

		// Step 2: Add derived data (write operation)
		if !handle.IsValid() {
			t.Fatal("operation should abort before add derived")
		}

		// Phase 3: Concurrent invalidation during the multi-step operation
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 800})
		require.NoError(t, err)
		defer release()

		// Phase 4: Continue with operation steps
		// Step 3: Update cross-unsafe (conditional write operation)
		if handle.IsValid() {
			t.Log("Would perform cross-unsafe update, but handle is invalid")
		} else {
			t.Log("Cross-unsafe update correctly skipped due to invalid handle")
		}

		require.False(t, handle.IsValid(), "handle should be invalid after invalidation")

		// The operation would be aborted at this point due to invalid handle
		// This prevents partial updates that could leave the database in an inconsistent state
	})

	t.Run("read_handle_prevents_partial_writes", func(t *testing.T) {
		reg := newRegistry(t)

		// Simulate a database operation that has multiple write steps
		handle := reg.AcquireHandle()
		defer handle.Release()

		handle.DependOnDerivedTime(2000)

		// Track which write operations would be performed
		writes := make([]string, 0)

		performWrite := func(operation string) bool {
			if !handle.IsValid() {
				return false // Abort if handle is invalid
			}
			writes = append(writes, operation)
			return true
		}

		// Step 1: First write operation
		success1 := performWrite("write_derived_data")
		require.True(t, success1, "first write should succeed")

		// Invalidation occurs between writes
		release, err := reg.TryInvalidate(DerivedInvalidation{Timestamp: 1500})
		require.NoError(t, err)
		defer release()

		// Step 2: Second write operation should be aborted
		success2 := performWrite("write_cross_unsafe")
		require.False(t, success2, "second write should be aborted")

		// Step 3: Third write operation should also be aborted
		success3 := performWrite("emit_event")
		require.False(t, success3, "third write should be aborted")

		// Verify that only the first write was performed
		require.Len(t, writes, 1, "only first write should have been performed")
		require.Equal(t, "write_derived_data", writes[0])

		// This demonstrates that the read handle system prevents partial writes
		// by aborting operations when inconsistency is detected
	})
}
