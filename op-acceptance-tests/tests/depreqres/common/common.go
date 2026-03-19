package common

import (
	"testing"
	"time"

	bss "github.com/ethereum-optimism/optimism/op-batcher/batcher"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
			cfg.UseReqRespSync = false
		}))
}

func syncModeReqRespSyncOpt() presets.Option {
	return presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
		func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.UseReqRespSync = true
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
	}
}

func SyncModeReqRespSyncOpts(syncMode sync.Mode) []presets.Option {
	return []presets.Option{
		syncModeOpt(syncMode),
		syncModeReqRespSyncOpt(),
		noDiscoveryOpt(),
		batcherStoppedOpt(),
	}
}

func UnsafeChainNotStalling_Disconnect(gt *testing.T, syncMode sync.Mode, sleep time.Duration, opts ...presets.Option) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t, opts...)
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
	sys.L2CLB.WaitForStall(types.LocalUnsafe)
	ssB_before := sys.L2CLB.SyncStatus()

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

func UnsafeChainNotStalling_RestartOpNode(gt *testing.T, syncMode sync.Mode, sleep time.Duration, opts ...presets.Option) {
	t := devtest.SerialT(gt)
	sys := presets.NewSingleChainMultiNodeWithoutCheck(t, opts...)
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
	sys.L2CLB.WaitForStall(types.LocalUnsafe)
	ssB_before := sys.L2CLB.SyncStatus()

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
