package depreqres

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

func TestUnsafeChainStalling_DisabledReqRespSync(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNode(t)
	require := t.Require()
	l := t.Logger()

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	target := uint64(10)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, target, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, target, 30),
	)

	l.Info("Stop the L2 batcher")
	sys.L2Batcher.Stop()

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	ssA_before := sys.L2CL.SyncStatus()
	ssB_before := sys.L2CLB.SyncStatus()

	l.Info("L2CL status before delay", "unsafeL2", ssA_before.UnsafeL2.ID(), "safeL2", ssA_before.SafeL2.ID())
	l.Info("L2CLB status before delay", "unsafeL2", ssB_before.UnsafeL2.ID(), "safeL2", ssB_before.SafeL2.ID())

	time.Sleep(20 * time.Second)

	ssA_after := sys.L2CL.SyncStatus()
	ssB_after := sys.L2CLB.SyncStatus()

	l.Info("L2CL status after delay", "unsafeL2", ssA_after.UnsafeL2.ID(), "safeL2", ssA_after.SafeL2.ID())
	l.Info("L2CLB status after delay", "unsafeL2", ssB_after.UnsafeL2.ID(), "safeL2", ssB_after.SafeL2.ID())

	require.Greater(ssA_after.UnsafeL2.Number, ssA_before.UnsafeL2.Number, "unsafe chain for L2CL should have advanced")
	require.Equal(ssB_after.UnsafeL2.Number, ssB_before.UnsafeL2.Number, "unsafe chain for L2CLB should have stalled")

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the unsafe chain for L2CLB is stalled")
	sys.L2CLB.NotAdvanced(types.LocalUnsafe, 10)
}
