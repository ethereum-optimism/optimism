package multiblock

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestVerifierFollowsMultiBlocksOverP2P asserts that a verifier picks up a block group from gossip:
// its unsafe chain holds the sequencer's sibling blocks, hash for hash.
func TestVerifierFollowsMultiBlocksOverP2P(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newMultiBlockSystem(t)

	sys.StartTransferLoad(t)
	sys.L2Network.AwaitMultiBlockActivation(t)
	// Take L1 away as a source: with the batcher running, a verifier whose gossip is broken still
	// reaches the group by deriving it, and the test would pass on the strength of the one path it
	// does not claim to cover.
	sys.L2Batcher.Stop()
	first, last := sys.SequencerEL.WaitForSiblingBlocks(2, siblingTimeout)

	sys.VerifierEL.VerifyMatchesChain(sys.SequencerEL, eth.Unsafe, first.Number, last.Number, p2pTimeout)
}

// TestVerifierDerivesMultiBlocksFromBatcher asserts that a block group survives the round trip
// through L1: with the batcher held back until the group exists, it then submits a backlog against
// spans of only maxBlocksPerSpanBatch blocks, and the verifier's safe chain still comes out
// hash-identical to the sequencer's.
func TestVerifierDerivesMultiBlocksFromBatcher(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newMultiBlockSystem(t)

	sys.StartTransferLoad(t)
	sys.L2Network.AwaitMultiBlockActivation(t)
	sys.L2Batcher.Stop()
	first, last := sys.SequencerEL.WaitForSiblingBlocks(2, siblingTimeout)
	sys.L2Batcher.Start()

	sys.VerifierEL.VerifyMatchesChain(sys.SequencerEL, eth.Safe, first.Number, last.Number, safeHeadTimeout)
}

// TestSafeHeadAdvancesAcrossMultiBlockActivation asserts that derivation carries on through the
// batcher's switch from span batch v1 to v2: the verifier's safe head crosses the activation
// timestamp on a chain whose blocks before and after the switch both become safe.
func TestSafeHeadAdvancesAcrossMultiBlockActivation(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newMultiBlockSystem(t)

	sys.StartTransferLoad(t)
	activation := sys.L2Network.AwaitMultiBlockActivation(t)
	t.Require().Greater(activation.Number, uint64(1),
		"multi-blocks must activate mid-chain so the batcher has v1 spans to switch away from")

	safe := sys.VerifierEL.WaitForHeadPastTime(eth.Safe, activation.Time, safeHeadTimeout)
	sys.VerifierEL.VerifyMatchesChain(sys.SequencerEL, eth.Safe, activation.Number-1, safe.Number, safeHeadTimeout)
}
