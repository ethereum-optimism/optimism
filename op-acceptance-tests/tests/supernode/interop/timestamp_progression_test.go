package interop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestSupernodeInteropVerifiedAt tests that the VerifiedAt endpoint returns
// correct data after the interop activity has processed timestamps.
func TestSupernodeInteropVerifiedAt(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	genesisTime := sys.L2A.Escape().RollupConfig().Genesis.L2Time
	ctx := t.Ctx()
	snClient := sys.SuperNodeClient()

	// Query for a timestamp that should be verified
	// Use genesis time + one block time to ensure we're past the first block
	targetTimestamp := genesisTime + blockTime

	t.Logger().Info("querying verified at timestamp",
		"genesis_time", genesisTime,
		"target_timestamp", targetTimestamp,
	)

	// Wait for the interop activity to verify the target timestamp
	t.Require().Eventually(func() bool {
		resp, err := snClient.SuperRootAtTimestamp(ctx, targetTimestamp)
		if err != nil {
			return false
		}
		return resp.Data != nil
	}, 60*time.Second, time.Second, "interop should verify target timestamp")

	// Log the final state
	resp, err := snClient.SuperRootAtTimestamp(ctx, targetTimestamp)
	t.Require().NoError(err)
	t.Logger().Info("verified at test complete",
		"target_timestamp", targetTimestamp,
		"super_root", resp.Data.SuperRoot,
	)
}

