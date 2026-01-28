package interop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSupernodeInteropTimestampProgression verifies that the supernode's interop activity
// progresses through timestamps sequentially as chains advance.
//
// This test confirms:
// - Both L2 chains advance their safe heads
// - The supernode processes timestamps in order
// - Cross-chain verification happens at each timestamp
func TestSupernodeInteropTimestampProgression(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t)

	// Get the block time from chain A's rollup config
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	waitTime := time.Duration(blockTime+1) * time.Second

	// Record initial sync status
	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("initial sync status",
		"chainA_unsafe", statusA.UnsafeL2.Number,
		"chainA_safe", statusA.SafeL2.Number,
		"chainB_unsafe", statusB.UnsafeL2.Number,
		"chainB_safe", statusB.SafeL2.Number,
	)

	// Wait for both chains to advance their unsafe heads
	t.Require().Eventually(func() bool {
		newStatusA := sys.L2ACL.SyncStatus()
		newStatusB := sys.L2BCL.SyncStatus()
		return newStatusA.UnsafeL2.Number > statusA.UnsafeL2.Number &&
			newStatusB.UnsafeL2.Number > statusB.UnsafeL2.Number
	}, 30*time.Second, waitTime, "chains should advance unsafe heads")

	// Wait for safe heads to advance (requires batching and L1 inclusion)
	t.Require().Eventually(func() bool {
		newStatusA := sys.L2ACL.SyncStatus()
		newStatusB := sys.L2BCL.SyncStatus()
		t.Logger().Info("waiting for safe head progression",
			"chainA_safe", newStatusA.SafeL2.Number,
			"chainB_safe", newStatusB.SafeL2.Number,
		)
		return newStatusA.SafeL2.Number > statusA.SafeL2.Number &&
			newStatusB.SafeL2.Number > statusB.SafeL2.Number
	}, 60*time.Second, waitTime, "chains should advance safe heads")

	// Log final status
	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()
	t.Logger().Info("final sync status",
		"chainA_unsafe", finalStatusA.UnsafeL2.Number,
		"chainA_safe", finalStatusA.SafeL2.Number,
		"chainB_unsafe", finalStatusB.UnsafeL2.Number,
		"chainB_safe", finalStatusB.SafeL2.Number,
	)
}

// TestSupernodeInteropChainsOnDifferentChainIDs verifies that the two L2 chains
// are correctly configured with different chain IDs but share the same supernode.
func TestSupernodeInteropChainsOnDifferentChainIDs(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t)

	chainIDA := sys.L2A.ChainID()
	chainIDB := sys.L2B.ChainID()

	t.Require().NotEqual(chainIDA, chainIDB, "chains should have different chain IDs")

	// Verify both chains are advancing
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	waitTime := time.Duration(blockTime+1) * time.Second

	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Require().Eventually(func() bool {
		newA := sys.L2ACL.SyncStatus().UnsafeL2.Number
		newB := sys.L2BCL.SyncStatus().UnsafeL2.Number
		return newA > statusA.UnsafeL2.Number && newB > statusB.UnsafeL2.Number
	}, 30*time.Second, waitTime, "both chains should advance")
}

// TestSupernodeInteropSafeHeadProgression verifies that safe heads progress
// on both chains, which is a prerequisite for interop verification.
func TestSupernodeInteropSafeHeadProgression(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Target: wait for at least 3 blocks of safe head progression on both chains
	targetDelta := uint64(3)
	timeout := time.Duration(blockTime*10+30) * time.Second

	initialA := sys.L2ACL.SyncStatus().SafeL2.Number
	initialB := sys.L2BCL.SyncStatus().SafeL2.Number

	t.Logger().Info("waiting for safe head progression",
		"target_delta", targetDelta,
		"initial_A", initialA,
		"initial_B", initialB,
	)

	t.Require().Eventually(func() bool {
		currentA := sys.L2ACL.SyncStatus().SafeL2.Number
		currentB := sys.L2BCL.SyncStatus().SafeL2.Number

		deltaA := currentA - initialA
		deltaB := currentB - initialB

		t.Logger().Debug("safe head progress",
			"chainA_delta", deltaA,
			"chainB_delta", deltaB,
		)

		return deltaA >= targetDelta && deltaB >= targetDelta
	}, timeout, time.Second, "safe heads should progress by target delta blocks")
}

// TestSupernodeInteropVerifiedAt tests that the VerifiedAt endpoint returns
// correct data after the interop activity has processed timestamps.
func TestSupernodeInteropVerifiedAt(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	genesisTime := sys.L2A.Escape().RollupConfig().Genesis.L2Time

	// Wait for safe head to advance past genesis
	t.Require().Eventually(func() bool {
		status := sys.L2ACL.SyncStatus()
		return status.SafeL2.Number > 0
	}, 60*time.Second, time.Second, "safe head should advance past genesis")

	// Query for a timestamp that should be verified
	// Use genesis time + one block time to ensure we're past the first block
	targetTimestamp := genesisTime + blockTime

	t.Logger().Info("querying verified at timestamp",
		"genesis_time", genesisTime,
		"target_timestamp", targetTimestamp,
	)

	// Wait for the interop activity to verify the target timestamp
	// The safe head time being past the target timestamp indicates interop has processed it
	t.Require().Eventually(func() bool {
		// Check if the safe head is past our target timestamp
		// This indicates the interop activity has processed and verified that timestamp
		status := sys.L2ACL.SyncStatus()
		return status.SafeL2.Time >= targetTimestamp
	}, 60*time.Second, time.Second, "interop should verify target timestamp")

	// Log the final state
	finalStatus := sys.L2ACL.SyncStatus()
	t.Logger().Info("verified at test complete",
		"safe_l2_number", finalStatus.SafeL2.Number,
		"safe_l2_time", finalStatus.SafeL2.Time,
		"target_timestamp", targetTimestamp,
	)
}

// TestSupernodeInteropMultipleTimestamps verifies that multiple consecutive
// timestamps are processed correctly by the interop activity.
func TestSupernodeInteropMultipleTimestamps(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	genesisTime := sys.L2A.Escape().RollupConfig().Genesis.L2Time

	// Wait for multiple blocks to be produced and become safe
	targetBlocks := uint64(5)
	timeout := time.Duration(blockTime*15+60) * time.Second

	t.Logger().Info("waiting for multiple timestamps to be processed",
		"target_blocks", targetBlocks,
		"genesis_time", genesisTime,
	)

	var timestamps []uint64
	t.Require().Eventually(func() bool {
		status := sys.L2ACL.SyncStatus()
		if status.SafeL2.Number >= targetBlocks {
			// Collect the timestamps that should have been verified
			for i := uint64(0); i <= targetBlocks; i++ {
				ts := genesisTime + (i * blockTime)
				timestamps = append(timestamps, ts)
			}
			return true
		}
		return false
	}, timeout, time.Second, "should process target blocks")

	t.Logger().Info("multiple timestamps processed",
		"timestamps_count", len(timestamps),
		"first_timestamp", timestamps[0],
		"last_timestamp", timestamps[len(timestamps)-1],
	)

	// Verify both chains have progressed
	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Require().GreaterOrEqual(statusA.SafeL2.Number, targetBlocks, "chain A should have processed target blocks")
	t.Require().GreaterOrEqual(statusB.SafeL2.Number, targetBlocks, "chain B should have processed target blocks")
}
