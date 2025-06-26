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
	"github.com/stretchr/testify/assert"
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
			assert.False(t, sys.L2ELA.IsCanonical(unsafeRef), "Previous unsafe block should have been reorged: %s", unsafeRef)
			assert.True(t, sys.L2ELA.IsCanonical(localSafeRef), "Previous local-safe block should still be canonical: %s", localSafeRef)
			assert.True(t, sys.L2ELA.IsCanonical(crossSafeRef), "Previous cross-safe block should still be canonical: %s", crossSafeRef)
		}
		testL2ReorgAfterL1Reorg(gt, pre, post, eth.Unsafe)
	})

	gt.Run("unsafe, local-safe, cross-unsafe, cross-safe reorgs", func(gt *testing.T) {
		var crossSafeRef, crossUnsafeRef, localSafeRef, unsafeRef eth.BlockID
		pre := func(t devtest.T, sys *presets.SimpleInterop) {
			ss := sys.Supervisor.FetchSyncStatus()
			crossUnsafeRef = ss.Chains[sys.L2ChainA.ChainID()].CrossUnsafe
			crossSafeRef = ss.Chains[sys.L2ChainA.ChainID()].CrossSafe
			localSafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalSafe
			unsafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalUnsafe.ID()
		}
		post := func(t devtest.T, sys *presets.SimpleInterop) {
			assert.False(t, sys.L2ELA.IsCanonical(unsafeRef), "Previous unsafe block should have been reorged: %s", unsafeRef)
			assert.False(t, sys.L2ELA.IsCanonical(crossUnsafeRef), "Previous cross-unsafe block should have been reorged: %s", crossUnsafeRef)
			assert.False(t, sys.L2ELA.IsCanonical(localSafeRef), "Previous local-safe block should have been reorged: %s", localSafeRef)
			assert.False(t, sys.L2ELA.IsCanonical(crossSafeRef), "Previous cross-safe block should have been reorged: %s", crossSafeRef)
		}
		testL2ReorgAfterL1Reorg(gt, pre, post, eth.Safe)
	})
}

// testL2ReorgAfterL1Reorg tests that the L2 chain reorgs after an L1 reorg, and takes n, number of blocks to reorg, as parameter
// for unsafe reorgs - n must be at least >= confDepth, which is 2 in our test deployments
// for cross-safe reorgs - n must be at least >= safe distance, which is 10 in our test deployments (set in
// op-e2e/e2eutils/geth/geth.go when initialising FakePoS)
// pre- and post-checks are sanity checks to ensure that the blocks we expected to be reorged were indeed reorged or not
func testL2ReorgAfterL1Reorg(gt *testing.T, preChecks, postChecks checksFunc, reorgTarget eth.BlockLabel) {
	t := devtest.SerialT(gt)

	sys := presets.NewSimpleInterop(t)
	ts := sys.TestSequencer.Escape().ControlAPI(sys.L1Network.ChainID())

	cl := sys.L1Network.Escape().L1CLNode(match.FirstL1CL)

	sys.L1Network.WaitForBlock()

	sys.ControlPlane.FakePoSState(cl.ID(), stack.Stop)

	const divergenceNumber = 3

	// sequence until the divergence block is the origin of the reorg target
	var reorgRef eth.L2BlockRef
	for {
		sequenceL1Block(t, ts, common.Hash{})

		sys.L2ChainA.WaitForBlock()
		sys.L2ChainA.WaitForBlock()

		ref := sys.L2ELA.BlockRefByLabel(reorgTarget)
		logger := sys.Log.With("reorgTarget", reorgTarget, "ref", ref, "refL1Origin", ref.L1Origin)
		if ref.L1Origin.Number < divergenceNumber {
			logger.Warn("L1 origin has not reached divergence number yet")
			continue
		}

		reorgRef = ref
		logger.Info("L1 origin has reached divergence number")
		break
	}

	divergence := sys.L1EL.BlockRefByNumber(divergenceNumber)

	// pre reorg trigger validations and checks
	preChecks(t, sys)

	// tipL2_preReorg := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

	// reorg the L1 chain -- sequence an alternative L1 block from divergence block parent
	sequenceL1Block(t, ts, divergence.ParentHash)

	// continue building on the alternative L1 chain
	sys.ControlPlane.FakePoSState(cl.ID(), stack.Start)

	// confirm L1 reorged
	sys.L1EL.ReorgTriggered(divergence, 5)

	sys.Log.Info("L1 origin reorg confirmed")

	// wait until L2 chain A cross-safe ref caught up to where it was before the reorg
	// TODO: weird to wait for cross-safe to reach previous local safe, which will be much further in the future already
	// sys.L2CLA.Reached(types.CrossSafe, tipL2_preReorg.Number, 50)

	// wait for reorg of L1Origin
	require.Eventually(t, func() bool {
		ref := sys.L2ELA.BlockRefByLabel(reorgTarget)
		logger := sys.Log.With("reorgTarget", reorgTarget, "ref", ref, "refL1Origin", ref.L1Origin, "reorgRef", reorgRef, "reorgRefL1Origin", reorgRef.L1Origin)

		if ref.L1Origin.Number < reorgRef.L1Origin.Number {
			logger.Warn("L1 origin below divergence number")
			return false
		} else if ref.L1Origin.Number == reorgRef.L1Origin.Number {
			if ref.L1Origin.Hash == reorgRef.L1Origin.Hash {
				logger.Warn("L1 origin has not changed yet")
				return false
			}
			logger.Info("L1 origin has changed")
			return true
		}

		// now ahead of old reorg, need to search backwards...
		logger.Warn("L1 origin ahead of original, searching backwards")
		for {
			ref = sys.L2ELA.BlockRefByNumber(ref.Number - 1)
			logger = logger.With("ref", ref, "refL1Origin", ref.L1Origin)
			if ref.L1Origin.Number == reorgRef.L1Origin.Number {
				if ref.L1Origin.Hash == reorgRef.L1Origin.Hash {
					logger.Warn("L1 origin has not changed yet")
					return false
				}
				logger.Info("L1 origin has changed")
				return true
			}
		}
	}, 120*time.Second, 1*time.Second, "L1Origin should have changed due to reorg")

	// confirm all L1Origin fields point to canonical blocks
	/*
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
	*/

	// post reorg test validations and checks
	postChecks(t, sys)
}

func sequenceL1Block(t devtest.T, ts apis.TestSequencerControlAPI, parent common.Hash) {
	require.NoError(t, ts.New(t.Ctx(), seqtypes.BuildOpts{Parent: parent}))
	require.NoError(t, ts.Next(t.Ctx()))
}
