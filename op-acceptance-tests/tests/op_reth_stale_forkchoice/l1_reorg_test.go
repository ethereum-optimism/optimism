package op_reth_stale_forkchoice

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

const (
	// l1AdvanceBlocks bounds how many L1 blocks the test sequencer builds, one every 2s, before
	// the follower's safe head must derive from an L1 block past the one it took over at.
	l1AdvanceBlocks = 60
	// l1ReorgAttempts bounds, in 7s polls, how long the L1 node may take to switch to the block
	// the test sequencer built on the reorged block's parent.
	l1ReorgAttempts = 10
	// postReorgSafeBlocks is how far past its pre-reorg height the follower's safe head must get
	// again on the new L1 chain.
	postReorgSafeBlocks = 5
	// safeRecoveryAttempts bounds, in 2s polls, how long the follower may take after the L1 reorg
	// to replace its safe block and to be safe past the pre-reorg height again: the batcher must
	// notice the reorg and post again, and the follower's derivation must reset and catch up.
	safeRecoveryAttempts = 90
)

// TestFollowerSafeHeadAdvancesAgainAfterL1ReorgRewindsIt reorgs out the L1 block the follower's
// safe head derives from while the follower's consensus client runs, so the client resets and
// sends forkchoice updates whose safe hash moves back below the unsafe chain its engine still
// holds, and requires the follower to replace the safe block that derived from the reorged L1
// block, to derive the new L1 chain past where its safe head was, and to be in sync with the
// sequencer again at the safe and unsafe heads.
func TestFollowerSafeHeadAdvancesAgainAfterL1ReorgRewindsIt(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range syncModeMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupportedCell(t, c)
			sysgo.SkipOnKonaNode(t, "fails on develop (#18676): the kona-node sequencer resets its derivation on the L1 reorg "+
				"(`Derivation pipeline is being reset: L1 reorg detected`) but keeps building on the unsafe chain whose L1 origin was "+
				"reorged out, so nothing replaces that chain, every node rejects the old batches (`batch is for different L1 chain, "+
				"epoch hash does not match`) and the follower's safe block is never replaced: `operation failed permanently after 90 "+
				"attempts: expected head to reorg 0x3f26…d122:20, but got 0x3f26…d122:20`; the L1-reorg tests in sync/elsync/reorg "+
				"skip on kona-node for the same reason (#22275)")
			sys := newStaleForkchoiceSystem(t, systemConfig{
				persistenceThreshold: defaultPersistenceThreshold,
				syncMode:             c.syncMode,
				withTestSequencer:    true,
			})
			l1ChainID := sys.L1Network.ChainID()
			sys.L1Network.WaitForBlock()
			// Only the test sequencer builds L1 from here on, so the reorg below is the only fork.
			sys.L1CL.Stop()
			startL1 := sys.L1EL.BlockRefByLabel(eth.Unsafe)
			sys.TestSequencer.SequenceBlocksUntil(t, l1ChainID, l1AdvanceBlocks,
				sys.Follower.EL.L1OriginReachedFn(eth.Safe, startL1.Number+1, 1))
			safeBefore := sys.Follower.EL.BlockRefByLabel(eth.Safe)

			l1Origin := sys.L1EL.BlockRefByNumber(safeBefore.L1Origin.Number)
			t.Logger().Info("Reorging out the L1 block the follower's safe head derives from", "safe", safeBefore, "l1_origin", l1Origin)
			sys.TestSequencer.SequenceBlock(t, l1ChainID, l1Origin.ParentHash)
			sys.L1CL.Start()
			sys.L1EL.ReorgTriggered(l1Origin, l1ReorgAttempts)
			l1After := sys.L1EL.BlockRefByNumber(l1Origin.Number)
			t.Logger().Info("L1 reorged", "before", l1Origin, "after", l1After)

			sys.Follower.EL.ReorgTriggered(safeBefore, safeRecoveryAttempts)
			sys.Follower.EL.WaitL1OriginHash(eth.Safe, l1After.ID(), safeRecoveryAttempts)
			sys.Follower.EL.Reached(eth.Safe, safeBefore.Number+postReorgSafeBlocks, safeRecoveryAttempts)
			sys.Follower.EL.InSync(sys.Sequencer.EL, safety.LocalSafe, resyncAttempts)
			sys.Follower.EL.InSync(sys.Sequencer.EL, safety.LocalUnsafe, resyncAttempts)
		})
	}
}
