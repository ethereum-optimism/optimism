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

// activationDelay schedules interop activation shortly after genesis so the
// test crosses activation quickly without bloating runtime.
const activationDelay = uint64(20)

// TestSupernodeResyncResumesAtActivation_PostActivation drives a full
// supernode data-dir wipe after the chain has crossed activation, and
// asserts cross-safe advances post-restart — proving the sequencer is still
// producing blocks and the supernode is validating them.
func TestSupernodeResyncResumesAtActivation_PostActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, activationDelay,
		presets.WithTimeTravelEnabled(),
		presets.WithInteropLogBackfillDepth(60*time.Second),
	)

	sys.Supernode.AwaitBackfillCompleted()
	activation := sys.Supernode.ActivationTimestamp()
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Setup: prove the initial supernode is committing past activation.
	sys.Supernode.AwaitValidatedTimestamp(activation + 5*blockTime)

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitBackfillCompleted()

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(types.CrossSafe, 1, 60),
	)
}
