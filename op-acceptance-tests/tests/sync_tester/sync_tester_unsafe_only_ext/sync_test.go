package sync_tester_unsafe_only_ext

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestSyncTesterUnsafeOnlyReachUnsafeTip(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()

	sys := presets.NewMinimalExternalEL(t)
	sys.L2EL.UnsafeHead().IsGenesis()

	// Check external read only EL is advancing
	sys.L2ELReadOnly.Advanced(eth.Unsafe, 3)

	unsafeTip := sys.L2ELReadOnly.UnsafeHead()
	unsafeTipNum := unsafeTip.BlockRef.Number
	startNum := unsafeTipNum - 3
	// Trigger and finish EL Sync
	for i := startNum; i <= unsafeTipNum; i++ {
		sys.L2CL.SignalTarget(sys.L2ELReadOnly, i)
	}

	sys.L2EL.Reached(eth.Unsafe, unsafeTipNum, 5)
	require.Equal(unsafeTip.BlockRef, sys.L2EL.UnsafeHead().BlockRef)

	// Make sure the unsafe only CL can still advance unsafe
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
}
