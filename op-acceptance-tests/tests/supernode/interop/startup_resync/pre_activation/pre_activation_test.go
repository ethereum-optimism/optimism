// Package preactivation contains the pre-activation resync acceptance test
// for the op-supernode interop startup rework.
package preactivation

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// activationDelay schedules interop activation far enough past genesis that
// the chain tip stays comfortably pre-activation through the whole test,
// even under CI scheduling variance.
const activationDelay = uint64(300)

// TestSupernodeResyncResumesAtActivation_PreActivation drives a full
// supernode data-dir wipe while interop is scheduled but not yet active,
// and asserts:
//
//   - cold-start init resumes the verifier at the activation timestamp
//     (which is still in the future), and
//   - cross-safe keeps advancing on both chains while interop sits idle
//     waiting for activation.
//
// The verifier transition at activation itself is covered by other tests;
// here we only assert pre-activation cold-start behaviour.
func TestSupernodeResyncResumesAtActivation_PreActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, activationDelay,
		presets.WithTimeTravelEnabled(),
		presets.WithInteropLogBackfillDepth(60*time.Second),
	)

	sys.Supernode.AwaitBackfillCompleted()
	activation := sys.Supernode.ActivationTimestamp()

	// Setup: let local-safe accumulate enough that op-node's SafeDB has
	// entries to serve to the post-restart cold-start init.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.LocalSafe, 2, 30),
		sys.L2BCL.AdvancedFn(types.LocalSafe, 2, 30),
	)

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitVerificationStartsAt(activation)

	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(types.CrossSafe, 1, 60),
	)
}
