package interop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestSupernodeInterop_SafeHeadTrailsLocalSafe tests that the cross-safe head
// (SafeL2) trails behind the local safe head (LocalSafeL2) and eventually catches up
// after interop verification completes (assuming no node resets occur).
//
// This test verifies:
//   - SafeL2 <= LocalSafeL2 at all times (the exception to this might be during a node reset where the local safe has to catch back up,
//     but we don't trigger that here)
//   - SafeL2 advances after verification
//   - SafeL2 eventually catches up to LocalSafeL2 (assuming we don't insert any invalid message, which we don't)
//   - EL safe label is consistent with the SafeL2 from the CL
func TestSupernodeInterop_SafeHeadTrailsLocalSafe(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)
	attempts := 15 // each attempt is hardcoded with a 2s by the DSL.

	// Pause interop verification at block 5 on both chains
	// check safe heads get to at least that height,
	// let local safe heads run ahead
	pauseTimestamp := sys.L2A.TimestampForBlockNum(5)
	sys.Supernode.PauseInterop(pauseTimestamp)
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.LocalSafe, 10, attempts),
		sys.L2BCL.ReachedFn(types.LocalSafe, 10, attempts),
		sys.L2ACL.ReachedFn(types.CrossSafe, 4, attempts),
		sys.L2BCL.ReachedFn(types.CrossSafe, 4, attempts),
	)

	// Expect cross safe to stall since we paused the interop activity
	dsl.CheckAll(t,
		sys.L2ACL.NotAdvancedFn(types.CrossSafe, 2),
		sys.L2BCL.NotAdvancedFn(types.CrossSafe, 2),
	)

	// Check EL labels
	safeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	safeB := sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.LessOrEqual(t, safeA.Number, uint64(5))
	require.LessOrEqual(t, safeB.Number, uint64(5))

	// Resume interop verification
	// expect cross safe to catch up
	sys.Supernode.ResumeInterop()
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.CrossSafe, 10, attempts),
		sys.L2BCL.ReachedFn(types.CrossSafe, 10, attempts),
	)

	// check EL labels
	safeA = sys.L2ELA.BlockRefByLabel(eth.Safe)
	safeB = sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.GreaterOrEqual(t, safeA.Number, uint64(10))
	require.GreaterOrEqual(t, safeB.Number, uint64(10))
}

// TestSupernodeInterop_SafeHeadWithUnevenProgress tests safe head behavior
// when chains advance at different rates.
//
// This test verifies:
// - Local safe heads can diverge between chains
// - Cross-safe head is gated by the slower chain
// - Safe head advances after slower chain catches up
func TestSupernodeInterop_SafeHeadWithUnevenProgress(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	waitTime := 500 * time.Millisecond

	// Wait for initial sync
	waitForInitialSync(t, sys, 5, 30*time.Second)

	baselineStatusA := sys.L2ACL.SyncStatus()
	baselineStatusB := sys.L2BCL.SyncStatus()

	logSyncState(t, "baseline state", baselineStatusA, baselineStatusB)

	// Stop chain B's batcher temporarily to create uneven progress
	sys.L2BatcherB.Stop()
	t.Logger().Info("stopped chain B batcher to create uneven progress")

	// Let chain A advance while chain B's local safe head is frozen
	time.Sleep(10 * time.Second)

	unevenStatusA := sys.L2ACL.SyncStatus()
	unevenStatusB := sys.L2BCL.SyncStatus()

	logSyncState(t, "uneven progress state", unevenStatusA, unevenStatusB)

	// KEY ASSERTION 1: Chain A's local safe should have advanced
	assert.Greater(t, unevenStatusA.LocalSafeL2.Number, baselineStatusA.LocalSafeL2.Number,
		"chain A local safe should advance while chain B is stopped")

	// KEY ASSERTION 2: Chain B's local safe should be relatively stable (may advance slightly from in-flight batches)
	assert.LessOrEqual(t, unevenStatusB.LocalSafeL2.Number, baselineStatusB.LocalSafeL2.Number+5,
		"chain B local safe should not advance much with batcher stopped")

	// KEY ASSERTION 3: Cross-safe heads should be gated by slower chain
	// The safe head should not exceed chain B's local safe head
	assert.LessOrEqual(t, unevenStatusA.SafeL2.Number, unevenStatusB.LocalSafeL2.Number+2,
		"safe heads should be gated by slower chain's local safe")
	assert.LessOrEqual(t, unevenStatusB.SafeL2.Number, unevenStatusB.LocalSafeL2.Number+2,
		"safe heads should be gated by slower chain's local safe")

	// Resume chain B's batcher
	sys.L2BatcherB.Start()
	t.Logger().Info("resumed chain B batcher")

	// Wait for chain B to catch up
	t.Require().Eventually(func() bool {
		statusB := sys.L2BCL.SyncStatus()
		// Chain B should catch up to approximately where chain A was
		return statusB.LocalSafeL2.Number >= unevenStatusA.LocalSafeL2.Number-5
	}, 60*time.Second, waitTime, "chain B should catch up after batcher resumes")

	// Wait for safe heads to catch up
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		// Safe heads should advance significantly from the uneven state
		return statusA.SafeL2.Number > unevenStatusA.SafeL2.Number+5 &&
			statusB.SafeL2.Number > unevenStatusB.SafeL2.Number+5
	}, 60*time.Second, waitTime, "safe heads should advance after chain B catches up")

	// Take snapshot of local safe heads after recovery
	snapshotStatusA := sys.L2ACL.SyncStatus()
	snapshotStatusB := sys.L2BCL.SyncStatus()
	snapshotLocalSafeA := snapshotStatusA.LocalSafeL2.Number
	snapshotLocalSafeB := snapshotStatusB.LocalSafeL2.Number

	logSyncState(t, "snapshot state after recovery", snapshotStatusA, snapshotStatusB)

	// Wait for safe heads to catch up to the snapshot of local safe heads
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.SafeL2.Number >= snapshotLocalSafeA &&
			statusB.SafeL2.Number >= snapshotLocalSafeB
	}, 60*time.Second, waitTime, "safe heads should eventually catch up to snapshot of local safe heads after recovery")
}

