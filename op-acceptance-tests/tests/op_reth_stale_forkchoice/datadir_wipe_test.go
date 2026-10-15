package op_reth_stale_forkchoice

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

const (
	// finalityAttempts bounds, in 2s polls, how long the follower may take to finalize a block
	// above genesis, which takes L1 finality: 20 L1 blocks of 6s.
	finalityAttempts = 120
	// preWipeChainBlocks is how tall the chain is at least when the follower's datadir is wiped,
	// so syncing it again is a download of many blocks rather than of a handful.
	preWipeChainBlocks = 60
	// refinalizeAttempts bounds, in 2s polls, how long the follower may take to be finalized
	// again where it was: after its reset a consensus client finalizes only blocks it derived
	// since, so this takes about two L1 finality delays (kona-node took 225s and 282s).
	refinalizeAttempts = 210
	// resyncFromScratchAttempts bounds, in 2s polls, how long the follower may take to sync the
	// whole chain again from an empty datadir and to be safe and finalized where it was.
	resyncFromScratchAttempts = 120
)

// TestFollowerResyncsFromScratchUnderConsensusClientThatKeptItsHeads wipes the follower's
// op-reth datadir while its consensus client keeps running, so the first forkchoice updates the
// empty node receives name a head, a safe block and a finalized block it has not downloaded, and
// requires the follower to sync the chain again from genesis: to hold the sequencer's unsafe
// head, and to be safe and finalized at least where it was before the wipe.
func TestFollowerResyncsFromScratchUnderConsensusClientThatKeptItsHeads(gt *testing.T) {
	t := devtest.ParallelT(gt)
	for _, c := range syncModeMatrix() {
		t.Run(c.name(), func(t devtest.T) {
			t.Parallel()
			skipUnsupportedCell(t, c)
			sys := newStaleForkchoiceSystem(t, systemConfig{
				persistenceThreshold: defaultPersistenceThreshold,
				syncMode:             c.syncMode,
			})
			sys.Follower.EL.Reached(eth.Unsafe, preWipeChainBlocks, chainBuildUpAttempts)
			// The consensus client must hold a safe and a finalized block above genesis, which
			// an empty datadir does not have.
			sys.Follower.EL.Reached(eth.Finalized, 1, finalityAttempts)
			before := readEngineLabels(sys.Follower.EL)

			t.Logger().Info("Wiping the follower's datadir under its consensus client", "head", before.head, "safe", before.safe, "finalized", before.finalized)
			sys.Follower.EL.Stop()
			sys.Follower.EL.WipeDatadir()
			sys.Follower.EL.Start()
			sys.Follower.EL.WaitForOnline()
			t.Require().Less(sys.Follower.EL.BlockRefByLabel(eth.Unsafe).Number, before.head.Number, "the wipe must drop the follower's chain")
			sys.Follower.EL.PeerWith(sys.Sequencer.EL)

			sys.Follower.EL.InSync(sys.Sequencer.EL, safety.LocalUnsafe, resyncFromScratchAttempts)
			sys.Follower.EL.Reached(eth.Safe, before.safe.Number, resyncFromScratchAttempts)
			sys.Follower.EL.Reached(eth.Finalized, before.finalized.Number, refinalizeAttempts)
			sys.Follower.CL.InSync(sys.Sequencer.CL, safety.LocalSafe, resyncAttempts)
		})
	}
}
