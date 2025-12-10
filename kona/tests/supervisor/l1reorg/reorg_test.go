package reorgl1

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/op-rs/kona/supervisor/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checksFunc func(t devtest.T, sys *presets.SimpleInterop)

func TestL1Reorg(gt *testing.T) {
	gt.Run("unsafe reorg", func(gt *testing.T) {
		var crossSafeRef, localSafeRef, unsafeRef, reorgAfter eth.BlockID
		pre := func(t devtest.T, sys *presets.SimpleInterop) {
			ss := sys.Supervisor.FetchSyncStatus()

			crossSafeRef = ss.Chains[sys.L2ChainA.ChainID()].CrossSafe
			localSafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalSafe
			unsafeRef = ss.Chains[sys.L2ChainA.ChainID()].LocalUnsafe.ID()
			gt.Logf("Pre:: CrossSafe: %s, LocalSafe: %s, Unsafe: %s", crossSafeRef, localSafeRef, unsafeRef)

			// Calculate the divergent block
			blockRef, err := sys.Supervisor.Escape().QueryAPI().CrossDerivedToSource(t.Ctx(), sys.L2ChainA.ChainID(), localSafeRef)
			assert.Nil(gt, err, "Failed to query cross derived to source")
			reorgAfter = blockRef.ID()
		}
		post := func(t devtest.T, sys *presets.SimpleInterop) {
			require.True(t, sys.L2ELA.IsCanonical(crossSafeRef), "Previous cross-safe block should still be canonical")
			require.True(t, sys.L2ELA.IsCanonical(localSafeRef), "Previous local-safe block should still be canonical")
			require.False(t, sys.L2ELA.IsCanonical(unsafeRef), "Previous unsafe block should have been reorged")
		}
		testL2ReorgAfterL1Reorg(gt, &reorgAfter, pre, post)
	})
}

func testL2ReorgAfterL1Reorg(gt *testing.T, reorgAfter *eth.BlockID, preChecks, postChecks checksFunc) {
	t := devtest.SerialT(gt)
	ctx := t.Ctx()

	sys := presets.NewSimpleInterop(t)
	trm := utils.NewTestReorgManager(t)

	sys.L1Network.WaitForBlock()

	trm.StopL1CL()

	// sequence some l1 blocks initially
	for range 10 {
		trm.GetBlockBuilder().BuildBlock(ctx, nil)
		time.Sleep(5 * time.Second)
	}

	// pre reorg trigger validations and checks
	preChecks(t, sys)

	tip := sys.L1EL.BlockRefByLabel(eth.Unsafe).Number

	// create at least 5 blocks after the divergence point
	for tip-reorgAfter.Number < 5 {
		trm.GetBlockBuilder().BuildBlock(ctx, nil)
		time.Sleep(5 * time.Second)
		tip++
	}

	// Give some time so that those block are derived
	time.Sleep(time.Second * 10)

	divergence := sys.L1EL.BlockRefByNumber(reorgAfter.Number + 1)

	tipL2_preReorg := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

	// reorg the L1 chain -- sequence an alternative L1 block from divergence block parent
	t.Log("Building Divergence Chain from:", divergence)
	trm.GetBlockBuilder().BuildBlock(ctx, &divergence.ParentHash)

	t.Log("Stopping the batchers")
	sys.L2BatcherA.Stop()
	sys.L2BatcherB.Stop()

	t.Log("Starting the batchers again")
	sys.L2BatcherA.Start()
	sys.L2BatcherB.Start()

	// Give some time to batcher catch up
	time.Sleep(5 * time.Second)

	// Start sequential block building
	trm.GetPOS().Start()

	// Wait sometime(5*5 = 25 at least) so that pos can create required
	time.Sleep(30 * time.Second)

	// confirm L1 reorged
	sys.L1EL.ReorgTriggered(divergence, 5)

	// wait until L2 chain A cross-safe ref caught up to where it was before the reorg
	sys.L2CLA.Reached(types.CrossSafe, tipL2_preReorg.Number, 100)

	// test that latest chain A unsafe is not referencing a reorged L1 block (through the L1Origin field)
	require.Eventually(t, func() bool {
		unsafe := sys.L2ELA.BlockRefByLabel(eth.Unsafe)

		block, err := sys.L1EL.Escape().EthClient().InfoByNumber(ctx, unsafe.L1Origin.Number)
		if err != nil {
			sys.Log.Warn("failed to get L1 block info by number", "number", unsafe.L1Origin.Number, "err", err)
			return false
		}

		sys.Log.Info("current unsafe ref", "tip", unsafe, "tip_origin", unsafe.L1Origin, "l1blk", eth.InfoToL1BlockRef(block))

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
