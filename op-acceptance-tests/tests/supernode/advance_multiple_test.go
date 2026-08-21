package supernode

import (
	"net/url"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// TestTwoChainProgress confirms that two L2 chains advance when using a shared CL.
//
// It confirms:
//   - the two CLs are served by one supernode, on one socket, under different routes
//   - each of those routes answers for its own chain, asked of the node rather than assumed
//   - both chains gossip, and -- separately -- both chains derive
//
// The last two are not the same claim and the test does not let them stand in for one another.
// An unsafe head advancing says a payload arrived over gossip, which a supernode that derives
// nothing at all still does. Deriving is the local safe head moving, because that is set only by
// applying attributes read out of L1 batch data. A supernode that starts, peers and reports a
// healthy sync status while deriving on neither chain has to fail here, and so does one that
// derives on one chain and serves that same chain down both routes.
func TestTwoChainProgress(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewTwoL2Supernode(t)

	blockTime := sys.L2A.Escape().RollupConfig().BlockTime
	waitTime := time.Duration(blockTime+1) * time.Second

	// Check that the two CLs are on different chains
	require.NotEqual(t, sys.L2ACL.ChainID(), sys.L2BCL.ChainID())

	// One supernode, one socket: same scheme, host and port. Equality alone is a claim that gets
	// easier to satisfy the more the routing collapses -- two endpoints that are the same string
	// satisfy it perfectly -- so the routes are required to differ in the same breath. Without
	// that, everything below could be measuring one chain twice.
	uA, err := url.Parse(sys.L2ACL.Escape().UserRPC())
	require.NoError(t, err)
	uB, err := url.Parse(sys.L2BCL.Escape().UserRPC())
	require.NoError(t, err)
	require.Equal(t, uA.Scheme, uB.Scheme)
	require.Equal(t, uA.Host, uB.Host)
	require.Equal(t, uA.Port(), uB.Port())
	require.NotEqual(t, uA.Path, uB.Path,
		"the two chains must be different routes on the one socket, not the same endpoint twice")

	// Which chain a route answers for, asked of the node. ChainID() above is the id the devstack
	// configured, so it says what was intended, not what is being served; a supernode that
	// crossed its routes, or that serves one chain under both, agrees with it either way. The L2
	// genesis hash is the identity that cannot coincide between two chains.
	requireRouteServesChain(t, sys.L2ACL, sys.L2A, "A")
	requireRouteServesChain(t, sys.L2BCL, sys.L2B, "B")

	statusA := sys.L2ACL.SyncStatus()
	statusB := sys.L2BCL.SyncStatus()

	t.Logger().Info("initial sync status",
		"chainA_unsafe", statusA.UnsafeL2.Number,
		"chainA_safe", statusA.SafeL2.Number,
		"chainB_unsafe", statusB.UnsafeL2.Number,
		"chainB_safe", statusB.SafeL2.Number,
	)

	// Gossip. This says a payload reached each chain, and nothing about derivation.
	t.Require().Eventually(func() bool {
		newStatusA := sys.L2ACL.SyncStatus()
		newStatusB := sys.L2BCL.SyncStatus()
		advancedA := newStatusA.UnsafeL2.Number > statusA.UnsafeL2.Number
		advancedB := newStatusB.UnsafeL2.Number > statusB.UnsafeL2.Number
		return advancedA && advancedB
	}, 30*time.Second, waitTime, "chains should advance unsafe heads")

	// Derivation, measured from here rather than from the baseline above. The gossip wait can
	// take most of a minute, and a local safe head that moved during it would satisfy a check
	// against that older number on its first tick, reporting derivation that had already
	// finished. A fresh baseline makes both chains show the pipeline running now.
	deriveA := sys.L2ACL.SyncStatus().LocalSafeL2.Number
	deriveB := sys.L2BCL.SyncStatus().LocalSafeL2.Number
	t.Require().Eventually(func() bool {
		newStatusA := sys.L2ACL.SyncStatus()
		newStatusB := sys.L2BCL.SyncStatus()
		// Both are read every tick, rather than letting && skip the second: a failure should
		// report where each chain actually got to, not leave one of them unsampled.
		advancedA := newStatusA.LocalSafeL2.Number > deriveA
		advancedB := newStatusB.LocalSafeL2.Number > deriveB
		t.Logger().Info("waiting for local safe head progression",
			"chainA_local_safe", newStatusA.LocalSafeL2.Number, "chainA_advanced", advancedA,
			"chainB_local_safe", newStatusB.LocalSafeL2.Number, "chainB_advanced", advancedB,
		)
		return advancedA && advancedB
	}, 90*time.Second, waitTime, "both chains must advance a local safe head derived from L1")

	// Log final status
	finalStatusA := sys.L2ACL.SyncStatus()
	finalStatusB := sys.L2BCL.SyncStatus()
	t.Logger().Info("final sync status",
		"chainA_unsafe", finalStatusA.UnsafeL2.Number,
		"chainA_safe", finalStatusA.SafeL2.Number,
		"chainB_unsafe", finalStatusB.UnsafeL2.Number,
		"chainB_safe", finalStatusB.SafeL2.Number,
	)

	// Off genesis, stated separately from the advance above. A chain that never derived a block
	// is the failure a supernode hides best: it answers every call, reports a sync status, and
	// sits at the genesis it was configured with. Saying where the chain has to have got to,
	// rather than only that it moved, is what tells derivation from an inert chain.
	genesisA := sys.L2A.Escape().RollupConfig().Genesis.L2.Number
	genesisB := sys.L2B.Escape().RollupConfig().Genesis.L2.Number
	require.Greater(t, finalStatusA.LocalSafeL2.Number, genesisA,
		"chain A never derived a block past genesis")
	require.Greater(t, finalStatusB.LocalSafeL2.Number, genesisB,
		"chain B never derived a block past genesis")
}

// requireRouteServesChain asserts that the supernode route behind cl answers for network's chain,
// by asking it for the rollup config it is running and comparing the L2 genesis hash.
//
// Read off the node deliberately. Everything the devstack knows about which route belongs to
// which chain came from the configuration it generated, so only the node's own answer can
// disagree with it -- which is the whole point of asking.
func requireRouteServesChain(t devtest.T, cl *dsl.L2CLNode, network *dsl.L2Network, label string) {
	served, err := cl.Escape().RollupAPI().RollupConfig(t.Ctx())
	require.NoError(t, err, "read the rollup config chain %s's route serves", label)
	expected := network.Escape().RollupConfig().Genesis.L2.Hash
	// Guarded first: two zero hashes would compare equal and prove nothing.
	require.NotEqual(t, common.Hash{}, expected,
		"chain %s has no genesis hash to compare against", label)
	require.Equal(t, expected, served.Genesis.L2.Hash,
		"chain %s's route answered for another chain", label)
}
