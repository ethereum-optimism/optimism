// Package backfillutil contains shared helpers used by the interop log-backfill
// acceptance tests. The individual test cases live in sibling packages
// (backfill/happy) so that each runs in its own test binary and shares no
// in-process state.
package backfill

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// BackfillDepth is the look-back window configured on every test preset.
// Any value larger than a few block times is enough to exercise the full
// backfill path while still keeping test runtime reasonable.
const BackfillDepth = 60 * time.Second

// MinHistoryBeforeRestart is how much local+cross-safe history each chain
// must accumulate before we trigger a RestartInterop(wipeLogsDBs=true).
// Strictly larger than BackfillDepth so backfill has a non-empty range to
// reingest — otherwise the coverage assertion is vacuous.
const MinHistoryBeforeRestart = BackfillDepth + 30*time.Second

// NewTestSystem builds a two-L2 interop system with interop active at genesis,
// time-travel enabled, and the supernode configured to run log backfill
// with BackfillDepth on every (re)start of its interop activity.
func NewTestSystem(t devtest.T) *presets.TwoL2SupernodeInterop {
	return presets.NewTwoL2SupernodeInterop(t, 0,
		presets.WithTimeTravelEnabled(),
		presets.WithInteropLogBackfillDepth(BackfillDepth),
	)
}

// AwaitHistoryAtLeast blocks until both L2 chains' local-safe and
// cross-safe timestamps have advanced at least `age` past genesis.
// Intended to be called before wiping the logs DB so the subsequent
// backfill has a meaningful range to reingest.
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
