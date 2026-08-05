package reorgs

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"

	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type checksFunc func(t devtest.T, sys *presets.TwoL2SupernodeInterop)

func TestL2ReorgAfterL1Reorg(gt *testing.T) {
	gt.Run("unsafe reorg", func(gt *testing.T) {
		var crossSafeRef, localSafeRef, unsafeRef eth.BlockID
		// Capture refs that must remain canonical before the manual L1 sequencing loop runs,
		// so their L1 origins are in the pre-divergence prefix.
		preEarly := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
			ss := sys.L2ACL.SyncStatus()
			crossSafeRef = ss.SafeL2.ID()
			localSafeRef = ss.LocalSafeL2.ID()
		}
		pre := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
			ss := sys.L2ACL.SyncStatus()
			unsafeRef = ss.UnsafeL2.ID()
		}
		post := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
			require.True(t, sys.L2ELA.IsCanonical(crossSafeRef), "Previous cross-safe block should still be canonical")
			require.True(t, sys.L2ELA.IsCanonical(localSafeRef), "Previous local-safe block should still be canonical")
			require.False(t, sys.L2ELA.IsCanonical(unsafeRef), "Previous unsafe block should have been reorged")
		}
		testL2ReorgAfterL1Reorg(gt, 3, preEarly, pre, post)
	})

	gt.Run("unsafe, local-safe, cross-unsafe, cross-safe reorgs", func(gt *testing.T) {
		var crossSafeA, crossUnsafeA, localSafeA, unsafeA eth.L2BlockRef
		var crossSafeB, crossUnsafeB, localSafeB, unsafeB eth.L2BlockRef
		var verifiedTimestamp uint64
		var staleSuperRoot eth.Bytes32
		pre := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
			ssA := sys.L2ACL.SyncStatus()
			ssB := sys.L2BCL.SyncStatus()
			crossUnsafeA, crossSafeA = ssA.CrossUnsafeL2, ssA.SafeL2
			localSafeA, unsafeA = ssA.LocalSafeL2, ssA.UnsafeL2
			crossUnsafeB, crossSafeB = ssB.CrossUnsafeL2, ssB.SafeL2
			localSafeB, unsafeB = ssB.LocalSafeL2, ssB.UnsafeL2

			verifiedTimestamp = min(crossSafeA.Time, crossSafeB.Time)
			sys.Supernode.AwaitValidatedTimestamp(verifiedTimestamp)
			staleSuperRoot = sys.Supernode.SuperRootAt(
				verifiedTimestamp, sys.L2A.ChainID(), sys.L2B.ChainID(),
			).Data.SuperRoot
		}
		post := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
			for chain, refs := range map[string]struct {
				el                                        *dsl.L2ELNode
				crossSafe, crossUnsafe, localSafe, unsafe eth.L2BlockRef
			}{
				"A": {sys.L2ELA, crossSafeA, crossUnsafeA, localSafeA, unsafeA},
				"B": {sys.L2ELB, crossSafeB, crossUnsafeB, localSafeB, unsafeB},
			} {
				require.False(t, refs.el.IsCanonical(refs.crossSafe.ID()), "chain %s previous cross-safe block should have been reorged", chain)
				require.False(t, refs.el.IsCanonical(refs.crossUnsafe.ID()), "chain %s previous cross-unsafe block should have been reorged", chain)
				require.False(t, refs.el.IsCanonical(refs.localSafe.ID()), "chain %s previous local-safe block should have been reorged", chain)
				require.False(t, refs.el.IsCanonical(refs.unsafe.ID()), "chain %s previous unsafe block should have been reorged", chain)
			}

			replacementA := sys.L2ELA.BlockRefByNumber(crossSafeA.Number)
			replacementB := sys.L2ELB.BlockRefByNumber(crossSafeB.Number)
			require.NotEqual(t, replacementA.Hash, replacementB.Hash,
				"replacement lineages were contaminated across sibling chains")
			_, err := sys.L2ELA.Escape().L2EthClient().L2BlockRefByHash(t.Ctx(), replacementB.Hash)
			require.Error(t, err, "chain B replacement unexpectedly resolved in chain A's engine")
			_, err = sys.L2ELB.Escape().L2EthClient().L2BlockRefByHash(t.Ctx(), replacementA.Hash)
			require.Error(t, err, "chain A replacement unexpectedly resolved in chain B's engine")

			require.Eventually(t, func() bool {
				resp, err := sys.SuperNodeClient().SuperRootAtTimestamp(t.Ctx(), verifiedTimestamp)
				return err == nil && resp.Data != nil && resp.Data.SuperRoot != staleSuperRoot
			}, 2*time.Minute, time.Second,
				"verified super-root at the reorged timestamp remained stale")
			newRoot := sys.Supernode.SuperRootAt(
				verifiedTimestamp, sys.L2A.ChainID(), sys.L2B.ChainID(),
			)
			canonicalL1 := sys.L1EL.BlockRefByNumber(newRoot.Data.VerifiedRequiredL1.Number)
			require.Equal(t, canonicalL1.Hash, newRoot.Data.VerifiedRequiredL1.Hash,
				"replacement super-root retained an orphaned L1 dependency")
		}
		preEarly := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {}
		testL2ReorgAfterL1Reorg(gt, 10, preEarly, pre, post)
	})
}

