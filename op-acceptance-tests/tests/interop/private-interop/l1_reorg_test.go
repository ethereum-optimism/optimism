package privateinterop

import (
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum/go-ethereum/common"
)

func TestPrivatePublicationResumesAfterL1Reorg(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithDeployerOptions(sysgo.WithSequencingWindow(10)),
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck()))

	sys.L2BCL.Advanced(safety.CrossSafe, 20, 120)
	finalized := sys.L2ELB.BlockRefByLabel(eth.Finalized)
	sys.L1CL.Stop()
	parent := sys.L1EL.BlockRefByLabel(eth.Unsafe)

	// Freeze L1 finality while building a suffix whose origins will be removed.
	for range 4 {
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
		sys.L2B.WaitForBlock()
		sys.L2B.WaitForBlock()
	}
	divergence := sys.L1EL.BlockRefByNumber(parent.Number + 1)
	private := sys.L2ELB.WaitForL1Origin(divergence.Number)

	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), parent.Hash)
	sys.L1CL.Start()
	sys.L1EL.ReorgTriggered(divergence, 10)
	sys.L2BCL.Reached(safety.CrossSafe, private.Number+1, 180)
	sys.L2ELB.VerifyReorgRecovery(sys.L1EL, private)
	t.Require().True(sys.L2ELB.IsCanonical(finalized.ID()), "private finalized history must survive recovery")

	// A newly submitted private transfer must become claim-backed safe again.
	// Rebuilding an unsafe tip alone does not demonstrate publication recovery.
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	transfer := alice.Transfer(common.Address{0xee}, eth.OneGWei)
	included, err := transfer.Included.Eval(t.Ctx())
	t.Require().NoError(err)
	sys.L2BCL.ReachedRef(safety.CrossSafe, eth.BlockID{
		Hash: included.BlockHash, Number: bigs.Uint64Strict(included.BlockNumber),
	}, 180)
}

// Losing publication must revoke a private checkpoint even when none of the
// private blocks' L1 origins changed. The LightCL cannot detect this from its
// origin ancestry alone: it must consume the supernode's claimed-follow retreat.
func TestPrivatePublicationReorgWithCanonicalOrigins(gt *testing.T) {
	t := devtest.SerialT(gt)
	sys := presets.NewTwoL2SupernodeLightSequencerInterop(t, 0,
		presets.WithPrivateInteropChain(sysgo.WithoutRenderingInvariantCheck()))
	require := t.Require()
	sys.L2BCL.Advanced(safety.CrossSafe, 20, 120)
	finalized := sys.L2ELB.BlockRefByLabel(eth.Finalized)
	sys.L2BatcherB.Stop()
	sys.L2BatcherA.Stop()
	sys.L2ACL.StopSequencer()
	sys.L1CL.Stop()
	forkParent := sys.L1EL.BlockRefByLabel(eth.Unsafe)
	start := sys.L2ELB.BlockRefByLabel(eth.Unsafe)
	// Produce at normal wall-clock speed while L1 is frozen: all new origins
	// remain at or below the fork point, and gossip never sees future blocks.
	sys.L2BCL.Reached(safety.LocalUnsafe, start.Number+24, 90)
	privateTip := sys.L2ELB.BlockRefByHash(sys.L2BCL.StopSequencer())
	privateBlocks := make([]eth.L2BlockRef, 0, privateTip.Number-start.Number)
	for number := start.Number + 1; number <= privateTip.Number; number++ {
		ref := sys.L2ELB.BlockRefByNumber(number)
		require.LessOrEqual(ref.L1Origin.Number, forkParent.Number)
		privateBlocks = append(privateBlocks, ref)
	}

	// Prepare a longer replacement branch before any new publication exists.
	// Reorged L1 transactions return to the mempool; building this branch after
	// publication would silently include the very claims we intend to remove.
	// Keep the verifier offline during branch preparation so it cannot traverse
	// the temporary longer branch before the publication fork is installed.
	sys.Supernode.Stop()
	for range 9 {
		sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
	}
	replacementParent := sys.L1EL.BlockRefByNumber(forkParent.Number + 8)
	// Rebuilding the cached child changes its hash and starts the publication fork.
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), forkParent.Hash)
	removed := sys.L1EL.BlockRefByNumber(forkParent.Number + 1)
	sys.Supernode.Start()
	sys.L2BatcherB.Start()
	var published eth.L2BlockRef
	mined := 0
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		if err == nil && status.LocalSafeL2.Number >= start.Number+12 {
			published = status.LocalSafeL2
			return true
		}
		// Keep the fork point unfinalized (the manual L1 builder finalizes
		// twenty blocks behind). Poll the CL without extending beyond this cap.
		if mined < 6 {
			sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), common.Hash{})
			mined++
		}
		return false
	}, 90*time.Second, 2*time.Second, "new private blocks must be published on the removable L1 suffix")
	sys.L2BatcherB.Stop()
	require.LessOrEqual(published.Number, privateBlocks[len(privateBlocks)-1].Number)

	// Rebuild the cached ninth block on the prepared branch. It is longer than
	// the publication fork and contains no transactions published after the fork.
	sys.TestSequencer.SequenceBlock(t, sys.L1Network.ChainID(), replacementParent.Hash)
	sys.L1EL.ReorgTriggered(removed, 10)
	for _, ref := range privateBlocks {
		origin := sys.L1EL.BlockRefByNumber(ref.L1Origin.Number)
		require.Equal(ref.L1Origin, origin.ID(), "the reorg must leave every private origin canonical")
	}
	require.Eventually(func() bool {
		status, err := sys.L2BCL.Escape().RollupAPI().SyncStatus(t.Ctx())
		return err == nil && status.LocalSafeL2.Number < published.Number &&
			status.UnsafeL2.Number < published.Number
	}, 90*time.Second, time.Second, "claim loss must rewind private execution despite canonical L1 origins")
	require.True(sys.L2ELB.IsCanonical(finalized.ID()), "private finalized history must survive claim loss")

	sys.L1CL.Start()
	sys.L2ACL.StartSequencer()
	sys.L2BatcherA.Start()
	sys.L2BCL.StartSequencer()
	sys.L2BatcherB.Start()
	// Recovery may still replace newly sequenced unsafe blocks while the
	// replacement branch is being reconciled. Wait for publication beyond the
	// discarded interval before asserting preservation of a fresh transaction.
	sys.L2BCL.Reached(safety.CrossSafe, privateTip.Number+12, 180)
	alice := sys.FunderB.NewFundedEOA(eth.OneEther)
	transfer := alice.Transfer(common.Address{0xed}, eth.OneGWei)
	included, err := transfer.Included.Eval(t.Ctx())
	require.NoError(err)
	sys.L2BCL.ReachedRef(safety.CrossSafe, eth.BlockID{
		Hash: included.BlockHash, Number: bigs.Uint64Strict(included.BlockNumber),
	}, 180)
}
