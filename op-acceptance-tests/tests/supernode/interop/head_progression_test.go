package interop

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestSupernodeInterop_SafeHeadProgression tests that the cross-safe head
// (SafeL2) trails behind the local safe head (LocalSafeL2) and eventually catches up
// after interop verification completes (assuming no node resets occur).
//
// This test verifies:
//   - SafeL2 <= LocalSafeL2 at all times (the exception to this might be during a node reset where the local safe has to catch back up,
//     but we don't trigger that here)
//   - SafeL2 advances after verification
//   - SafeL2 eventually catches up to LocalSafeL2 (assuming we don't insert any invalid message, which we don't)
//   - EL safe label is consistent with the SafeL2 from the CL
//   - Finalized head eventually catches up to a snapshot of the safe head
//   - Finalized L2 blocks have sane L1 origins (behind the L1 finalized head)
func TestSupernodeInterop_SafeHeadProgression(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)
	attempts := 15 // each attempt is hardcoded with a 2s by the DSL.

	finalTargetBlockNum := uint64(10)

	// Pause interop and verify it has stopped
	// Uses max local safe timestamp from both chains, pauses at +10, awaits validation at +9
	pausedTimestamp := sys.Supernode.EnsureInteropPaused(sys.L2ACL, sys.L2BCL, 10)
	t.Logger().Info("interop paused", "paused", pausedTimestamp)

	// Compute the initial target block number for each chain based on the paused timestamp
	// Each chain may be at a different block number due to different block periods
	blockPeriodA := sys.L2A.Escape().RollupConfig().BlockTime
	blockPeriodB := sys.L2B.Escape().RollupConfig().BlockTime
	genesisTimeA := sys.L2A.Escape().RollupConfig().Genesis.L2Time
	genesisTimeB := sys.L2B.Escape().RollupConfig().Genesis.L2Time

	initialTargetBlockNumA := (pausedTimestamp - genesisTimeA) / blockPeriodA
	initialTargetBlockNumB := (pausedTimestamp - genesisTimeB) / blockPeriodB

	// check safe heads get to at least that height,
	// let local safe heads run ahead
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.LocalSafe, finalTargetBlockNum, attempts),
		sys.L2BCL.ReachedFn(types.LocalSafe, finalTargetBlockNum, attempts),
		sys.L2ACL.ReachedFn(types.CrossSafe, initialTargetBlockNumA-1, attempts),
		sys.L2BCL.ReachedFn(types.CrossSafe, initialTargetBlockNumB-1, attempts),
	)

	// Expect cross safe and finalized to stall since we paused the interop activity
	numAttempts := 2 // implies a 4s wait
	dsl.CheckAll(t,
		sys.L2ACL.NotAdvancedFn(types.CrossSafe, numAttempts),
		sys.L2BCL.NotAdvancedFn(types.CrossSafe, numAttempts),
		sys.L2ACL.NotAdvancedFn(types.Finalized, numAttempts),
		sys.L2BCL.NotAdvancedFn(types.Finalized, numAttempts),
	)

	// Check EL labels - cross-safeand finalized should be
	// stalled below initial target block numbers
	safeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	safeB := sys.L2ELB.BlockRefByLabel(eth.Safe)
	finalizedA := sys.L2ELA.BlockRefByLabel(eth.Finalized)
	finalizedB := sys.L2ELB.BlockRefByLabel(eth.Finalized)
	require.Less(t, safeA.Number, initialTargetBlockNumA)
	require.Less(t, safeB.Number, initialTargetBlockNumB)
	require.Less(t, finalizedA.Number, initialTargetBlockNumA)
	require.Less(t, finalizedB.Number, initialTargetBlockNumB)

	// Resume interop verification
	// expect cross safe to catch up
	sys.Supernode.ResumeInterop()
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.CrossSafe, finalTargetBlockNum, attempts),
		sys.L2BCL.ReachedFn(types.CrossSafe, finalTargetBlockNum, attempts),
	)

	// check EL labels
	safeA = sys.L2ELA.BlockRefByLabel(eth.Safe)
	safeB = sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.GreaterOrEqual(t, safeA.Number, finalTargetBlockNum)
	require.GreaterOrEqual(t, safeB.Number, finalTargetBlockNum)

	// Snapshot the current safe head to verify finalized catches up
	snapshotSafeA := safeA.Number
	snapshotSafeB := safeB.Number
	t.Logger().Info("snapshotted safe heads", "safeA", snapshotSafeA, "safeB", snapshotSafeB)

	// Sanity check: finalized should be behind safe at this point
	preFinalizedStatusA := sys.L2ACL.SyncStatus()
	preFinalizedStatusB := sys.L2BCL.SyncStatus()
	require.LessOrEqual(t, preFinalizedStatusA.FinalizedL2.Number, snapshotSafeA,
		"finalized A should be at or behind safe head")
	require.LessOrEqual(t, preFinalizedStatusB.FinalizedL2.Number, snapshotSafeB,
		"finalized B should be at or behind safe head")
	t.Logger().Info("pre-finalized state",
		"finalizedA", preFinalizedStatusA.FinalizedL2.Number,
		"finalizedB", preFinalizedStatusB.FinalizedL2.Number)

	// Wait for L1 head to finalise, which should imply L2 finalized head progression
	// Use time travel to reduce walltime of test
	sys.AdvanceTime(90 * time.Second)
	sys.L1Network.WaitForFinalization()

	// Wait for finalized heads to catch up to or past the snapshotted safe heads
	// Finalized advancement depends on L1 finality, so use more attempts
	finalizedAttempts := 30
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.Finalized, snapshotSafeA, finalizedAttempts),
		sys.L2BCL.ReachedFn(types.Finalized, snapshotSafeB, finalizedAttempts),
	)

	// Verify finalized heads on EL
	finalizedA = sys.L2ELA.BlockRefByLabel(eth.Finalized)
	finalizedB = sys.L2ELB.BlockRefByLabel(eth.Finalized)
	require.GreaterOrEqual(t, finalizedA.Number, snapshotSafeA, "finalized A should catch up to safe snapshot")
	require.GreaterOrEqual(t, finalizedB.Number, snapshotSafeB, "finalized B should catch up to safe snapshot")

	// Get current safe heads to verify finalized is still at or behind safe
	currentSafeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	currentSafeB := sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.LessOrEqual(t, finalizedA.Number, currentSafeA.Number,
		"finalized A should be at or behind current safe head")
	require.LessOrEqual(t, finalizedB.Number, currentSafeB.Number,
		"finalized B should be at or behind current safe head")

	// Sanity check: L1 origin of L2 finalized head should be <= L1 finalized head
	l1FinalizedHead := sys.L1EL.BlockRefByLabel(eth.Finalized)
	t.Logger().Info("L1 finalized head", "number", l1FinalizedHead.Number)
	t.Logger().Info("L2A finalized L1 origin", "number", finalizedA.L1Origin.Number)
	t.Logger().Info("L2B finalized L1 origin", "number", finalizedB.L1Origin.Number)

	require.LessOrEqual(t, finalizedA.L1Origin.Number, l1FinalizedHead.Number,
		"L2A finalized block's L1 origin should be at or behind L1 finalized head")
	require.LessOrEqual(t, finalizedB.L1Origin.Number, l1FinalizedHead.Number,
		"L2B finalized block's L1 origin should be at or behind L1 finalized head")
}

