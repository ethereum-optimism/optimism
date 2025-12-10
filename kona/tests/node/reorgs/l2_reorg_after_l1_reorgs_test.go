package reorgs

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	node_utils "github.com/op-rs/kona/node/utils"
	"github.com/stretchr/testify/require"
)

type checksFunc func(t devtest.T, sys *node_utils.MinimalWithTestSequencersPreset)

func TestL2ReorgAfterL1Reorg(gt *testing.T) {
	gt.Run("unsafe reorg", func(gt *testing.T) {
		var localSafeRef, unsafeRef []eth.L2BlockRef
		pre := func(t devtest.T, sys *node_utils.MinimalWithTestSequencersPreset) {
			for _, elNode := range sys.L2ELNodes() {
				localSafeRef = append(localSafeRef, elNode.BlockRefByLabel(eth.Safe))
				unsafeRef = append(unsafeRef, elNode.BlockRefByLabel(eth.Unsafe))
			}
		}
		post := func(t devtest.T, sys *node_utils.MinimalWithTestSequencersPreset) {
			for i, elNode := range sys.L2ELNodes() {
				require.True(t, elNode.IsCanonical(localSafeRef[i].ID()), "Previous local-safe block should still be canonical")
				require.False(t, elNode.IsCanonical(unsafeRef[i].ID()), "Previous unsafe block should have been reorged")
			}
		}
		testL2ReorgAfterL1Reorg(gt, 3, pre, post)
	})

	gt.Run("unsafe, local-safe, cross-unsafe, cross-safe reorgs", func(gt *testing.T) {
		var localSafeRef, unsafeRef []eth.L2BlockRef
		pre := func(t devtest.T, sys *node_utils.MinimalWithTestSequencersPreset) {
			for _, elNode := range sys.L2ELNodes() {
				localSafeRef = append(localSafeRef, elNode.BlockRefByLabel(eth.Safe))
				unsafeRef = append(unsafeRef, elNode.BlockRefByLabel(eth.Unsafe))
			}
		}
		post := func(t devtest.T, sys *node_utils.MinimalWithTestSequencersPreset) {
			for i, elNode := range sys.L2ELNodes() {
				require.False(t, elNode.IsCanonical(unsafeRef[i].ID()), "Previous unsafe block should have been reorged", "elNode", elNode.ID(), "unsafeRef", unsafeRef[i].ID())
				require.False(t, elNode.IsCanonical(localSafeRef[i].ID()), "Previous local-safe block should have been reorged", "elNode", elNode.ID(), "localSafeRef", localSafeRef[i].ID())
			}
		}
		testL2ReorgAfterL1Reorg(gt, 20, pre, post)
	})
}

// testL2ReorgAfterL1Reorg tests that the L2 chain reorgs after an L1 reorg, and takes n, number of blocks to reorg, as parameter
// for unsafe reorgs - n must be at least >= confDepth, which is 2 in our test deployments
// for cross-safe reorgs - n must be at least >= safe distance, which is 10 in our test deployments
// pre- and post-checks are sanity checks to ensure that the blocks we expected to be reorged were indeed reorged or not
func testL2ReorgAfterL1Reorg(gt *testing.T, n int, preChecks, postChecks checksFunc) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := node_utils.NewMixedOpKonaWithTestSequencer(t)
	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())

	cl := sys.L1Network.Escape().L1CLNode(match.FirstL1CL)

	sys.L1Network.WaitForBlock()

	sys.ControlPlane.FakePoSState(cl.ID(), stack.Stop)

	// sequence a few L1 and L2 blocks
	for range n + 1 {
		sequenceL1Block(t, ts, common.Hash{})

		sys.L2Chain.WaitForBlock()
		sys.L2Chain.WaitForBlock()
	}

	// select a divergence block to reorg from
	var divergence eth.L1BlockRef
	{
		tip := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		require.Greater(t, tip.Number, uint64(n), "n is larger than L1 tip, cannot reorg out block number `tip-n`")

		divergence = sys.L1EL.BlockRefByNumber(tip.Number - uint64(n))
	}

	// print the chains before sequencing an alternative L1 block
	sys.L2Chain.PrintChain()
	sys.L1Network.PrintChain()

	// pre reorg trigger validations and checks
	preChecks(t, sys)

	tipL2_preReorg := sys.L2ELSequencerNodes()[0].BlockRefByLabel(eth.Unsafe)

	// reorg the L1 chain -- sequence an alternative L1 block from divergence block parent
	sequenceL1Block(t, ts, divergence.ParentHash)

	// continue building on the alternative L1 chain
	sys.ControlPlane.FakePoSState(cl.ID(), stack.Start)

	// confirm L1 reorged
	sys.L1EL.ReorgTriggered(divergence, 5)

	// wait until L2 chain cross-safe ref caught up to where it was before the reorg
	var waitFunc []dsl.CheckFunc
	for _, clNode := range sys.L2CLNodes() {
		waitFunc = append(waitFunc, clNode.ReachedFn(types.CrossSafe, tipL2_preReorg.Number, 200))
	}

	dsl.CheckAll(t, waitFunc...)

	// test that latest chain unsafe is not referencing a reorged L1 block (through the L1Origin field)
	require.Eventually(t, func() bool {
		for _, elNode := range sys.L2ELNodes() {
			unsafe := elNode.BlockRefByLabel(eth.Unsafe)

			block, err := sys.L1EL.Escape().EthClient().InfoByNumber(ctx, unsafe.L1Origin.Number)
			if err != nil {
				sys.Log.Warn("failed to get L1 block info by number", "number", unsafe.L1Origin.Number, "err", err)
				return false
			}

			sys.Log.Info("current unsafe ref", "tip", unsafe, "tip_origin", unsafe.L1Origin, "l1blk", eth.InfoToL1BlockRef(block))

			if block.Hash() != unsafe.L1Origin.Hash {
				return false
			}
		}

		return true
	}, 120*time.Second, 7*time.Second, "L1 block origin hash should match hash of block on L1 at that number. If not, it means there was a reorg, and L2 blocks L1Origin field is referencing a reorged block.")

	// confirm all L1Origin fields point to canonical blocks
	require.Eventually(t, func() bool {
		for _, elNode := range sys.L2ELNodes() {
			ref := elNode.BlockRefByLabel(eth.Unsafe)
			var err error

			// wait until L2 chain's L1Origin points to a L1 block after the one that was reorged
			if ref.L1Origin.Number < divergence.Number {
				return false
			}

			sys.Log.Info("L2 chain progressed, pointing to newer L1 block", "ref", ref, "ref_origin", ref.L1Origin, "divergence", divergence)

			for i := ref.Number; i > 0 && ref.L1Origin.Number >= divergence.Number; i-- {
				ref, err = elNode.Escape().L2EthClient().L2BlockRefByNumber(ctx, i)
				if err != nil {
					return false
				}

				if !sys.L1EL.IsCanonical(ref.L1Origin) {
					return false
				}
			}
		}

		return true
	}, 120*time.Second, 5*time.Second, "all L1Origin fields should point to canonical L1 blocks")

	// post reorg test validations and checks
	postChecks(t, sys)
}

func sequenceL1Block(t devtest.T, ts apis.TestSequencerControlAPI, parent common.Hash) {
	require.NoError(t, ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent}))
	require.NoError(t, ts.Next(t.Ctx()))
}
