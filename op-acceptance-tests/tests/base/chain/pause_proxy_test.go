package chain

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/stretchr/testify/require"
)

// TestBatcherPauseProxy tests the batcher pause functionality via RPC proxy.
// It verifies that:
// 1. The batcher can be paused at a specific block number via test control
// 2. The pause state can be queried via IsPaused()
// 3. The batcher can be unpaused
// This is a unit/integration test of the pause control API, not an end-to-end
// test of pause behavior (which would require complex timing and state verification)
func TestBatcherPauseProxy(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewMinimal(t)
	l := t.Logger()

	// Use the batcher directly from the system
	batcher := sys.L2Batcher

	// Test 1: Verify initially not paused
	l.Info("Test 1: Verifying batcher is initially not paused")
	isPaused, pauseBlock := batcher.IsPaused()
	require.False(t, isPaused, "batcher should not be paused initially")
	require.Equal(t, uint64(0), pauseBlock, "pause block should be 0 when not paused")
	sys.L2CL.Advanced(types.LocalSafe, 1, 10)
	l.Info("Verified batcher is not paused initially")

	// Test 2: Pause at a specific block
	testPauseBlock := uint64(10)
	l.Info("Test 2: Pausing batcher", "pauseBlock", testPauseBlock)
	result := batcher.PauseAtBlock(testPauseBlock)
	require.Equal(t, testPauseBlock, result, "PauseAtBlock should return the pause block")
	sys.L2CL.Reached(types.LocalSafe, testPauseBlock, 10)

	// Verify the pause was set
	isPaused, pauseBlock = batcher.IsPaused()
	require.True(t, isPaused, "batcher should be paused")
	require.Equal(t, testPauseBlock, pauseBlock, "pause block should match")
	sys.L2CL.NotAdvanced(types.LocalSafe, 3)
	l.Info("Verified batcher is paused", "pauseBlock", pauseBlock)

	// Test 3: Change pause block
	newPauseBlock := uint64(15)
	l.Info("Test 3: Changing pause block", "newPauseBlock", newPauseBlock)
	result = batcher.PauseAtBlock(newPauseBlock)
	require.Equal(t, newPauseBlock, result, "PauseAtBlock should return the new pause block")

	// Verify the pause was updated
	isPaused, pauseBlock = batcher.IsPaused()
	require.True(t, isPaused, "batcher should still be paused")
	require.Equal(t, newPauseBlock, pauseBlock, "pause block should be updated")
	sys.L2CL.Reached(types.LocalSafe, newPauseBlock, 10)
	l.Info("Verified pause block was updated", "pauseBlock", pauseBlock)

	// Test 4: Unpause
	l.Info("Test 4: Unpausing batcher")
	batcher.Unpause()

	// Verify the batcher is no longer paused
	isPaused, pauseBlock = batcher.IsPaused()
	require.False(t, isPaused, "batcher should not be paused after unpause")
	require.Equal(t, uint64(0), pauseBlock, "pause block should be 0 after unpause")
	sys.L2CL.Advanced(types.LocalSafe, 1, 10)
	l.Info("Verified batcher is not paused after unpause")
}
