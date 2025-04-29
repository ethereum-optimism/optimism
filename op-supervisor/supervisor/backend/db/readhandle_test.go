package db

import (
	"fmt"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/db/logs"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadHandleBasic(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Acquire a handle
	handle := registry.AcquireHandle(100)
	assert.True(t, handle.IsValid())
	assert.Equal(t, uint64(100), handle.blockNum)

	// Update block number
	assert.True(t, handle.UpdateBlock(200))
	assert.Equal(t, uint64(200), handle.blockNum)

	// Validate
	assert.NoError(t, handle.Validate())

	// Invalidate
	registry.InvalidateHandlesAfter(150)
	assert.False(t, handle.IsValid())
	assert.Error(t, handle.Validate())

	// Release
	handle.Release()
}

func TestReadHandleMultipleHandles(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Acquire multiple handles
	handle1 := registry.AcquireHandle(100)
	handle2 := registry.AcquireHandle(200)
	handle3 := registry.AcquireHandle(300)

	// All should be valid
	assert.True(t, handle1.IsValid())
	assert.True(t, handle2.IsValid())
	assert.True(t, handle3.IsValid())

	// Invalidate handles after block 150
	registry.InvalidateHandlesAfter(150)

	// handle1 should still be valid, others invalid
	assert.True(t, handle1.IsValid())
	assert.False(t, handle2.IsValid())
	assert.False(t, handle3.IsValid())

	// Clean up
	handle1.Release()
	handle2.Release()
	handle3.Release()
}

func TestReadHandleChainsDB(t *testing.T) {
	db := setupTestDB(t)
	chainID := eth.ChainID{1}

	// Test successful operation
	err := db.WithReadHandle(chainID, 100, func(handle *ReadHandle) error {
		return nil
	})
	assert.NoError(t, err)

	// Test invalidation during operation
	var wg sync.WaitGroup
	wg.Add(1)

	operationComplete := make(chan struct{})
	invalidationComplete := make(chan struct{})

	go func() {
		err := db.WithReadHandle(chainID, 100, func(handle *ReadHandle) error {
			// Signal the main thread that we're inside the operation
			close(operationComplete)

			// Wait for invalidation
			<-invalidationComplete

			// Continue and return success
			return nil
		})

		// Should return invalid handle error
		assert.Equal(t, ErrInvalidHandle, err)
		wg.Done()
	}()

	// Wait for operation to start
	<-operationComplete

	// Simulate rewind
	registry, _ := db.readRegistries.Get(chainID)
	registry.InvalidateHandlesAfter(50)

	close(invalidationComplete)
	wg.Wait()
}

func TestReadHandleChainsDBMultipleChains(t *testing.T) {
	db := setupTestDB(t)
	chain1 := eth.ChainID{1}
	chain2 := eth.ChainID{2}

	// Test successful operation with multiple handles
	err := db.WithReadHandles(
		[]eth.ChainID{chain1, chain2},
		[]uint64{100, 200},
		func(handles []*ReadHandle) error {
			assert.Len(t, handles, 2)
			assert.True(t, handles[0].IsValid())
			assert.True(t, handles[1].IsValid())
			return nil
		})
	assert.NoError(t, err)

	// Test error when chain counts don't match
	err = db.WithReadHandles(
		[]eth.ChainID{chain1, chain2},
		[]uint64{100},
		func(handles []*ReadHandle) error {
			return nil
		})
	assert.Error(t, err)

	// Test handle cleanup on error
	err = db.WithReadHandles(
		[]eth.ChainID{chain1, chain2},
		[]uint64{100, 200},
		func(handles []*ReadHandle) error {
			// Simulate rewind on chain1
			registry, _ := db.readRegistries.Get(chain1)
			registry.InvalidateHandlesAfter(50)
			return nil
		})
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidHandle)
}

