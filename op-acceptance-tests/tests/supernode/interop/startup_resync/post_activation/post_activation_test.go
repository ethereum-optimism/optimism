// Package postactivation contains the post-activation resync acceptance test
// for the op-supernode interop startup rework.
package postactivation

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// TestSupernodeResyncResumesAtActivation_PostActivation drives a full
// supernode data-dir wipe after the chain has crossed activation, and
// asserts cross-safe advances post-restart — proving the sequencer is still
// producing blocks and the supernode is validating them.
func TestSupernodeResyncResumesAtActivation_PostActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, 0,
		presets.WithTimeTravelEnabled(),
		presets.WithInteropLogBackfillDepth(60*time.Second),
	)

	sys.Supernode.AwaitBackfillCompleted()

	// Setup: prove the initial supernode is producing committed cross-safe
	// entries on both chains.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.CrossSafe, 5, 60),
		sys.L2BCL.AdvancedFn(types.CrossSafe, 5, 60),
	)

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitBackfillCompleted()

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(types.CrossSafe, 1, 60),
	)
}
