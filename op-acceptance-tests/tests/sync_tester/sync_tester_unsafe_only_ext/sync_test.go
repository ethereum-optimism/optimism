package sync_tester_unsafe_only_ext

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
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
}