func TestReadHandleValidateAccessList(t *testing.T) {
	db := setupTestDB(t)
	chain1 := eth.ChainID{1}
	chain2 := eth.ChainID{2}

	// Create test access list
	accessList := []types.Access{
		{
			BlockNumber: 100,
			Timestamp:   1000,
			LogIndex:    1,
			ChainID:     chain1,
			Checksum:    types.MessageChecksum(common.Hash{1}),
		},
		{
			BlockNumber: 200,
			Timestamp:   2000,
			LogIndex:    2,
			ChainID:     chain2,
			Checksum:    types.MessageChecksum(common.Hash{2}),
		},
	}

	// Test successful validation
	err := db.ValidateAccessList(accessList)
	assert.NoError(t, err)

	// Test validation failure on reorg
	// Use a sync channel to ensure Rewind is fully completed
	reorgDone := make(chan struct{})
	go func() {
		// Simulate reorg on chain1
		// Note: Using Number=49 so block 100 will be invalidated since Rewind
		// invalidates blocks > 49
		err := db.Rewind(chain1, eth.BlockID{Number: 49})
		assert.NoError(t, err)
		close(reorgDone)
	}()

	// Wait for reorg to complete
	<-reorgDone

	// Try validation again, should fail with InvalidHandle
	// This should fail because chain1/block100 is now invalid
	err = db.ValidateAccessList(accessList)
	assert.Error(t, err)
}

func TestReadHandleConcurrentHandles(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)
	logDB := newMockLogDB()

	// Add some test blocks
	for i := uint64(1); i <= 10; i++ {
		logDB.addBlock(i, common.Hash{byte(i)})
	}

	// Test concurrent reads and rewinds
	var wg sync.WaitGroup
	readStarted := make(chan struct{})
	rewindDone := make(chan struct{})

	// Start a goroutine that will try to read block 5
	wg.Add(1)
	go func() {
		defer wg.Done()

		// Acquire a read handle for block 5
		handle := registry.AcquireHandle(5)
		defer handle.Release()

		// Signal that we've started the read operation
		close(readStarted)

		// Wait for the rewind to happen
		<-rewindDone

		// Try to validate after rewind
		err := handle.Validate()
		assert.Error(t, err, "handle should be invalid after rewind")
		assert.Equal(t, ErrInvalidHandle, err)
	}()

	// Wait for read operation to start
	<-readStarted

	// Perform a rewind to block 3
	registry.InvalidateHandlesAfter(3)
	logDB.rewind(3)

	// Signal that rewind is complete
	close(rewindDone)

	// Wait for all operations to complete
	wg.Wait()

	// Verify that new reads after rewind work correctly
	handle := registry.AcquireHandle(3)
	assert.NoError(t, handle.Validate(), "new handle at rewind point should be valid")
	handle.Release()

	handle = registry.AcquireHandle(4)
	assert.NoError(t, handle.Validate(), "new handle after rewind point should be valid")
	handle.Release()
}

func TestReadHandleSimple(t *testing.T) {
	// Create a test registry
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Create a handle for block 5
	handle := registry.AcquireHandle(5)
	defer handle.Release()

	// Verify it's initially valid
	assert.True(t, handle.IsValid())
	assert.NoError(t, handle.Validate())

	// Invalidate handles for blocks >= 5
	registry.InvalidateHandlesAfter(5)

	// Verify it's now invalid
	assert.False(t, handle.IsValid())
	assert.Equal(t, ErrInvalidHandle, handle.Validate())
}

func TestReadHandleUpdateBlock(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Create a handle for block 10
	handle := registry.AcquireHandle(10)
	defer handle.Release()

	// Update block number to 20
	ok := handle.UpdateBlock(20)
	assert.True(t, ok)

	// Invalidate blocks 15 and above
	registry.InvalidateHandlesAfter(15)

	// Handle should now be invalid due to the new block number
	assert.False(t, handle.IsValid())
	assert.Equal(t, ErrInvalidHandle, handle.Validate())

	// Attempting to update an invalid handle should fail
	ok = handle.UpdateBlock(5)
	assert.False(t, ok)
}

func TestReadHandleMultiple(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Create multiple handles at different block numbers
	handles := make([]*ReadHandle, 5)
	for i := range handles {
		handles[i] = registry.AcquireHandle(uint64(i + 1))
	}

	// Verify all handles are initially valid
	for _, h := range handles {
		require.NoError(t, h.Validate())
	}

	// Invalidate handles at block 3
	registry.InvalidateHandlesAfter(3)

	// Verify handles for blocks 1-2 are still valid
	for i := 0; i < 2; i++ {
		assert.NoError(t, handles[i].Validate())
	}

	// Verify handles for blocks 3-5 are invalid
	for i := 2; i < 5; i++ {
		assert.Equal(t, ErrInvalidHandle, handles[i].Validate())
	}

	for _, h := range handles {
		h.Release()
	}
}

