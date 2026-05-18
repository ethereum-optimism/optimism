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
// the chain tip stays comfortably pre-activation when we restart the
// supernode, even with time-travel.
const activationDelay = uint64(120)

// TestSupernodeResyncResumesAtActivation_PreActivation drives a full
// supernode data-dir wipe while interop is scheduled but not yet active,
// and asserts:
//
//   - cold-start init resumes the verifier at the activation timestamp,
//   - pre-activation RPC queries are answered via the optimistic
//     short-circuit despite no verifiedDB commits,
//   - and once time advances past activation, cross-safe advances on both
//     chains without any further restart.
func TestSupernodeResyncResumesAtActivation_PreActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeInterop(t, activationDelay,
		presets.WithTimeTravelEnabled(),
		presets.WithInteropLogBackfillDepth(60*time.Second),
	)

	sys.Supernode.AwaitBackfillCompleted()
	activation := sys.Supernode.ActivationTimestamp()
	blockTime := sys.L2A.Escape().RollupConfig().BlockTime

	// Setup: let local-safe accumulate enough that op-node's SafeDB has
	// entries to serve to the post-restart cold-start init.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.LocalSafe, 2, 30),
		sys.L2BCL.AdvancedFn(types.LocalSafe, 2, 30),
	)

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitVerificationStartsAt(activation)
	sys.Supernode.AwaitValidatedTimestamp(sys.GenesisTime + blockTime)

	sys.AdvanceTime(time.Duration(activationDelay) * time.Second)
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(types.CrossSafe, 1, 60),
		sys.L2BCL.AdvancedFn(types.CrossSafe, 1, 60),
	)
}
