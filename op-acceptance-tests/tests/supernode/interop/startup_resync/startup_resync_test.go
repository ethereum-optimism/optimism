// Package startup_resync contains acceptance tests for the op-supernode
// interop startup rework's cold-start resync path: stopping the supernode,
// deleting its on-disk data dir, and starting a fresh supernode against the
// same chain containers and virtual nodes.
package startup_resync

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

const (
	l2BlockTime         = uint64(1)
	backfillDepth       = 3 * time.Second
	preRestartFinalized = uint64(5)
)

// TestSupernodeResyncResumesAtActivation_PostActivation drives a full
// supernode data-dir wipe after the chain has crossed activation, and
// asserts that cross-safe keeps advancing post-restart and that the
// cold-start backfill restored history into the logs DB.
func TestSupernodeResyncResumesAtActivation_PostActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0,
		presets.WithUniformL2BlockTimes(l2BlockTime),
		presets.WithInteropLogBackfillDepth(backfillDepth),
	)

	sys.Supernode.AwaitBackfillCompleted()

	// Setup: let L2 finalized advance several blocks on both chains. On
	// restart, op-node may drop back as part of its safe start process,
	// but won't go past the finalized head. With finalized well past
	// genesis the post-restart cold-start backfill has a real window to
	// populate, instead of collapsing to empty against a re-recorded
	// genesis SafeDB entry.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.Finalized, preRestartFinalized, 180),
		sys.L2BCL.AdvancedFn(safety.Finalized, preRestartFinalized, 180),
	)

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitBackfillCompleted()

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(safety.CrossSafe, 1, 60),
	)

	// Verify the cold-start backfill repopulated the logs DB.
	sys.Supernode.AssertBackfillCovers(backfillDepth, l2BlockTime,
		sys.L2A.ChainID(), sys.L2B.ChainID())
}

// TestSupernodeResyncSchedulesAtActivation_PreActivation drives a full
// supernode data-dir wipe while interop is scheduled but not yet active,
// and asserts that cold-start init parks the verifier at the (future)
// activation timestamp while cross-safe keeps advancing on both chains.
func TestSupernodeResyncSchedulesAtActivation_PreActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	// 60-minute delay: ensures the chain never approaches activation during
	// the test, so we always exercise the genuine pre-activation cold-start
	// path regardless of CI scheduling variance.
	sys := presets.NewTwoL2SupernodeInterop(t, 60*60,
		presets.WithUniformL2BlockTimes(l2BlockTime),
		presets.WithInteropLogBackfillDepth(backfillDepth),
	)

	sys.Supernode.AwaitBackfillCompleted()
	activation := sys.Supernode.ActivationTimestamp()

	// Setup: let local-safe accumulate enough that op-node's SafeDB has
	// entries to serve to the post-restart cold-start init.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.LocalSafe, 2, 30),
		sys.L2BCL.AdvancedFn(safety.LocalSafe, 2, 30),
	)

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitVerificationStartsAt(activation)

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(safety.CrossSafe, 1, 60),
	)
}
