package common

import (
	"testing"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/log"
)

func syncModeOpt(syncMode sync.Mode) presets.Option {
	return presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
		func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			if syncMode == sync.CLSync {
				cfg.SequencerSyncMode = sync.CLSync
			}
			cfg.VerifierSyncMode = syncMode
		}))
}

func reqRespSyncDisabledOpt() presets.Option {
	return presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
		func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.EnableReqRespSync = false
		}))
}

func noDiscoveryOpt() presets.Option {
	return presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
		func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.NoDiscovery = true
		}))
}

func batcherStoppedOpt() presets.Option {
	return presets.WithBatcherOption(func(_ sysgo.ComponentTarget, cfg *bss.CLIConfig) {
		cfg.Stopped = true
	})
}

func ReqRespSyncDisabledOpts(syncMode sync.Mode) []presets.Option {
	return []presets.Option{
		syncModeOpt(syncMode),
		reqRespSyncDisabledOpt(),
		noDiscoveryOpt(),
		batcherStoppedOpt(),
		presets.WithUniformL2BlockTimes(1),
	}
}

func UnsafeChainNotStalling_Disconnect(gt *testing.T, syncMode sync.Mode, advanceBlocks uint64, opts ...presets.Option) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t, opts...)
	require := t.Require()
	l := t.Logger().With("syncmode", syncMode)

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	target := uint64(3)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalUnsafe, target, 30),
		sys.L2CLB.AdvancedFn(safety.LocalUnsafe, target, 30),
	)

	logPeerState(l, "L2CLB", sys.L2CLB)
	logPeerState(l, "L2CL", sys.L2CL)

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	sys.L2CLB.WaitForPeerDisconnected(sys.L2CL)
	sys.L2CL.WaitForPeerDisconnected(sys.L2CLB)

	sys.L2CLB.WaitForStall(safety.LocalUnsafe)
	ssB_before := sys.L2CLB.SyncStatus()

	l.Info("L2CLB stalled", "unsafeL2", ssB_before.UnsafeL2.ID(), "safeL2", ssB_before.SafeL2.ID())

	l.Info("Wait for sequencer to advance while verifier is disconnected", "advanceBlocks", advanceBlocks)
	// Allow generous time: advanceBlocks * ~2s block time, plus buffer for CI pressure.
	advanceAttempts := int(advanceBlocks*2 + 30)
	sys.L2CL.Advanced(safety.LocalUnsafe, advanceBlocks, advanceAttempts)

	ssA_after := sys.L2CL.SyncStatus()
	ssB_after := sys.L2CLB.SyncStatus()

	l.Info("L2CL status after advance", "unsafeL2", ssA_after.UnsafeL2.ID(), "safeL2", ssA_after.SafeL2.ID())
	l.Info("L2CLB status after advance", "unsafeL2", ssB_after.UnsafeL2.ID(), "safeL2", ssB_after.SafeL2.ID())

	logPeerState(l, "L2CLB", sys.L2CLB)
	logPeerState(l, "L2CL", sys.L2CL)

	require.Equal(ssB_after.UnsafeL2.Number, ssB_before.UnsafeL2.Number, "unsafe chain for L2CLB should have stalled")

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the unsafe chain for L2CLB is not stalled")
	sys.L2CLB.Reached(safety.LocalUnsafe, ssA_after.UnsafeL2.Number, 30)
	sys.L2ELB.Reached(eth.Unsafe, ssA_after.UnsafeL2.Number, 30)
}

func UnsafeChainNotStalling_RestartOpNode(gt *testing.T, syncMode sync.Mode, advanceBlocks uint64, opts ...presets.Option) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t, opts...)
	require := t.Require()
	l := t.Logger().With("syncmode", syncMode)

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	target := uint64(3)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalUnsafe, target, 30),
		sys.L2CLB.AdvancedFn(safety.LocalUnsafe, target, 30),
	)

	logPeerState(l, "L2CLB", sys.L2CLB)
	logPeerState(l, "L2CL", sys.L2CL)

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	sys.L2CLB.WaitForPeerDisconnected(sys.L2CL)
	sys.L2CL.WaitForPeerDisconnected(sys.L2CLB)

	sys.L2CLB.WaitForStall(safety.LocalUnsafe)
	ssB_before := sys.L2CLB.SyncStatus()

	l.Info("L2CLB stalled", "unsafeL2", ssB_before.UnsafeL2.ID(), "safeL2", ssB_before.SafeL2.ID())

	sys.L2CLB.Stop()

	l.Info("Wait for sequencer to advance while verifier is stopped", "advanceBlocks", advanceBlocks)
	advanceAttempts := int(advanceBlocks*2 + 30)
	sys.L2CL.Advanced(safety.LocalUnsafe, advanceBlocks, advanceAttempts)

	sys.L2CLB.Start()

	ssA_after := sys.L2CL.SyncStatus()
	ssB_after := sys.L2CLB.SyncStatus()

	l.Info("L2CL status after advance", "unsafeL2", ssA_after.UnsafeL2.ID(), "safeL2", ssA_after.SafeL2.ID())
	l.Info("L2CLB status after advance", "unsafeL2", ssB_after.UnsafeL2.ID(), "safeL2", ssB_after.SafeL2.ID())

	logPeerState(l, "L2CLB", sys.L2CLB)
	logPeerState(l, "L2CL", sys.L2CL)

	require.LessOrEqual(ssB_after.UnsafeL2.Number, ssB_before.UnsafeL2.Number, "unsafe chain for L2CLB should have stalled")

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the unsafe chain for L2CLB is not stalled")
	sys.L2CLB.Reached(safety.LocalUnsafe, ssA_after.UnsafeL2.Number, 30)
	sys.L2ELB.Reached(eth.Unsafe, ssA_after.UnsafeL2.Number, 30)
}

