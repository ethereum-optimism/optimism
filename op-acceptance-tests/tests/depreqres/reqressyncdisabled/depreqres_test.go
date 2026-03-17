package depreqres

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/depreqres/common"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestUnsafeChainNotStalling_DisabledReqRespSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t, common.ReqRespSyncDisabledOpts(sync.ELSync)...)
	require := t.Require()
	l := t.Logger()

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	delta := uint64(3)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, delta, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, delta, 30),
	)

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	l.Info("Wait for L2CLB head to stabilize after disconnect")
	// After DisconnectPeer, buffered gossip messages may still arrive and advance
	// L2CLB's unsafe head. Poll until the head is stable before taking a snapshot.
	ssB_before := sys.L2CLB.SyncStatus()
	require.Eventually(func() bool {
		next := sys.L2CLB.SyncStatus()
		stable := next.UnsafeL2.Number == ssB_before.UnsafeL2.Number
		ssB_before = next
		return stable
	}, 5*time.Second, 200*time.Millisecond, "L2CLB head should stabilize after disconnect")

	ssA_before := sys.L2CL.SyncStatus()
	l.Info("L2CL status before wait", "unsafeL2", ssA_before.UnsafeL2.ID())
	l.Info("L2CLB status before wait", "unsafeL2", ssB_before.UnsafeL2.ID())

	l.Info("Confirm that the unsafe chain for L2CL advances while L2CLB stalls")
	sys.L2CL.AdvancedUnsafe(delta, 30)

	ssA_after := sys.L2CL.SyncStatus()
	ssB_after := sys.L2CLB.SyncStatus()
	l.Info("L2CL status after wait", "unsafeL2", ssA_after.UnsafeL2.ID())
	l.Info("L2CLB status after wait", "unsafeL2", ssB_after.UnsafeL2.ID())

	require.Greater(ssA_after.UnsafeL2.Number, ssA_before.UnsafeL2.Number, "unsafe chain for L2CL should have advanced")
	require.Equal(ssB_after.UnsafeL2.Number, ssB_before.UnsafeL2.Number, "unsafe chain for L2CLB should have stalled")

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the unsafe chain for L2CLB can advance")
	sys.L2CLB.Reached(types.LocalUnsafe, ssA_after.UnsafeL2.Number, 30)
	sys.L2ELB.Reached(eth.Unsafe, ssA_after.UnsafeL2.Number, 30)
}
