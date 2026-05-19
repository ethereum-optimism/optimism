// Package startup_resync contains acceptance tests for the op-supernode
// interop startup rework's cold-start resync path.
package startup_resync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

const (
	l2BlockTime         = uint64(1)
	backfillDepth       = 3 * time.Second
	preRestartFinalized = uint64(5)
)

// TestSupernodeResyncResumesAtActivation_PostActivation wipes the supernode
// data dir after the chain has crossed activation and asserts cross-safe
// resumes. The "EL data wiped" subtest additionally wipes the supernode-
// fronted verifier EL so it must execution-layer-sync from the chain's
// sequencer EL.
func TestSupernodeResyncResumesAtActivation_PostActivation(gt *testing.T) {
	for _, tc := range []struct {
		name        string
		restartOpts []func(*dsl.RestartOpts)
	}{
		{"EL data intact", nil},
		{"EL data wiped", []func(*dsl.RestartOpts){dsl.WithELWiped}},
	} {
		gt.Run(tc.name, func(gt *testing.T) {
			runPostActivationResync(gt, tc.restartOpts)
		})
	}
}

// TestSupernodeResyncSchedulesAtActivation_PreActivation wipes the supernode
// data dir while interop is still scheduled and asserts cold-start parks the
// verifier at activation. The "EL data wiped" subtest additionally wipes the
// supernode-fronted EL.
func TestSupernodeResyncSchedulesAtActivation_PreActivation(gt *testing.T) {
	for _, tc := range []struct {
		name        string
		restartOpts []func(*dsl.RestartOpts)
	}{
		{"EL data intact", nil},
		{"EL data wiped", []func(*dsl.RestartOpts){dsl.WithELWiped}},
	} {
		gt.Run(tc.name, func(gt *testing.T) {
			runPreActivationResync(gt, tc.restartOpts)
		})
	}
}

func runPostActivationResync(gt *testing.T, restartOpts []func(*dsl.RestartOpts)) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInteropPeerEL(t, 0,
		presets.WithUniformL2BlockTimes(l2BlockTime),
		presets.WithInteropLogBackfillDepth(backfillDepth),
	)
	sys.Supernode.AwaitBackfillCompleted()

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.Finalized, preRestartFinalized, 180),
		sys.L2BCL.AdvancedFn(types.Finalized, preRestartFinalized, 180),
	)

	activation := sys.Supernode.ActivationTimestamp()
	sys.Supernode.RestartWithFreshDataDir(restartOpts...)
	sys.Supernode.AwaitVerificationStartsAtOrAfter(activation)
	sys.Supernode.AwaitBackfillCompleted()

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(types.CrossSafe, 1, 60),
	)

	sys.Supernode.AssertBackfillCovers(backfillDepth, l2BlockTime,
		sys.L2A.ChainID(), sys.L2B.ChainID())
}

func runPreActivationResync(gt *testing.T, restartOpts []func(*dsl.RestartOpts)) {
	t := devtest.SerialT(gt)
	// Delay activation by an hour so the chain stays well below it throughout
	// the test, and cold-start always parks at the future activation timestamp
	// regardless of CI scheduling variance.
	sys := presets.NewTwoL2SupernodeInteropPeerEL(t, uint64(60*60),
		presets.WithUniformL2BlockTimes(l2BlockTime),
		presets.WithInteropLogBackfillDepth(backfillDepth),
	)
	sys.Supernode.AwaitBackfillCompleted()
	activation := sys.Supernode.ActivationTimestamp()

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.LocalSafe, 2, 60),
		sys.L2BCL.AdvancedFn(types.LocalSafe, 2, 60),
	)

	sys.Supernode.RestartWithFreshDataDir(restartOpts...)
	sys.Supernode.AwaitVerificationStartsAt(activation)

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(types.CrossSafe, 1, 60),
	)
}
