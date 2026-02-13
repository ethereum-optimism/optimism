package interop

import (
	"testing"

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

	// Expect cross safe to stall since we paused the interop activity
	numAttempts := 2 // implies a 4s wait
	dsl.CheckAll(t,
		sys.L2ACL.NotAdvancedFn(types.CrossSafe, numAttempts),
		sys.L2BCL.NotAdvancedFn(types.CrossSafe, numAttempts),
	)

	// Check EL labels - cross-safe should be stalled below initial target block numbers
	safeA := sys.L2ELA.BlockRefByLabel(eth.Safe)
	safeB := sys.L2ELB.BlockRefByLabel(eth.Safe)
	require.Less(t, safeA.Number, initialTargetBlockNumA)
	require.Less(t, safeB.Number, initialTargetBlockNumB)

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
