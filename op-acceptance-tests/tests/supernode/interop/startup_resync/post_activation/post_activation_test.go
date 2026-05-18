// Package postactivation contains the post-activation resync acceptance test
// for the op-supernode interop startup rework. It lives in its own package
// so it runs in its own test binary, isolated from sibling cases.
package postactivation

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/supernode/interop/startup_resync"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// TestSupernodeResyncResumesAtActivation_PostActivation exercises the
// cold-start resync path when interop is already active:
//
//  1. Bring up a two-L2 supernode with interop scheduled to activate shortly
//     after genesis.
//  2. Let both chains cross the activation boundary so the initial supernode
//     has committed real verifiedDB entries.
//  3. Stop the supernode, delete its entire on-disk data dir (verifiedDB,
//     per-chain logsDB and safe_db), and start a fresh supernode against
//     the same chain containers and virtual nodes. With no prior commits
//     and no SafeDB entries to consult, the new supernode must go through
//     the cold-start path.
//  4. Assert cold-start init completes without panic or fatal error and
//     produces a verifier start at or after the activation floor —
//     verificationStartTimestamp = max(activation, max per-chain first
//     SafeDB entry) is by construction >= activation.
//  5. Assert verification then drives forward from the cold-start anchor by
//     waiting for a post-anchor timestamp to be validated.
func TestSupernodeResyncResumesAtActivation_PostActivation(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := startupresync.NewTestSystem(t, startupresync.PostActivationDelay)

	sys.Supernode.AwaitBackfillCompleted()
	activation := sys.Supernode.ActivationTimestamp()
	t.Require().GreaterOrEqualf(sys.Supernode.VerificationStartTimestamp(), activation,
		"initial verificationStartTimestamp must be >= activation %d", activation)

	startupresync.AwaitHistoryAtLeast(t, sys, startupresync.PostActivationHistoryAge)

	sys.Supernode.RestartWithFreshDataDir()
	sys.Supernode.AwaitBackfillCompleted()

	postRestartStart := sys.Supernode.VerificationStartTimestamp()
	t.Require().GreaterOrEqualf(postRestartStart, activation,
		"post-restart verificationStartTimestamp %d must be >= activation %d",
		postRestartStart, activation)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	sys.Supernode.AwaitValidatedTimestamp(postRestartStart + blockTime)
}
