// Package preactivation contains the pre-activation resync acceptance test
// for the op-supernode interop startup rework. It lives in its own package
// so it runs in its own test binary, isolated from sibling cases.
package preactivation

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/supernode/interop/startup_resync"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// TestSupernodeResyncResumesAtActivation_PreActivation exercises the
// cold-start resync path while interop is scheduled but not yet active:
//
//  1. Bring up a two-L2 supernode with interop activation set well past
//     genesis.
//  2. Let both chains accumulate enough safe-head history that the SafeDBs
//     are non-empty, while keeping the chain tip firmly pre-activation.
//  3. Stop the supernode, delete its entire on-disk data dir, and start a
//     fresh supernode against the same chain containers and virtual nodes.
//     The replacement has no prior verifiedDB commits, no logsDB / safe_db
//     state, and the chain tip is still before activation.
//  4. Assert cold-start init completes and the verifier resumes at exactly
//     the activation timestamp — every chain's first SafeDB entry is before
//     activation, so max(activation, max firstSafeDB) == activation.
//  5. Assert pre-activation RPC queries return valid Data via the optimistic
//     short-circuit, despite no verifiedDB commits existing.
//  6. Advance time past activation and assert the verifier transitions to
//     producing committed results without any further restart.
func TestSupernodeResyncResumesAtActivation_PreActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := startupresync.NewTestSystem(t, startupresync.PreActivationDelay)

	sys.Supernode.AwaitBackfillCompleted()
	activation := sys.Supernode.ActivationTimestamp()
	t.Require().Equalf(activation, sys.Supernode.VerificationStartTimestamp(),
		"pre-activation: initial verificationStartTimestamp must equal activation %d",
		activation)

	startupresync.AwaitHistoryAtLeast(t, sys, startupresync.PreActivationHistoryAge)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	preActivationTs := sys.GenesisTime + blockTime

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitVerificationStartsAt(activation)
	sys.Supernode.AwaitValidatedTimestamp(preActivationTs)

	sys.AdvanceTime(time.Duration(startupresync.PreActivationDelay) * time.Second)
	sys.Supernode.AwaitValidatedTimestamp(activation + blockTime)
}