func TestReadHandleRegistryLifecycle(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Acquire and release 100 handles
	for i := 0; i < 100; i++ {
		h := registry.AcquireHandle(uint64(i))
		assert.True(t, h.IsValid())
		h.Release()
	}

	// Verify active handles map is empty after all releases
	count := 0
	registry.activeHandles.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 0, count, "activeHandles map should be empty after all handles are released")

	// Verify next handle ID was incremented
	nextID := registry.nextHandleID.Load()
	assert.Equal(t, uint64(100), nextID)

	// Check that invalidating an empty registry doesn't panic
	assert.NotPanics(t, func() {
		registry.InvalidateHandlesAfter(0)
	})
}

func TestReadHandleHighVolume(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)
	const numHandles = 10000

	// Create many handles
	handles := make([]*ReadHandle, numHandles)
	for i := range handles {
		blockNum := uint64(i % 100) // Create handles for 100 different blocks
		handles[i] = registry.AcquireHandle(blockNum)
	}

	// Verify all handles are valid initially
	for _, h := range handles {
		assert.True(t, h.IsValid(), "All handles should be valid initially")
	}

	// Test invalidation performance
	registry.InvalidateHandlesAfter(50)

	// Verify correct invalidation
	validCount := 0
	invalidCount := 0
	for _, h := range handles {
		if h.IsValid() {
			validCount++
			assert.Less(t, h.blockNum, uint64(50), "Only blocks < 50 should be valid")
		} else {
			invalidCount++
			assert.GreaterOrEqual(t, h.blockNum, uint64(50), "Blocks >= 50 should be invalid")
		}
		h.Release()
	}

	// Since we created handles for blocks 0-99 with equal distribution,
	// and invalidated blocks 50-99, we expect approximately half to be valid and half invalid
	assert.Equal(t, 5000, validCount, "Expected half of handles to be valid")
	assert.Equal(t, 5000, invalidCount, "Expected half of handles to be invalid")

	// Check that all handles were properly released
	count := 0
	registry.activeHandles.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 0, count, "activeHandles map should be empty after all handles are released")
}

func TestReadHandleInvalidationBoundaries(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Create handles at precise boundaries
	handle0 := registry.AcquireHandle(0)
	handle1 := registry.AcquireHandle(1)
	handle2 := registry.AcquireHandle(2)
	handle5 := registry.AcquireHandle(5)
	handle10 := registry.AcquireHandle(10)

	// Invalidate at boundary 1
	registry.InvalidateHandlesAfter(1)

	// Check which handles are valid and which are invalid
	assert.True(t, handle0.IsValid(), "Handle for block 0 should be valid")
	assert.False(t, handle1.IsValid(), "Handle for block 1 should be invalid")
	assert.False(t, handle2.IsValid(), "Handle for block 2 should be invalid")
	assert.False(t, handle5.IsValid(), "Handle for block 5 should be invalid")
	assert.False(t, handle10.IsValid(), "Handle for block 10 should be invalid")

	// Test with InvalidateHandlesAfter(0) which should invalidate all handles
	registry = NewReadRegistry(logger)

	handle0 = registry.AcquireHandle(0)
	handle1 = registry.AcquireHandle(1)

	registry.InvalidateHandlesAfter(0)

	assert.False(t, handle0.IsValid(), "Handle for block 0 should be invalid after InvalidateHandlesAfter(0)")
	assert.False(t, handle1.IsValid(), "Handle for block 1 should be invalid after InvalidateHandlesAfter(0)")

	// Clean up
	handle0.Release()
	handle1.Release()
	handle2.Release()
	handle5.Release()
	handle10.Release()
}

func TestReadHandleConcurrentOperations(t *testing.T) {
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)
	const numHandles = 1000
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for g := 0; g < numGoroutines; g++ {
		go func(goroutineID int) {
			defer wg.Done()

			// Each goroutine acquires and releases numHandles/numGoroutines handles
			handles := make([]*ReadHandle, 0, numHandles/numGoroutines)
			for i := 0; i < numHandles/numGoroutines; i++ {
				h := registry.AcquireHandle(uint64(goroutineID*1000 + i))
				assert.True(t, h.IsValid())
				handles = append(handles, h)
			}

			// Release all handles
			for _, h := range handles {
				h.Release()
			}
		}(g)
	}

	wg.Wait()

	// Verify all handles were released
	count := 0
	registry.activeHandles.Range(func(_, _ interface{}) bool {
		count++
		return true
	})
	assert.Equal(t, 0, count, "activeHandles map should be empty after all handles are released")

	// Check the next handle ID is as expected
	assert.Equal(t, uint64(numHandles), registry.nextHandleID.Load())
}

