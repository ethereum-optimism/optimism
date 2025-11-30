package sync_tester_follow_l2_ext

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterFollowL2ReachTips(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	logger := t.Logger()

	sys := presets.NewMinimalExternalEL(t)
	sys.L2EL.UnsafeHead().IsGenesis()

	// Check external read only EL is advancing
	sys.L2ELReadOnly.Advanced(eth.Unsafe, 3)

	initialELSyncTarget := sys.L2ELReadOnly.FinalizedHead()
	initialELSyncTargetNum := initialELSyncTarget.BlockRef.Number
	// Once op-node completes its initial EL sync, the CL incorrectly sets the
	// safe, finalized, and head blocks to match the observed unsafe head.
	// Issue: https://github.com/ethereum-optimism/optimism/issues/17631
	// In this test, we verify that a follow-L2-enabled op-node correctly mirrors
	// the external safe and finalized heads. To properly observe this mirroring,
	// we complete the initial EL sync up to the finalized head.
	startNum := initialELSyncTargetNum - 2
	// Trigger and finish EL Sync
	for i := startNum; i <= initialELSyncTargetNum; i++ {
		sys.L2CL.SignalTarget(sys.L2ELReadOnly, i)
	}

	sys.L2EL.Reached(eth.Unsafe, initialELSyncTargetNum, 5)
	require.Equal(initialELSyncTarget.BlockRef, sys.L2EL.UnsafeHead().BlockRef)
	require.Equal(initialELSyncTarget.BlockRef, sys.L2EL.SafeHead().BlockRef)
	require.Equal(initialELSyncTarget.BlockRef, sys.L2EL.FinalizedHead().BlockRef)
	logger.Info("Initial EL Sync done", "target", initialELSyncTarget)

	// After initial EL sync is done, follow l2 starts to mirror external safe and finalized

	// Choose target near the real unsafe tip
	unsafeTipNum := sys.L2ELReadOnly.UnsafeHead().BlockRef.Number - 10
	// Make sure the follow L2 CL can still advance unsafe
	target := unsafeTipNum + 3
	sys.L2ELReadOnly.Reached(eth.Unsafe, target, 3)
	for i := unsafeTipNum + 1; i <= target; i++ {
		sys.L2CL.SignalTarget(sys.L2ELReadOnly, i)
	}
	sys.L2EL.Reached(eth.Unsafe, target, 5)
	sys.L2CL.Reached(types.LocalUnsafe, target, 5)

	// Check unsafe gap is closed
	target = unsafeTipNum + 9
	sys.L2ELReadOnly.Reached(eth.Unsafe, target, 6)
	for i := unsafeTipNum + 6; i <= target; i++ {
		sys.L2CL.SignalTarget(sys.L2ELReadOnly, i)
	}
	sys.L2EL.Reached(eth.Unsafe, target, 5)
	sys.L2CL.Reached(types.LocalUnsafe, target, 5)

	// Check safe and finalized head is synced with follow l2
	dsl.CheckAll(t,
		sys.L2CL.MatchedFn(sys.L2ELReadOnly, types.LocalSafe, 10),
		sys.L2CL.MatchedFn(sys.L2ELReadOnly, types.Finalized, 10),
		sys.L2EL.MatchedFn(sys.L2ELReadOnly, types.LocalSafe, 10),
		sys.L2EL.MatchedFn(sys.L2ELReadOnly, types.Finalized, 10),
	)

	logger.Info("External L2", "safe", sys.L2ELReadOnly.SafeHead().BlockRef, "finalized", sys.L2ELReadOnly.FinalizedHead().BlockRef)
	logger.Info("Follow L2", "safe", sys.L2EL.SafeHead().BlockRef, "finalized", sys.L2EL.FinalizedHead().BlockRef)
}
