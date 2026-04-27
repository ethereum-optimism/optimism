// Package happy contains the happy-path acceptance test for interop log
// backfill. It lives in its own package (rather than a single file) so it
// runs in its own test binary, isolated from the retry-path test.
package happy

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/supernode/interop/backfill"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
)

// TestSupernodeLogBackfill_HappyPath exercises the happy path:
//
//  1. Bring up a two-L2 supernode with interop enabled at genesis.
//  2. Let both chains accumulate more than BackfillDepth of local+cross-safe
//     history.
//  3. Hot-restart only the interop activity, wiping its on-disk logs DBs.
//     Because every other component (chain containers, virtual nodes, RPC
//     server, other activities) keeps running, the replacement activity is
//     guaranteed to see ready VNs when it starts backfilling.
//  4. Assert that the logs DBs now span [T_lo, localSafe] for each chain.
//     This is the strongest evidence that backfill actually did work: every
//     block in the DB was sealed by backfill, because the disk was empty.
func TestSupernodeLogBackfill_HappyPath(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := backfill.NewTestSystem(t)

	sys.Supernode.AwaitBackfillCompleted()
	backfill.AwaitHistoryAtLeast(t, sys, backfill.MinHistoryBeforeRestart)

	sys.Supernode.RestartInterop(true)
	sys.Supernode.AwaitBackfillCompleted()

	t.Require().GreaterOrEqual(sys.Supernode.BackfillAttempts(), int32(1),
		"post-restart backfill should run at least once")

	sys.Supernode.AssertBackfillCovers(backfill.BackfillDepth,
		sys.L2A.Escape().RollupConfig().BlockTime,
		sys.L2A.ChainID(), sys.L2B.ChainID())
}