func TestReadHandleSafeContains(t *testing.T) {
	db := setupTestDB(t)
	chainID := eth.ChainID{1}

	// Create a test access
	access := types.Access{
		BlockNumber: 100,
		Timestamp:   1000,
		LogIndex:    1,
		ChainID:     chainID,
		Checksum:    types.MessageChecksum(common.Hash{1}),
	}

	// Test successful case
	seal, err := db.SafeContains(chainID, access)
	assert.NoError(t, err)
	assert.Equal(t, uint64(100), seal.Number)

	// Get the existing mockLogDB and add block 25
	var mockDB1 *mockLogDB
	logDB, ok := db.logDBs.Get(chainID)
	if ok {
		mockDB1, ok = logDB.(*mockLogDB)
		if ok {
			require.NoError(t, mockDB1.AddLog(common.Hash{1}, eth.BlockID{Number: 25}, 1, &types.ExecutingMessage{
				BlockNum:  25,
				LogIdx:    1,
				Timestamp: 500,
			}))
			require.NoError(t, mockDB1.SealBlock(common.Hash{}, eth.BlockID{Number: 25, Hash: common.Hash{5}}, 500))
		}
	}

	// Test reorg by directly using Rewind which properly invalidates handles
	err = db.Rewind(chainID, eth.BlockID{Number: 50})
	assert.NoError(t, err)

	// The next call should fail
	_, err = db.SafeContains(chainID, access)
	assert.Error(t, err)

	// After reorg, if we try with a block number less than the reorg point, it should work
	// Since we added block 25 to our mock DB, this should succeed
	access.BlockNumber = 25
	access.Timestamp = 500
	seal, err = db.SafeContains(chainID, access)
	assert.NoError(t, err)
	assert.Equal(t, uint64(25), seal.Number)
}

func TestReadHandleTransactionRetryLogic(t *testing.T) {
	db := setupTestDB(t)
	chainID := eth.ChainID{1}

	// Set up a test counter to track retry attempts
	attemptCount := 0
	maxRetries := 3

	// First test: success on first try
	attemptCount = 0
	result := ""
	err := db.WithRetry(chainID, 100, maxRetries, func(handle *ReadHandle) error {
		attemptCount++
		result = "success on first try"
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attemptCount, "Should have succeeded on first attempt")
	assert.Equal(t, "success on first try", result)

	// Second test: fail on first try due to reorg, succeed on second
	attemptCount = 0
	result = ""

	err = db.WithRetry(chainID, 100, maxRetries, func(handle *ReadHandle) error {
		attemptCount++

		if attemptCount == 1 {
			// Simulate reorg during first attempt
			registry, _ := db.readRegistries.Get(chainID)
			registry.InvalidateHandlesAfter(50)
			return ErrInvalidHandle
		}

		// On second attempt, succeed
		result = "success on retry"
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, 2, attemptCount, "Should have succeeded on second attempt")
	assert.Equal(t, "success on retry", result)

	// Third test: fail all attempts
	attemptCount = 0
	err = db.WithRetry(chainID, 100, maxRetries, func(handle *ReadHandle) error {
		attemptCount++
		// Simulate reorg on every attempt
		registry, _ := db.readRegistries.Get(chainID)
		registry.InvalidateHandlesAfter(50)
		return ErrInvalidHandle
	})

	assert.Error(t, err)
	assert.Equal(t, ErrInvalidHandle, err, "Should return ErrInvalidHandle after all retries fail")
	assert.Equal(t, maxRetries, attemptCount, "Should have attempted exactly maxRetries times")

	// Fourth test: non-reorg error should not be retried
	attemptCount = 0
	customErr := fmt.Errorf("custom error")
	err = db.WithRetry(chainID, 100, maxRetries, func(handle *ReadHandle) error {
		attemptCount++
		return customErr
	})

	assert.Error(t, err)
	assert.Equal(t, customErr, err, "Should return the custom error without retrying")
	assert.Equal(t, 1, attemptCount, "Should have attempted exactly once")
}

func TestReadHandleUpdateRewindConcurrency(t *testing.T) {
	// For this test, let's use a direct ReadRegistry to avoid mock complexity
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)

	// Create a barrier to ensure all goroutines are blocked at the same point
	barrier := make(chan struct{}, 5)
	proceed := make(chan struct{})

	// Start multiple goroutines to read with handles
	const numGoroutines = 5
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			// Acquire a handle directly from the registry
			handle := registry.AcquireHandle(100)
			defer handle.Release()

			// Verify it's initially valid
			if !handle.IsValid() {
				errors <- fmt.Errorf("handle should be valid initially")
				return
			}

			// Signal we've reached the barrier
			barrier <- struct{}{}

			// Wait for all goroutines to be signaled to proceed
			<-proceed

			// Try to validate the handle after rewind
			err := handle.Validate()
			errors <- err
		}(i)
	}

	// Wait for all goroutines to reach the barrier
	for i := 0; i < numGoroutines; i++ {
		<-barrier
	}

	// Now perform a rewind while goroutines are all paused
	registry.InvalidateHandlesAfter(50)

	// Stop all goroutines to complete and check if we got the expected errors
	close(proceed)
	wg.Wait()
	close(errors)
	errCount := 0
	for err := range errors {
		if err == ErrInvalidHandle {
			errCount++
		} else if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	}
	assert.Equal(t, numGoroutines, errCount, "All goroutines should have received ErrInvalidHandle")
}

