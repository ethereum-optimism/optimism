package startup_resync

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestSupernodeRestoresVerifiedCrossHeadsAfterRestart checkpoints a live
// supernode with verified history, restarts it without deleting its data, and
// proves that the first restored cross-safe reads never regress or change the
// canonical lineage. The stored super-root must also remain byte-for-byte
// stable while both chains continue advancing after restart.
func TestSupernodeRestoresVerifiedCrossHeadsAfterRestart(gt *testing.T) {
	t := devtest.SerialT(gt)
	require := t.Require()
	sys := presets.NewTwoL2SupernodeInterop(t, 0,
		presets.WithUniformL2BlockTimes(l2BlockTime),
	)

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.CrossSafe, 4, 90),
		sys.L2BCL.ReachedFn(safety.CrossSafe, 4, 90),
	)
	safeA := sys.L2ACL.HeadBlockRef(safety.CrossSafe)
	safeB := sys.L2BCL.HeadBlockRef(safety.CrossSafe)
	verifiedTimestamp := min(safeA.Time, safeB.Time)
	sys.Supernode.AwaitValidatedTimestamp(verifiedTimestamp)
	beforeRoot := sys.Supernode.SuperRootAt(
		verifiedTimestamp, sys.L2A.ChainID(), sys.L2B.ChainID(),
	)

	sys.Supernode.Stop()
	sys.Supernode.Start()

	// These are intentionally immediate, single-shot reads after Start. A
	// transient reset to genesis is itself a restart regression.
	restoredA := sys.L2ACL.HeadBlockRef(safety.CrossSafe)
	restoredB := sys.L2BCL.HeadBlockRef(safety.CrossSafe)
	require.GreaterOrEqual(restoredA.Number, safeA.Number,
		"chain A cross-safe head regressed across restart")
	require.GreaterOrEqual(restoredB.Number, safeB.Number,
		"chain B cross-safe head regressed across restart")
	require.Equal(safeA.Hash, sys.L2ELA.BlockRefByNumber(safeA.Number).Hash,
		"chain A rewrote its pre-restart cross-safe block")
	require.Equal(safeB.Hash, sys.L2ELB.BlockRefByNumber(safeB.Number).Hash,
		"chain B rewrote its pre-restart cross-safe block")
	afterRoot := sys.Supernode.SuperRootAt(
		verifiedTimestamp, sys.L2A.ChainID(), sys.L2B.ChainID(),
	)
	require.Equal(beforeRoot.Data.SuperRoot, afterRoot.Data.SuperRoot,
		"verified super-root changed across restart")

	dsl.CheckAll(t,
		sys.L2ACL.ReachedFn(safety.CrossSafe, safeA.Number+2, 120),
		sys.L2BCL.ReachedFn(safety.CrossSafe, safeB.Number+2, 120),
	)
	require.Equal(safeA.Hash, sys.L2ELA.BlockRefByNumber(safeA.Number).Hash,
		"chain A rewrote restored history while extending")
	require.Equal(safeB.Hash, sys.L2ELB.BlockRefByNumber(safeB.Number).Hash,
		"chain B rewrote restored history while extending")
	require.Equal(beforeRoot.Data.SuperRoot, sys.Supernode.SuperRootAt(
		verifiedTimestamp, sys.L2A.ChainID(), sys.L2B.ChainID(),
	).Data.SuperRoot, "verified super-root changed after post-restart extension")
}
