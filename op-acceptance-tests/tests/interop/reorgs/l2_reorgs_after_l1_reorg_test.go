package reorgs

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
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
		var crossSafeRef, crossUnsafeRef, localSafeRef, unsafeRef eth.BlockID
		pre := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
			ss := sys.L2ACL.SyncStatus()
			crossUnsafeRef = ss.CrossUnsafeL2.ID()
			crossSafeRef = ss.SafeL2.ID()
			localSafeRef = ss.LocalSafeL2.ID()
			unsafeRef = ss.UnsafeL2.ID()
		}
		post := func(t devtest.T, sys *presets.TwoL2SupernodeInterop) {
			require.False(t, sys.L2ELA.IsCanonical(crossSafeRef), "Previous cross-safe block should have been reorged")
			require.False(t, sys.L2ELA.IsCanonical(crossUnsafeRef), "Previous cross-unsafe block should have been reorged")
			require.False(t, sys.L2ELA.IsCanonical(localSafeRef), "Previous local-safe block should have been reorged")
			require.False(t, sys.L2ELA.IsCanonical(unsafeRef), "Previous unsafe block should have been reorged")
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
	t := devtest.ParallelT(gt)
	ctx := t.Ctx()

	sys := presets.NewTwoL2SupernodeInterop(t, 0)

	sys.L1Network.WaitForBlock()

	// Build a stable cross-safe foundation before we stop the L1 CL and manually sequence.
	// This ensures the supernode has verified state that references canonical L1 blocks,
	// so after the reorg it doesn't need to rewind all the way back to genesis.
	sys.L2ACL.Advanced(types.CrossSafe, 20, 100)

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

	tipL2_preReorg := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

	// reorg the L1 chain -- sequence an alternative L1 block from divergence block parent
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), divergence.ParentHash)

	// continue building on the alternative L1 chain
	sys.L1CL.Start()

	// confirm L1 reorged
	sys.L1EL.ReorgTriggered(divergence, 5)

	// Wait until L2 chain A cross-safe ref caught up to where it was before the reorg.
	// Use require.Eventually instead of sys.L2ACL.Reached because the supernode rewinds
	// one timestamp at a time after an L1 reorg, stopping and restarting VNs each cycle.
	// During these rewinds the CL RPC is temporarily unavailable, and Reached() would
	// fatally fail via require.NoError on the transient RPC error.
	require.Eventually(t, func() bool {
		ss, err := sys.L2ACL.Escape().RollupAPI().SyncStatus(ctx)
		if err != nil {
			sys.Log.Info("SyncStatus unavailable during rewind, retrying", "err", err)
			return false
		}
		sys.Log.Info("waiting for cross-safe to reach pre-reorg tip",
			"cross_safe", ss.SafeL2.Number, "target", tipL2_preReorg.Number)
		return ss.SafeL2.Number >= tipL2_preReorg.Number
	}, 10*time.Minute, 5*time.Second, "L2 chain A cross-safe should reach pre-reorg tip %d", tipL2_preReorg.Number)

	// test that latest chain A unsafe is not referencing a reorged L1 block (through the L1Origin field)
	require.Eventually(t, func() bool {
		unsafe := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

		block, err := sys.L1EL.Escape().EthClient().InfoByNumber(ctx, unsafe.L1Origin.Number)
		if err != nil {
			sys.Log.Warn("failed to get L1 block info by number", "number", unsafe.L1Origin.Number, "err", err)
			return false
		}

		sys.Log.Info("current unsafe ref", "tip", unsafe, "tip_origin", unsafe.L1Origin, "l1blk", eth.InfoToL1BlockRef(block))

		// print the chains so we have information to debug if the test fails
		sys.L2A.PrintChain()
		sys.L1Network.PrintChain()

		return block.Hash() == unsafe.L1Origin.Hash
	}, 120*time.Second, 7*time.Second, "L1 block origin hash should match hash of block on L1 at that number. If not, it means there was a reorg, and L2 blocks L1Origin field is referencing a reorged block.")

	// confirm all L1Origin fields point to canonical blocks
	require.Eventually(t, func() bool {
		ref := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		var err error

		// wait until L2 chains' L1Origin points to a L1 block after the one that was reorged
		if ref.L1Origin.Number < divergence.Number {
			return false
		}

		sys.Log.Info("L2 chain progressed, pointing to newer L1 block", "ref", ref, "ref_origin", ref.L1Origin, "divergence", divergence)

		for i := ref.Number; i > 0 && ref.L1Origin.Number >= divergence.Number; i-- {
			ref, err = sys.L2ELA.Escape().L2EthClient().L2BlockRefByNumber(ctx, i)
			if err != nil {
				return false
			}

			if !sys.L1EL.IsCanonical(ref.L1Origin) {
				return false
			}
		}

		return true
	}, 120*time.Second, 5*time.Second, "all L1Origin fields should point to canonical L1 blocks")

	// post reorg test validations and checks
	postChecks(t, sys)
}