// UnsafeChainNotStalling_Backfill drives the verifier EL through backfill
// (staged pipeline) sync: the unsafe gap accumulated while the CL P2P link is
// severed exceeds the EL's backfill threshold (32 blocks in reth), so the gap
// is filled by the sync pipeline downloading from EL peers, rather than by
// live sync as in UnsafeChainNotStalling_Disconnect. The EL peering is
// deliberately basic (not trusted) so peer-management regressions around
// backfill — reputation bans, disconnects, eviction — fail the test instead
// of being masked by trusted-peer exemptions.
func UnsafeChainNotStalling_Backfill(gt *testing.T, syncMode sync.Mode, advanceBlocks uint64, opts ...presets.Option) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutP2PWithoutCheck(t, opts...)
	require := t.Require()
	l := t.Logger().With("syncmode", syncMode)

	l.Info("Peer the EL nodes (basic, ban-sensitive) and the CL nodes")
	sys.L2EL.PeerWithBasic(sys.L2ELB)
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	l.Info("Confirm that the CL nodes are progressing the unsafe chain")
	target := uint64(3)
	dsl.CheckAll(t,
		sys.L2CL.AdvancedFn(safety.LocalUnsafe, target, 30),
		sys.L2CLB.AdvancedFn(safety.LocalUnsafe, target, 30),
	)

	l.Info("Disconnect L2CL from L2CLB, and vice versa")
	sys.L2CLB.DisconnectPeer(sys.L2CL)
	sys.L2CL.DisconnectPeer(sys.L2CLB)

	sys.L2CLB.WaitForPeerDisconnected(sys.L2CL)
	sys.L2CL.WaitForPeerDisconnected(sys.L2CLB)

	sys.L2CLB.WaitForStall(safety.LocalUnsafe)
	ssB_before := sys.L2CLB.SyncStatus()

	l.Info("L2CLB stalled", "unsafeL2", ssB_before.UnsafeL2.ID(), "safeL2", ssB_before.SafeL2.ID())

	l.Info("Wait for sequencer to advance past the EL backfill threshold while verifier is disconnected", "advanceBlocks", advanceBlocks)
	advanceAttempts := int(advanceBlocks*2 + 30)
	sys.L2CL.Advanced(safety.LocalUnsafe, advanceBlocks, advanceAttempts)

	require.Equal(sys.L2CLB.SyncStatus().UnsafeL2.Number, ssB_before.UnsafeL2.Number, "unsafe chain for L2CLB should have stalled")

	l.Info("Confirm the EL peering survived the stall")
	sys.L2EL.VerifyPeeredWith(sys.L2ELB)

	l.Info("Re-connect L2CL to L2CLB")
	sys.L2CLB.ConnectPeer(sys.L2CL)
	sys.L2CL.ConnectPeer(sys.L2CLB)

	ssA_after := sys.L2CL.SyncStatus()
	l.Info("Confirm that L2CLB fills the unsafe gap via EL sync", "target", ssA_after.UnsafeL2.ID())
	sys.L2CLB.Reached(safety.LocalUnsafe, ssA_after.UnsafeL2.Number, 60)
	sys.L2ELB.Reached(eth.Unsafe, ssA_after.UnsafeL2.Number, 60)

	l.Info("Confirm the EL peering survived the backfill")
	sys.L2EL.VerifyPeeredWith(sys.L2ELB)
	sys.L2ELB.VerifyPeeredWith(sys.L2EL)
}

func logPeerState(l log.Logger, name string, cl *dsl.L2CLNode) {
	peers := cl.Peers()
	l.Info("Peer state",
		"node", name,
		"totalConnected", peers.TotalConnected,
	)
	for id, p := range peers.Peers {
		l.Info("Peer detail",
			"node", name,
			"peerID", id,
			"connectedness", p.Connectedness,
			"direction", p.Direction,
			"gossipBlocks", p.GossipBlocks,
		)
	}
}
