package interop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSupernodeInterop_SafeHeadTrailsLocalSafe tests that the cross-safe head
// (SafeL2) trails behind the local safe head (LocalSafeL2) and eventually catches up
// after interop verification completes.
//
// This test verifies:
// - SafeL2 <= LocalSafeL2 at all times
// - SafeL2 advances after verification
// - SafeL2 eventually catches up to LocalSafeL2
func TestSupernodeInterop_SafeHeadTrailsLocalSafe(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	waitTime := 500 * time.Millisecond

	// Wait for initial sync
	waitForInitialSync(t, sys, 5, 30*time.Second)

	// Track progression over time
	var previousSafeA, previousSafeB uint64
	progressCount := 0

	for i := 0; i < 20; i++ {
		time.Sleep(waitTime)

		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()

		t.Logger().Info("safe head progression", "iteration", i)
		logSyncState(t, "current state", statusA, statusB)

		// KEY ASSERTION 1: Safe head must never exceed local safe head
		requireSafeNotAboveLocalSafe(t, statusA, "chain A")
		requireSafeNotAboveLocalSafe(t, statusB, "chain B")

		// KEY ASSERTION 2: SafeL2 fields should be populated
		requireSafeL2FieldsPopulated(t, statusA, "chain A")
		requireSafeL2FieldsPopulated(t, statusB, "chain B")

		// Track if safe heads are progressing
		if statusA.SafeL2.Number > previousSafeA || statusB.SafeL2.Number > previousSafeB {
			progressCount++
			previousSafeA = statusA.SafeL2.Number
			previousSafeB = statusB.SafeL2.Number
		}
	}

	// KEY ASSERTION 3: Safe heads should have progressed multiple times
	assert.Greater(t, progressCount, 3, "safe heads should progress multiple times during test")

	// Final check: safe heads should eventually catch up to a snapshot of local safe heads
	snapshotStatusA := sys.L2ACL.SyncStatus()
	snapshotStatusB := sys.L2BCL.SyncStatus()
	snapshotLocalSafeA := snapshotStatusA.LocalSafeL2.Number
	snapshotLocalSafeB := snapshotStatusB.LocalSafeL2.Number

	logDetailedState(t, "snapshot state for catchup", snapshotStatusA, snapshotStatusB)

	// Wait for safe heads to catch up to the snapshot of local safe heads
	t.Require().Eventually(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.SafeL2.Number >= snapshotLocalSafeA &&
			statusB.SafeL2.Number >= snapshotLocalSafeB
	}, 60*time.Second, waitTime, "safe heads should eventually catch up to snapshot of local safe heads")
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