// Helper functions for safe head progression tests

// requireSafeNotAboveLocalSafe asserts that SafeL2 never exceeds LocalSafeL2
func requireSafeNotAboveLocalSafe(t devtest.T, status *eth.SyncStatus, chainName string) {
	assert.LessOrEqual(t, status.SafeL2.Number, status.LocalSafeL2.Number,
		"%s: SafeL2 should never exceed LocalSafeL2", chainName)
}

// requireSafeL2FieldsPopulated asserts that SafeL2 fields are non-zero when block number is non-zero
func requireSafeL2FieldsPopulated(t devtest.T, status *eth.SyncStatus, chainName string) {
	if status.SafeL2.Number > 0 {
		assert.NotZero(t, status.SafeL2.Time, "%s: SafeL2.Time should be non-zero", chainName)
		assert.NotZero(t, status.SafeL2.Hash, "%s: SafeL2.Hash should be non-zero", chainName)
	}
}

// logSyncState logs the current sync status for both chains
func logSyncState(t devtest.T, label string, statusA, statusB *eth.SyncStatus) {
	t.Logger().Info(label,
		"chainA_local_safe", statusA.LocalSafeL2.Number,
		"chainA_safe", statusA.SafeL2.Number,
		"chainB_local_safe", statusB.LocalSafeL2.Number,
		"chainB_safe", statusB.SafeL2.Number,
	)
}

// logDetailedState logs detailed state including gaps
func logDetailedState(t devtest.T, label string, statusA, statusB *eth.SyncStatus) {
	gapA := statusA.LocalSafeL2.Number - statusA.SafeL2.Number
	gapB := statusB.LocalSafeL2.Number - statusB.SafeL2.Number
	t.Logger().Info(label,
		"chainA_local_safe", statusA.LocalSafeL2.Number,
		"chainA_safe", statusA.SafeL2.Number,
		"chainA_gap", gapA,
		"chainB_local_safe", statusB.LocalSafeL2.Number,
		"chainB_safe", statusB.SafeL2.Number,
		"chainB_gap", gapB,
	)
}

// waitForInitialSync waits for both chains to reach a minimum block number
func waitForInitialSync(t devtest.T, sys *presets.TwoL2SupernodeInterop, minBlocks uint64, timeout time.Duration) {
	waitTime := 500 * time.Millisecond
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.LocalSafeL2.Number > minBlocks && statusB.LocalSafeL2.Number > minBlocks
	}, timeout, waitTime, "chains should sync initially")
}