func TestReadHandleCrossChainConsistency(t *testing.T) {
	db := setupTestDB(t)
	chain1 := eth.ChainID{1}
	chain2 := eth.ChainID{2}

	// Create access list that for cross-chain dependencies
	accessList := []types.Access{
		{
			BlockNumber: 100,
			Timestamp:   1000,
			LogIndex:    1,
			ChainID:     chain1,
			Checksum:    types.MessageChecksum(common.Hash{1}),
		},
		{
			BlockNumber: 200,
			Timestamp:   2000,
			LogIndex:    2,
			ChainID:     chain2,
			Checksum:    types.MessageChecksum(common.Hash{2}),
		},
	}

	// Test successful validation before reorg
	err := db.ValidateAccessList(accessList)
	assert.NoError(t, err, "Access list should be valid before reorg")

	// Simulate a reorg on chain1, which should invalidate the access list
	// since the second access (on chain2) might depend on data from chain1
	err = db.Rewind(chain1, eth.BlockID{Number: 50})
	assert.NoError(t, err, "Rewind should succeed")

	// Validation should now fail because chain1 was rewound
	err = db.ValidateAccessList(accessList)
	assert.Error(t, err, "Access list validation should fail after reorg on chain1")

	// Set up a new scenario with independent chains
	logDB, found := db.logDBs.Get(chain1)
	require.True(t, found, "Failed to get logDB for chain1")
	mockDB1, ok := logDB.(*mockLogDB)
	require.True(t, ok, "Failed to cast logDB to mockLogDB")
	require.NoError(t, mockDB1.AddLog(common.Hash{3}, eth.BlockID{Number: 75}, 3, &types.ExecutingMessage{
		BlockNum:  75,
		LogIdx:    3,
		Timestamp: 1500,
	}))
	require.NoError(t, mockDB1.SealBlock(common.Hash{}, eth.BlockID{Number: 75, Hash: common.Hash{3}}, 1500))

	// Create new access list with the valid block on chain1
	newAccessList := []types.Access{
		{
			BlockNumber: 75,
			Timestamp:   1500,
			LogIndex:    3,
			ChainID:     chain1,
			Checksum:    types.MessageChecksum(common.Hash{3}),
		},
		{
			BlockNumber: 200,
			Timestamp:   2000,
			LogIndex:    2,
			ChainID:     chain2,
			Checksum:    types.MessageChecksum(common.Hash{2}),
		},
	}

	// This should now succeed because we're using a valid block on chain1
	err = db.ValidateAccessList(newAccessList)
	assert.NoError(t, err, "Access list with valid blocks should succeed after reorg")

	// Verify concurrent access across chains
	err = db.WithReadHandles(
		[]eth.ChainID{chain1, chain2},
		[]uint64{75, 200},
		func(handles []*ReadHandle) error {
			assert.Len(t, handles, 2, "Should have two handles")
			assert.True(t, handles[0].IsValid(), "Handle for chain1 should be valid")
			assert.True(t, handles[1].IsValid(), "Handle for chain2 should be valid")
			return nil
		})
	assert.NoError(t, err, "WithReadHandles should succeed with valid blocks")

	// Create a new registry
	logger := testlog.Logger(t, log.LvlTrace)
	registry := NewReadRegistry(logger)
	handle1 := registry.AcquireHandle(50)
	handle2 := registry.AcquireHandle(75)

	// Verify both handles are initially valid
	assert.True(t, handle1.IsValid(), "Handle1 should be valid initially")
	assert.True(t, handle2.IsValid(), "Handle2 should be valid initially")

	// Invalidate handles at block 70 and assert validity
	registry.InvalidateHandlesAfter(70)
	assert.True(t, handle1.IsValid(), "Handle1 should remain valid after InvalidateHandlesAfter(70)")
	assert.False(t, handle2.IsValid(), "Handle2 should be invalid after InvalidateHandlesAfter(70)")

	handle1.Release()
	handle2.Release()
}

