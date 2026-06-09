package supernode_sync_tester

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	nodeSync "github.com/ethereum-optimism/optimism/op-node/rollup/sync"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// All safety heads of a supernode VN driven by a sync-tester EL advance
// alongside the sequencer. Interop active from genesis.
func TestSupernodeVerifierAdvancesViaSyncTester(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleSupernodeWithSyncTester(t)

	dsl.CheckAll(t,
		sys.L2CL.AllHeadsAdvancedFn(),
		sys.SupernodeCL.AllHeadsAdvancedFn(),
	)
}

// Supernode joins a chain already in flight: stop, reset the mocked EL, start.
// VN must catch up across all safety heads (default CLSync + reqresp).
func TestSupernodeVerifierCatchesUpFromCold(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleSupernodeWithSyncTester(t)

	dsl.CheckAll(t,
		sys.L2CL.AllHeadsAdvancedFn(),
		sys.SupernodeCL.AllHeadsAdvancedFn(),
	)

	snap := sys.SupernodeCL.SafetyHeads()
	sys.Supernode.Stop()
	sys.SyncTester.ResetAllSessions()
	sys.Supernode.Start()

	dsl.CheckAll(t, sys.SupernodeCL.AllHeadsReachedFn(snap))
}

// Same cold catch-up under ELSync mode. Mirror of TestSyncTesterELSync with
// the verifier op-node replaced by a supernode VN.
func TestSupernodeVerifierELSyncsFromCold(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewSingleSupernodeWithSyncTester(t,
		presets.WithSupernodeVerifierSyncMode(nodeSync.ELSync),
		presets.WithGlobalSyncTesterELOption(sysgo.SyncTesterELOptionFn(
			func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.SyncTesterELConfig) {
				cfg.ELSyncActive = true
			},
		)),
	)

	dsl.CheckAll(t,
		sys.L2CL.AllHeadsAdvancedFn(),
		sys.SupernodeCL.AllHeadsAdvancedFn(),
	)

	snap := sys.SupernodeCL.SafetyHeads()
	sys.Supernode.Stop()
	sys.SyncTester.ResetAllSessions()
	sys.Supernode.Start()

	dsl.CheckAll(t, sys.SupernodeCL.AllHeadsReachedFn(snap))
}

// Interop activates a few seconds after genesis; the VN must cross the
// activation boundary both initially and during a cold catch-up resync
// (default CLSync + reqresp).
func TestSupernodeVerifierSyncsThroughInteropActivation(gt *testing.T) {
	t := devtest.ParallelT(gt)
	const activationDelay = uint64(6) // ~3 L2 blocks at the 2s block time
	sys := presets.NewSingleSupernodeWithSyncTester(t,
		presets.WithInteropActivationDelay(activationDelay),
	)
	require := t.Require()

	dsl.CheckAll(t,
		sys.L2CL.AllHeadsAdvancedFn(),
		sys.SupernodeCL.AllHeadsAdvancedFn(),
	)
	require.GreaterOrEqual(
		sys.SupernodeCL.HeadBlockRef(safety.LocalUnsafe).Time,
		sys.InteropActivationTime,
		"supernode VN must have crossed the interop activation timestamp",
	)

	snap := sys.SupernodeCL.SafetyHeads()
	sys.Supernode.Stop()
	sys.SyncTester.ResetAllSessions()
	sys.Supernode.Start()

	dsl.CheckAll(t, sys.SupernodeCL.AllHeadsReachedFn(snap))
	require.GreaterOrEqual(
		sys.SupernodeCL.HeadBlockRef(safety.LocalUnsafe).Time,
		sys.InteropActivationTime,
		"resynced supernode VN must again be past the interop activation timestamp",
	)
}

// Same activation-boundary catch-up under ELSync mode.
func TestSupernodeVerifierELSyncsThroughInteropActivation(gt *testing.T) {
	t := devtest.ParallelT(gt)
	const activationDelay = uint64(6) // ~3 L2 blocks at the 2s block time
	sys := presets.NewSingleSupernodeWithSyncTester(t,
		presets.WithInteropActivationDelay(activationDelay),
		presets.WithSupernodeVerifierSyncMode(nodeSync.ELSync),
		presets.WithGlobalSyncTesterELOption(sysgo.SyncTesterELOptionFn(
			func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.SyncTesterELConfig) {
				cfg.ELSyncActive = true
			},
		)),
	)
	require := t.Require()

	dsl.CheckAll(t,
		sys.L2CL.AllHeadsAdvancedFn(),
		sys.SupernodeCL.AllHeadsAdvancedFn(),
	)
	require.GreaterOrEqual(
		sys.SupernodeCL.HeadBlockRef(safety.LocalUnsafe).Time,
		sys.InteropActivationTime,
		"supernode VN must have crossed the interop activation timestamp",
	)

	snap := sys.SupernodeCL.SafetyHeads()
	sys.Supernode.Stop()
	sys.SyncTester.ResetAllSessions()
	sys.Supernode.Start()

	dsl.CheckAll(t, sys.SupernodeCL.AllHeadsReachedFn(snap))
	require.GreaterOrEqual(
		sys.SupernodeCL.HeadBlockRef(safety.LocalUnsafe).Time,
		sys.InteropActivationTime,
		"resynced supernode VN must again be past the interop activation timestamp",
	)
}