// testL2ReorgAfterL1Reorg tests that the L2 chain reorgs after an L1 reorg, and takes n, number of blocks to reorg, as parameter
// for unsafe reorgs - n must be at least >= confDepth, which is 2 in our test deployments
// for cross-safe reorgs - n must be at least >= safe distance, which is 10 in our test deployments (set in
// op-e2e/e2eutils/geth/geth.go when initialising FakePoS)
// preEarlyChecks runs before the L1 CL is stopped, so refs captured there have L1 origins in
// the pre-divergence prefix (anything visible before Stop has L1Origin.Number <= T0, and the
// reorg's alternative chain branches at T0's child).
// preChecks runs after the manual L1 sequencing loop, so refs captured there can land in the
// to-be-reorged window.
// postChecks runs after the reorg has been recovered.
func testL2ReorgAfterL1Reorg(gt *testing.T, n int, preEarlyChecks, preChecks, postChecks checksFunc) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	sys.L1Network.WaitForBlock()

	// Build a stable cross-safe foundation before we stop the L1 CL and manually sequence.
	// This ensures the supernode has verified state that references canonical L1 blocks,
	// so after the reorg it doesn't need to rewind all the way back to genesis.
	dsl.CheckAll(t,
		sys.L2ACL.AdvancedFn(safety.CrossSafe, 20, 100),
		sys.L2BCL.AdvancedFn(safety.CrossSafe, 20, 100),
	)

	preEarlyChecks(t, sys)

	sys.L1CL.Stop()

	// sequence a few L1 and L2 blocks
	for range n + 1 {
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})

		sys.L2A.WaitForBlock()
		sys.L2A.WaitForBlock()
	}

	// select a divergence block to reorg from
	var divergence eth.L1BlockRef
	{
		tip := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		require.Greater(t, tip.Number, uint64(n), "n is larger than L1 tip, cannot reorg out block number `tip-n`")

		divergence = sys.L1EL.BlockRefByNumber(tip.Number - uint64(n))
	}

	// print the chains before sequencing an alternative L1 block
	sys.L2A.PrintChain()
	sys.L1Network.PrintChain()

	// pre reorg trigger validations and checks
	preChecks(t, sys)

	tipL2APreReorg := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
	tipL2BPreReorg := sys.L2ELB.BlockRefByLabel(eth.Unsafe)

	// reorg the L1 chain -- sequence an alternative L1 block from divergence block parent
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), divergence.ParentHash)

	// continue building on the alternative L1 chain
	sys.L1CL.Start()

	// confirm L1 reorged
	sys.L1EL.ReorgTriggered(divergence, 5)

	// Wait until both L2 cross-safe refs catch up to where they were before the reorg.
	// Use require.Eventually instead of sys.L2ACL.Reached because the supernode rewinds
	// one timestamp at a time after an L1 reorg, stopping and restarting VNs each cycle.
	// During these rewinds the CL RPC is temporarily unavailable, and Reached() would
	// fatally fail via require.NoError on the transient RPC error.
	require.Eventually(t, func() bool {
		ssA, err := sys.L2ACL.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			sys.Log.Info("chain A SyncStatus unavailable during rewind, retrying", "err", err)
			return false
		}
		ssB, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			sys.Log.Info("chain B SyncStatus unavailable during rewind, retrying", "err", err)
			return false
		}
		sys.Log.Info("waiting for both cross-safe heads to reach their pre-reorg tips",
			"chain_a_cross_safe", ssA.SafeL2.Number, "chain_a_target", tipL2APreReorg.Number,
			"chain_b_cross_safe", ssB.SafeL2.Number, "chain_b_target", tipL2BPreReorg.Number)
		return ssA.SafeL2.Number >= tipL2APreReorg.Number && ssB.SafeL2.Number >= tipL2BPreReorg.Number
	}, 10*time.Minute, 5*time.Second,
		"both L2 cross-safe heads should reach their pre-reorg tips A=%d B=%d",
		tipL2APreReorg.Number, tipL2BPreReorg.Number)

	// Test that neither latest unsafe head references a reorged L1 block.
	require.Eventually(t, func() bool {
		for chain, el := range map[string]*dsl.L2ELNode{"A": sys.L2ELA, "B": sys.L2ELB} {
			unsafe := el.BlockRefByLabel(eth.Unsafe)
			block, err := sys.L1EL.Escape().EthClient().InfoByNumber(ctx, unsafe.L1Origin.Number)
			if err != nil {
				sys.Log.Warn("failed to get L1 block info by number", "chain", chain, "number", unsafe.L1Origin.Number, "err", err)
				return false
			}
			if block.Hash() != unsafe.L1Origin.Hash {
				return false
			}
		}
		return true
	}, 120*time.Second, 7*time.Second, "both L2 unsafe heads should reference canonical L1 origins")

	// Confirm all recent L1Origin fields on both chains point to canonical blocks.
	require.Eventually(t, func() bool {
		for _, el := range []*dsl.L2ELNode{sys.L2ELA, sys.L2ELB} {
			ref := el.BlockRefByLabel(eth.Unsafe)
			if ref.L1Origin.Number < divergence.Number {
				return false
			}
			for i := ref.Number; i > 0 && ref.L1Origin.Number >= divergence.Number; i-- {
				var err error
				ref, err = el.Escape().L2EthClient().L2BlockRefByNumber(ctx, i)
				if err != nil || !sys.L1EL.IsCanonical(ref.L1Origin) {
					return false
				}
			}
		}
		return true
	}, 120*time.Second, 5*time.Second, "all recent L1 origins on both chains should be canonical")

	// post reorg test validations and checks
	postChecks(t, sys)
}