func setupTestDB(t *testing.T) *ChainsDB {
	logger := testlog.Logger(t, log.LvlTrace)
	db := NewChainsDB(logger, nil, nil)

	// Add mock LogDBs for testing
	chain1 := eth.ChainID{1}
	chain2 := eth.ChainID{2}

	mockDB1 := newMockLogDB()
	mockDB2 := newMockLogDB()

	// Add some test data
	require.NoError(t, mockDB1.AddLog(common.Hash{1}, eth.BlockID{Number: 100}, 1, &types.ExecutingMessage{
		BlockNum:  100,
		LogIdx:    1,
		Timestamp: 1000,
	}))
	require.NoError(t, mockDB1.SealBlock(common.Hash{}, eth.BlockID{Number: 100, Hash: common.Hash{1}}, 1000))

	require.NoError(t, mockDB2.AddLog(common.Hash{2}, eth.BlockID{Number: 200}, 2, &types.ExecutingMessage{
		BlockNum:  200,
		LogIdx:    2,
		Timestamp: 2000,
	}))
	require.NoError(t, mockDB2.SealBlock(common.Hash{}, eth.BlockID{Number: 200, Hash: common.Hash{2}}, 2000))

	db.AddLogDB(chain1, mockDB1)
	db.AddLogDB(chain2, mockDB2)

	// Add mock local and cross DBs for Rewind to work
	mockLocalDB1 := &mockDerivationDB{}
	mockLocalDB2 := &mockDerivationDB{}
	mockCrossDB1 := &mockDerivationDB{}
	mockCrossDB2 := &mockDerivationDB{}

	db.AddLocalDerivationDB(chain1, mockLocalDB1)
	db.AddLocalDerivationDB(chain2, mockLocalDB2)
	db.AddCrossDerivationDB(chain1, mockCrossDB1)
	db.AddCrossDerivationDB(chain2, mockCrossDB2)

	return db
}

type mockLogDB struct {
	mu         sync.RWMutex
	logs       map[uint64]map[uint32]types.ExecutingMessage
	blockSeals map[uint64]types.BlockSeal
	rewindNum  uint64
}

func newMockLogDB() *mockLogDB {
	return &mockLogDB{
		logs:       make(map[uint64]map[uint32]types.ExecutingMessage),
		blockSeals: make(map[uint64]types.BlockSeal),
	}
}

func (m *mockLogDB) Close() error {
	return nil
}

func (m *mockLogDB) AddLog(logHash common.Hash, parentBlock eth.BlockID, logIdx uint32, execMsg *types.ExecutingMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.logs[parentBlock.Number] == nil {
		m.logs[parentBlock.Number] = make(map[uint32]types.ExecutingMessage)
	}
	m.logs[parentBlock.Number][logIdx] = *execMsg
	return nil
}

func (m *mockLogDB) SealBlock(parentHash common.Hash, block eth.BlockID, timestamp uint64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.blockSeals[block.Number] = types.BlockSeal{
		Hash:      block.Hash,
		Number:    block.Number,
		Timestamp: timestamp,
	}
	return nil
}

func (m *mockLogDB) Rewind(newHead eth.BlockID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for num := range m.logs {
		if num > newHead.Number {
			delete(m.logs, num)
			delete(m.blockSeals, num)
		}
	}
	return nil
}

func (m *mockLogDB) LatestSealedBlock() (id eth.BlockID, ok bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var latest uint64
	for num := range m.blockSeals {
		if num > latest {
			latest = num
		}
	}
	if latest == 0 {
		return eth.BlockID{}, false
	}
	seal := m.blockSeals[latest]
	return eth.BlockID{Hash: seal.Hash, Number: seal.Number}, true
}

