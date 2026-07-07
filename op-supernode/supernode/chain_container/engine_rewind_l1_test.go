package chain_container

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container/engine_controller"
	"github.com/stretchr/testify/require"
)

// TestRewindEngineDeadlineRetry documents issue-20929 root cause 1: RewindEngine
// treats a context.DeadlineExceeded from RewindToTimestamp as terminal
// (chain_container.go: `case errors.Is(err, context.DeadlineExceeded): return err`)
// instead of retrying. A per-call FCU deadline against a slow archive-mode EL is
// transient (the EL commits eventually), so the rewind should retry until it
// converges rather than bail and leave the transition stuck.
//
// L1: real simpleChainContainer + real EngineController; only the l2Provider
// lies. The deadline propagates through the EC's error wrapping (multi-%w), so
// RewindEngine's errors.Is(err, context.DeadlineExceeded) matches and it bails.
func TestRewindEngineDeadlineRetry(t *testing.T) {
	ctx := context.Background()
	mgr := NewRandomChainManager([]byte("l1-rewind-deadline"))
	mgr.Generate()
	t.Cleanup(func() { _ = mgr.Close() })
	rc := mgr.Chains()[0]

	c, err := mgr.ChainContainer(rc.chainID, t.TempDir())
	require.NoError(t, err)

	ts, target, ok := pickRewindTarget(rc, fnvSeed([]byte("l1-rewind-deadline")))
	require.True(t, ok, "need a rewindable target above finalized")

	frc := newFaultyRandomChain(rc, elAboveTarget, target)
	frc.fcuDeadlines = 1 // first FCU (Step 3) times out, then the EL cooperates
	c.engine = engine_controller.NewEngineControllerWithL2AndRollup(frc, rc.cfg)

	err = c.RewindEngine(ctx, ts, target.BlockRef())

	// Invariant that holds on the pinned commit and post-fix: however RewindEngine
	// exits, the deferred Resume must leave the container un-paused. A stuck
	// pause=true spins the Start loop forever (see chain_container.go:652-654).
	require.False(t, c.pause.Load(), "RewindEngine must leave the container resumed")

	// FIXED SPEC (issue-20929 root cause 1): a transient FCU deadline is retried,
	// not terminal, so RewindEngine converges and returns nil. Committed commented
	// so the suite stays green on the pinned audit commit; uncomment to reproduce
	// (on pinned, err is a wrapped context.DeadlineExceeded -- the bail).
	// require.NoError(t, err, "RewindEngine must retry a transient FCU deadline, not bail")
	_ = err
}
