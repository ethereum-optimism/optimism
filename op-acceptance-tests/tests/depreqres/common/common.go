package common

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// stableSyncStatus returns the sync status of node after any in-flight gossip messages
// have been drained. DisconnectPeer closes the libp2p connection but a buffered gossip
// payload can still arrive and be processed via AddUnsafePayload (SyncModeReqResp=true
// routes CL gossip through the CLSync path even in ELSync mode). Polling until the
// head is stable ensures the snapshot reflects a quiesced state.
func stableSyncStatus(require *testreq.Assertions, node *dsl.L2CLNode) *eth.SyncStatus {
	ss := node.SyncStatus()
	require.Eventually(func() bool {
		next := node.SyncStatus()
		stable := next.UnsafeL2.Number == ss.UnsafeL2.Number
		ss = next
		return stable
	}, 5*time.Second, 200*time.Millisecond, "L2CLB head should stabilize after disconnect")
	return ss
}

func UnsafeChainNotStalling_Disconnect(gt *testing.T, syncMode sync.Mode, sleep time.Duration) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()
	l := t.Logger().With("syncmode", syncMode)

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	target := uint64(3)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, target, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, target, 30),
	)

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	ssA_before := sys.L2CL.SyncStatus()
	ssB_before := stableSyncStatus(require, sys.L2CLB)

	l.Info("L2CL status before delay", "unsafeL2", ssA_before.UnsafeL2.ID(), "safeL2", ssA_before.SafeL2.ID())
	l.Info("L2CLB status before delay", "unsafeL2", ssB_before.UnsafeL2.ID(), "safeL2", ssB_before.SafeL2.ID())

	time.Sleep(sleep)

	ssA_after := sys.L2CL.SyncStatus()
	ssB_after := sys.L2CLB.SyncStatus()

	l.Info("L2CL status after delay", "unsafeL2", ssA_after.UnsafeL2.ID(), "safeL2", ssA_after.SafeL2.ID())
	l.Info("L2CLB status after delay", "unsafeL2", ssB_after.UnsafeL2.ID(), "safeL2", ssB_after.SafeL2.ID())

	require.Greater(ssA_after.UnsafeL2.Number, ssA_before.UnsafeL2.Number, "unsafe chain for L2CL should have advanced")
	require.Equal(ssB_after.UnsafeL2.Number, ssB_before.UnsafeL2.Number, "unsafe chain for L2CLB should have stalled")

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the unsafe chain for L2CLB is not stalled")
	sys.L2CLB.Reached(types.LocalUnsafe, ssA_after.UnsafeL2.Number, 30)
	sys.L2ELB.Reached(eth.Unsafe, ssA_after.UnsafeL2.Number, 30)
}

func UnsafeChainNotStalling_RestartOpNode(gt *testing.T, syncMode sync.Mode, sleep time.Duration) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t)
	require := t.Require()
	l := t.Logger().With("syncmode", syncMode)

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	target := uint64(3)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(types.LocalUnsafe, target, 30),
		sys.L2CLB.AdvancedFn(types.LocalUnsafe, target, 30),
	)

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	ssA_before := sys.L2CL.SyncStatus()
	ssB_before := stableSyncStatus(require, sys.L2CLB)

	l.Info("L2CL status before delay", "unsafeL2", ssA_before.UnsafeL2.ID(), "safeL2", ssA_before.SafeL2.ID())
	l.Info("L2CLB status before delay", "unsafeL2", ssB_before.UnsafeL2.ID(), "safeL2", ssB_before.SafeL2.ID())

	sys.L2CLB.Stop()

	time.Sleep(sleep)

	sys.L2CLB.Start()

	ssA_after := sys.L2CL.SyncStatus()
	ssB_after := sys.L2CLB.SyncStatus()

	l.Info("L2CL status after delay", "unsafeL2", ssA_after.UnsafeL2.ID(), "safeL2", ssA_after.SafeL2.ID())
	l.Info("L2CLB status after delay", "unsafeL2", ssB_after.UnsafeL2.ID(), "safeL2", ssB_after.SafeL2.ID())

	require.Greater(ssA_after.UnsafeL2.Number, ssA_before.UnsafeL2.Number, "unsafe chain for L2CL should have advanced")
	require.LessOrEqual(ssB_after.UnsafeL2.Number, ssB_before.UnsafeL2.Number, "unsafe chain for L2CLB should have stalled")

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the unsafe chain for L2CLB is not stalled")
	sys.L2CLB.Reached(types.LocalUnsafe, ssA_after.UnsafeL2.Number, 30)
	sys.L2ELB.Reached(eth.Unsafe, ssA_after.UnsafeL2.Number, 30)
}