func (m *mockLogDB) FindSealedBlock(number uint64) (types.BlockSeal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if seal, ok := m.blockSeals[number]; ok {
		return seal, nil
	}
	return types.BlockSeal{}, types.ErrFuture
}

func (m *mockLogDB) Contains(query types.ContainsQuery) (types.BlockSeal, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if query.BlockNum > m.rewindNum && m.rewindNum != 0 {
		return types.BlockSeal{}, types.ErrFuture
	}

	if blockLogs, ok := m.logs[query.BlockNum]; ok {
		if _, ok := blockLogs[query.LogIdx]; ok {
			if seal, ok := m.blockSeals[query.BlockNum]; ok {
				return seal, nil
			}
		}
	}
	return types.BlockSeal{}, types.ErrFuture
}

func (m *mockLogDB) IteratorStartingAt(sealedNum uint64, logsSince uint32) (logs.Iterator, error) {
	return nil, nil
}

func (m *mockLogDB) OpenBlock(blockNum uint64) (ref eth.BlockRef, logCount uint32, execMsgs map[uint32]*types.ExecutingMessage, err error) {
	return eth.BlockRef{}, 0, nil, nil
}

func (m *mockLogDB) addBlock(num uint64, hash common.Hash) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockSeals[num] = types.BlockSeal{
		Number: num,
		Hash:   hash,
	}
}

func (m *mockLogDB) rewind(num uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rewindNum = num
}

type mockDerivationDB struct{}

func (m *mockDerivationDB) First() (pair types.DerivedBlockSealPair, err error) {
	return types.DerivedBlockSealPair{}, nil
}

func (m *mockDerivationDB) Last() (pair types.DerivedBlockSealPair, err error) {
	return types.DerivedBlockSealPair{}, nil
}

func (m *mockDerivationDB) DerivedToFirstSource(derived eth.BlockID, revision types.Revision) (source types.BlockSeal, err error) {
	return types.BlockSeal{}, nil
}

func (m *mockDerivationDB) SourceToLastDerived(source eth.BlockID) (derived types.BlockSeal, err error) {
	return types.BlockSeal{}, nil
}

func (m *mockDerivationDB) NextSource(source eth.BlockID) (nextSource types.BlockSeal, err error) {
	return types.BlockSeal{}, nil
}

func (m *mockDerivationDB) Candidate(afterSource eth.BlockID, afterDerived eth.BlockID, revision types.Revision) (pair types.DerivedBlockRefPair, err error) {
	return types.DerivedBlockRefPair{}, nil
}

func (m *mockDerivationDB) PreviousSource(source eth.BlockID) (prevSource types.BlockSeal, err error) {
	return types.BlockSeal{}, nil
}

func (m *mockDerivationDB) PreviousDerived(derived eth.BlockID, revision types.Revision) (prevDerived types.BlockSeal, err error) {
	return types.BlockSeal{}, nil
}

func (m *mockDerivationDB) Invalidated() (pair types.DerivedBlockSealPair, err error) {
	return types.DerivedBlockSealPair{}, nil
}

func (m *mockDerivationDB) ContainsDerived(derived eth.BlockID, revision types.Revision) error {
	return nil
}

func (m *mockDerivationDB) DerivedToRevision(derived eth.BlockID) (types.Revision, error) {
	return types.RevisionAny, nil
}

func (m *mockDerivationDB) LastRevision() (revision types.Revision, err error) {
	return types.RevisionAny, nil
}

func (m *mockDerivationDB) SourceToRevision(source eth.BlockID) (types.Revision, error) {
	return types.RevisionAny, nil
}

func (m *mockDerivationDB) AddDerived(source eth.BlockRef, derived eth.BlockRef, revision types.Revision) error {
	return nil
}

func (m *mockDerivationDB) ReplaceInvalidatedBlock(replacementDerived eth.BlockRef, invalidated common.Hash) (types.DerivedBlockRefPair, error) {
	return types.DerivedBlockRefPair{}, nil
}

func (m *mockDerivationDB) RewindAndInvalidate(invalidated types.DerivedBlockRefPair) error {
	return nil
}

func (m *mockDerivationDB) RewindToScope(scope eth.BlockID) error {
	return nil
}

func (m *mockDerivationDB) RewindToFirstDerived(v eth.BlockID, revision types.Revision) error {
	return nil
}