// TestSupernodeInterop_SafeHeadWithUnevenProgress tests safe head behavior
// when chains advance at different rates.
//
// This test verifies:
// - Local safe heads can diverge between chains
// - Cross-safe head is gated by the slower chain
// - Safe head advances after slower chain catches up
func TestSupernodeInterop_SafeHeadWithUnevenProgress(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newSupernodeInteropWithTimeTravel(t, 0)
	attempts := 15

	initialTargetBlockNum := uint64(5)
	finalTargetBlockNum := uint64(10)

	// Wait for initial sync
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.LocalSafe, initialTargetBlockNum, attempts),
		sys.L2BCL.ReachedFn(types.LocalSafe, initialTargetBlockNum, attempts),
	)

	baselineLocalSafeB := sys.L2BCL.SyncStatus().LocalSafeL2.Number

	// Stop chain B's batcher to create uneven progress
	sys.L2BatcherB.Stop()

	// Chain A advances while B is frozen
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.LocalSafe, finalTargetBlockNum, attempts),
	)

	unevenStatusA := sys.L2ACL.SyncStatus()
	unevenStatusB := sys.L2BCL.SyncStatus()

	// Chain B's local safe should be relatively stable
	require.LessOrEqual(t, unevenStatusB.LocalSafeL2.Number, baselineLocalSafeB+5,
		"chain B local safe should not advance with batcher stopped")

	// Cross-safe heads should be gated by slower chain
	require.LessOrEqual(t, unevenStatusA.SafeL2.Number, unevenStatusB.LocalSafeL2.Number+2,
		"cross-safe should be gated by slower chain's local safe")

	snapshotCrossSafeA := unevenStatusA.SafeL2.Number

	// Resume chain B's batcher
	sys.L2BatcherB.Start()

	// Chain B catches up
	dsl.CheckAll(t,
		sys.L2BCL.ReachedFn(types.LocalSafe, finalTargetBlockNum, attempts),
	)

	// Cross-safe heads advance after chain B catches up
	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(types.CrossSafe, snapshotCrossSafeA+5, attempts),
		sys.L2BCL.ReachedFn(types.CrossSafe, snapshotCrossSafeA+5, attempts),
	)

	// Check EL labels
	safeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	safeB := sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.Greater(t, safeA.Number, snapshotCrossSafeA)
	require.Greater(t, safeB.Number, snapshotCrossSafeA)
}