// TestSupernodeInteropChainLag tests that interop verification is based on SAFE heads,
// not unsafe heads. When chain B's batcher is stopped (but CL keeps running):
// - Chain B's unsafe head continues to advance
// - Chain B's safe head is frozen (no batches submitted to L1)
// - Timestamps should NOT be verified until safe heads catch up
//
// This proves the supernode waits for all chains' safe heads before verifying.
func TestSupernodeInteropChainLag(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	ctx := t.Ctx()
	snClient := sys.SuperNodeClient()

	// Let both chains advance initially and wait for verification to catch up
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.SafeL2.Number > 2 && statusB.SafeL2.Number > 2
	}, 60*time.Second, time.Second, "both chains should advance initially")

	// Record the current state - this is the "baseline" verified timestamp
	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()
	baselineTimestamp := statusA.SafeL2.Time

	// Wait for baseline timestamp to be verified before proceeding
	t.Require().Eventually(func() bool {
		resp, err := snClient.SuperRootAtTimestamp(ctx, baselineTimestamp)
		if err != nil {
			return false
		}
		return resp.Data != nil
	}, 60*time.Second, time.Second, "baseline timestamp should be verified before stopping batcher")

	t.Logger().Info("initial state before lag test",
		"chainA_safe", statusA.SafeL2.Number,
		"chainA_safe_time", statusA.SafeL2.Time,
		"chainB_safe", statusB.SafeL2.Number,
		"chainB_safe_time", statusB.SafeL2.Time,
		"chainB_unsafe", statusB.UnsafeL2.Number,
		"baseline_timestamp", baselineTimestamp,
	)

	// Stop Chain B's batcher to halt safe head progression
	sys.L2BatcherB.Stop()
	t.Logger().Info("stopped chain B batcher (CL still running)")

	// Wait for in-flight batches to settle - the safe head should stabilize
	// We wait until the safe head doesn't change for 10 seconds
	// or up to 30 seconds to fail the test
	var bStoppedSafeNum uint64
	var bStoppedSafeTime uint64
	lastSafe := sys.L2BCL.SyncStatus().SafeL2.Number
	stableFor := 0
	start := time.Now()
	for stableFor < 10 {
		time.Sleep(time.Second)
		current := sys.L2BCL.SyncStatus()
		if current.SafeL2.Number == lastSafe {
			stableFor++
		} else {
			stableFor = 0
			lastSafe = current.SafeL2.Number
		}
		bStoppedSafeNum = current.SafeL2.Number
		bStoppedSafeTime = current.SafeL2.Time
		if time.Since(start) > 30*time.Second {
			t.Logger().Error("safe head did not stabilize after 30 seconds")
			t.FailNow()
		}
	}

	bStoppedStatus := sys.L2BCL.SyncStatus()
	t.Logger().Info("chain B batcher stopped (safe head stabilized)",
		"chainB_safe", bStoppedSafeNum,
		"chainB_safe_time", bStoppedSafeTime,
		"chainB_unsafe", bStoppedStatus.UnsafeL2.Number,
	)

	// Watch safe/unsafe and verified for 30 seconds
	// First wait for chain A to advance past chain B's frozen safe head
	start = time.Now()
	var aheadTimestamp uint64
	for {
		if time.Since(start) > 30*time.Second {
			break
		}

		time.Sleep(time.Second)

		// Check the state
		newStatusA := sys.L2ACL.SyncStatus()
		newStatusB := sys.L2BCL.SyncStatus()

		t.Logger().Info("state",
			"chainA_safe", newStatusA.SafeL2.Number,
			"chainA_safe_time", newStatusA.SafeL2.Time,
			"chainB_safe", newStatusB.SafeL2.Number,
			"chainB_safe_time", newStatusB.SafeL2.Time,
			"chainB_unsafe", newStatusB.UnsafeL2.Number,
		)

		// KEY ASSERTION 1: Chain B's unsafe head should have advanced (CL is still running)
		t.Require().Greater(newStatusB.UnsafeL2.Number, bStoppedSafeNum,
			"chain B unsafe head should advance even with batcher stopped")

		// KEY ASSERTION 2: Chain B's safe head should be frozen (no batches)
		t.Require().Equal(bStoppedSafeNum, newStatusB.SafeL2.Number,
			"chain B safe head should be frozen with batcher stopped")

		// Use chain A's ahead timestamp for verification check
		aheadTimestamp = newStatusA.SafeL2.Time

		// KEY ASSERTION 3: The timestamp should NOT be verified
		// Even though chain B's unsafe head is past this timestamp,
		// verification requires SAFE heads on all chains
		resp, err := snClient.SuperRootAtTimestamp(ctx, aheadTimestamp)
		t.Require().NoError(err, "SuperRootAtTimestamp should not error")
		t.Require().Nil(resp.Data,
			"timestamp should NOT be verified - chain B unsafe is ahead but safe is behind")

		t.Logger().Info("confirmed: timestamp not verified despite chain B unsafe being ahead",
			"ahead_timestamp", aheadTimestamp,
			"chainB_unsafe", newStatusB.UnsafeL2.Number,
			"chainB_safe", newStatusB.SafeL2.Number,
		)
	}

	// Resume the batcher
	sys.L2BatcherB.Start()
	t.Logger().Info("resumed chain B batcher")

	// Wait for chain B's SAFE head to catch up
	timeout := time.Duration(blockTime*20+60) * time.Second
	t.Require().Eventually(func() bool {
		currentB := sys.L2BCL.SyncStatus()
		return currentB.SafeL2.Time >= aheadTimestamp
	}, timeout, time.Second, "chain B safe head should catch up after batcher resumes")

	// KEY ASSERTION 4: Now that safe heads are caught up, timestamp should be verified
	t.Require().Eventually(func() bool {
		resp, err := snClient.SuperRootAtTimestamp(ctx, aheadTimestamp)
		if err != nil {
			t.Logger().Warn("SuperRootAtTimestamp error while waiting for verification", "err", err)
			return false
		}
		return resp.Data != nil
	}, 60*time.Second, time.Second, "ahead timestamp should be verified after chain B safe catches up")

	t.Logger().Info("confirmed: ahead timestamp is now verified after chain B safe caught up",
		"ahead_timestamp", aheadTimestamp,
	)

	// Both chains should continue advancing together
	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("final state after recovery",
		"chainA_safe", finalStatusA.SafeL2.Number,
		"chainB_safe", finalStatusB.SafeL2.Number,
		"chainB_unsafe", finalStatusB.UnsafeL2.Number,
	)

	t.Require().Greater(finalStatusA.SafeL2.Number, statusA.SafeL2.Number, "chain A should have advanced")
	t.Require().Greater(finalStatusB.SafeL2.Number, statusB.SafeL2.Number, "chain B should have advanced")
}
