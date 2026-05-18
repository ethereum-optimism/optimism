// Package startupresync contains shared helpers for the supernode interop
// startup-resync acceptance tests. The individual test cases live in sibling
// packages (post_activation, pre_activation) so each runs in its own test
// binary and shares no in-process state.
package startupresync

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// BackfillDepth is the configured look-back window. Cold-start backfill is a
// no-op whenever verificationStartTimestamp == activationTimestamp (which is
// the case for every test here), but configuring a non-zero depth still
// exercises the cold-start backfill code path.
const BackfillDepth = 60 * time.Second

// PostActivationDelay configures interop to activate shortly after genesis so
// the test can wait until the chain has crossed activation before triggering
// the resync. Long enough that the activation timestamp is well-defined,
// short enough not to bloat test runtime.
const PostActivationDelay = uint64(20)

// PostActivationHistoryAge is how much chain history (measured from genesis)
// each chain must accumulate before we trigger the resync in the
// post-activation test. Strictly larger than PostActivationDelay so both
// chains have at least one safe head past the activation boundary.
const PostActivationHistoryAge = time.Duration(PostActivationDelay+30) * time.Second

// PreActivationDelay configures interop to activate well past where the chain
// will be when the resync happens in the pre-activation test. Combined with
// PreActivationHistoryAge below, this guarantees that at the moment we
// restart the supernode the chain tip is firmly before activation.
const PreActivationDelay = uint64(120)

// PreActivationHistoryAge is how much chain history each chain must
// accumulate before triggering the resync in the pre-activation test. Smaller
// than PreActivationDelay so the chain tip stays comfortably pre-activation.
// Large enough that op-node has recorded multiple SafeDB entries per chain
// (otherwise cold-start init would spin in ErrSafeDBEmpty).
const PreActivationHistoryAge = 20 * time.Second

// NewTestSystem builds a two-L2 supernode-interop system with the requested
// activation delay, time-travel enabled, and a non-zero log-backfill depth so
// the cold-start backfill code path is exercised even when the resulting
// backfill window happens to be empty.
func NewTestSystem(t devtest.T, delaySeconds uint64) *presets.TwoL2SupernodeInterop {
	return presets.NewTwoL2SupernodeInterop(t, delaySeconds,
		presets.WithTimeTravelEnabled(),
		presets.WithInteropLogBackfillDepth(BackfillDepth),
	)
}

// AwaitHistoryAtLeast blocks until both L2 chains' local-safe and cross-safe
// timestamps have advanced at least `age` past genesis. Intended to be called
// before triggering the supernode-data wipe so the subsequent cold-start
// init has SafeDB entries to consult.
func AwaitHistoryAtLeast(t devtest.T, sys *presets.TwoL2SupernodeInterop, age time.Duration) {
	t.Helper()
	ageSec := uint64(age / time.Second)
	deadline := sys.GenesisTime + ageSec
	t.Require().Eventuallyf(func() bool {
		statusA := sys.L2ACL.SyncStatus()
		statusB := sys.L2BCL.SyncStatus()
		return statusA.LocalSafeL2.Time >= deadline &&
			statusB.LocalSafeL2.Time >= deadline &&
			statusA.SafeL2.Time >= deadline &&
			statusB.SafeL2.Time >= deadline
	}, 5*time.Minute, 2*time.Second,
		"both chains must accumulate local+cross safe history of at least %s", age)
}
