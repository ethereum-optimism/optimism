package reorgs

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/stack/match"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type checksFunc func(t devtest.T, sys *presets.SimpleInterop)

func TestL2ReorgAfterL1Reorg(gt *testing.T) {
	gt.Run("unsafe reorg", func(gt *testing.T) {
		var crossSafeRef, localSafeRef, unsafeRef eth.BlockID
		pre := func(t devtest.T, sys *presets.SimpleInterop) {
			ss := sys.Supervisor.FetchSyncStatus()
			crossSafeRef = ss.Chains[sys.L2ChainA.ChainID()].CrossSafe
			localSafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalSafe
			unsafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalUnsafe.ID()
		}
		post := func(t devtest.T, sys *presets.SimpleInterop) {
			require.True(t, sys.L2ELA.IsCanonical(crossSafeRef), "Previous cross-safe block should still be canonical")
			require.True(t, sys.L2ELA.IsCanonical(localSafeRef), "Previous local-safe block should still be canonical")
			require.False(t, sys.L2ELA.IsCanonical(unsafeRef), "Previous unsafe block should have been reorged")
		}
		testL2ReorgAfterL1Reorg(gt, 3, pre, post)
	})

	gt.Run("local-safe and cross-safe reorgs", func(gt *testing.T) {
		var crossSafeRef, localSafeRef, unsafeRef eth.BlockID
		pre := func(t devtest.T, sys *presets.SimpleInterop) {
			ss := sys.Supervisor.FetchSyncStatus()
			crossSafeRef = ss.Chains[sys.L2ChainA.ChainID()].CrossSafe
			localSafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalSafe
			unsafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalUnsafe.ID()
		}
		post := func(t devtest.T, sys *presets.SimpleInterop) {
			require.False(t, sys.L2ELA.IsCanonical(crossSafeRef), "Previous cross-safe block should have been reorged")
			require.False(t, sys.L2ELA.IsCanonical(localSafeRef), "Previous local-safe block should have been reorged")
			require.False(t, sys.L2ELA.IsCanonical(unsafeRef), "Previous unsafe block should have been reorged")
		}
		testL2ReorgAfterL1Reorg(gt, 10, pre, post)
	})
}

// testL2ReorgAfterL1Reorg tests that the L2 chain reorgs after an L1 reorg, and takes n, number of blocks to reorg, as parameter
// for unsafe reorgs - n must be at least >= confDepth, which is 2 in our test deployments
// for cross-safe reorgs - n must be at least >= safe distance, which is 10 in our test deployments (set in
// op-e2e/e2eutils/geth/geth.go when initialising FakePoS)
// pre- and post-checks are sanity checks to ensure that the blocks we expected to be reorged were indeed reorged or not
func testL2ReorgAfterL1Reorg(gt *testing.T, n int, preChecks, postChecks checksFunc) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSimpleInterop(t)
	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())

	cl := sys.L1Network.Escape().L1CLNode(match.FirstL1CL)

	sys.L1Network.WaitForBlock()

	sys.ControlPlane.FakePoSState(cl.ID(), stack.Stop)

	// sequence a few L1 and L2 blocks
	for range n + 1 {
		sequenceL1Block(t, ts, common.Hash{})

		sys.L2ChainA.WaitForBlock()
		sys.L2ChainA.WaitForBlock()
	}

	// select a divergence block to reorg from
	var divergence eth.L1BlockRef
	{
		tip := sys.L1EL.BlockRefByLabel(eth.Unsafe)
		require.Greater(t, tip.Number, uint64(n), "n is larger than L1 tip")

		divergence = sys.L1EL.BlockRefByNumber(tip.Number - uint64(n))
	}

	// print the chains before sequencing an alternative L1 block
	sys.L2ChainA.PrintChain()
	sys.L1Network.PrintChain()

	// pre reorg trigger validations and checks
	preChecks(t, sys)

	// reorg the L1 chain -- sequence an alternative L1 block from divergence block parent
	sequenceL1Block(t, ts, divergence.ParentHash)

	// continue building on the alternative L1 chain
	sys.ControlPlane.FakePoSState(cl.ID(), stack.Start)

	// confirm L1 reorged
	sys.L1EL.ReorgTriggered(divergence, 5)

	// test that latest chain A unsafe is not referencing a reorged L1 block (through the L1Origin field)
	require.Eventually(t, func() bool {
		unsafe := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

		block, err := sys.L1EL.Escape().EthClient().InfoByNumber(ctx, unsafe.L1Origin.Number)
		if err != nil {
			sys.Log.Warn("failed to get L1 block info by number", "number", unsafe.L1Origin.Number, "err", err)
			return false
		}

		sys.Log.Info("current unsafe ref", "tip", unsafe, "tip.L1Origin", unsafe.L1Origin, "L1 block", eth.InfoToL1BlockRef(block))

		// print the chains so we have information to debug if the test fails
		sys.L2ChainA.PrintChain()
		sys.L1Network.PrintChain()

		return block.Hash() == unsafe.L1Origin.Hash
	}, 120*time.Second, 15*time.Second, "L1 block origin hash should match hash of block on L1 at that number. If not, it means there was a reorg, and L2 blocks L1Origin field is referencing a reorged block.")

	// confirm all L1Origin fields point to canonical blocks
	{
		ref := sys.L2ELA.BlockRefByLabel(eth.Unsafe)
		var err error
		for i := ref.Number; i > 0 && ref.L1Origin.Number >= divergence.Number; i-- {
			ref, err = sys.L2ELA.Escape().L2EthClient().L2BlockRefByNumber(ctx, i)
			require.NoError(t, err, "Expected to get block ref by number", "number", i)

			require.True(t, sys.L1EL.IsCanonical(ref.L1Origin), "L1 block origin should be canonical")
		}
	}

	// post reorg test validations and checks
	postChecks(t, sys)
}

func sequenceL1Block(t devtest.T, ts apis.TestSequencerControlAPI, parent common.Hash) {
	require.NoError(t, ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent}))
	require.NoError(t, ts.Next(t.Ctx()))
}
