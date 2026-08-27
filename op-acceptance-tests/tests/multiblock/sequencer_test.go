package multiblock

import (
	"testing"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TestSequencerBuildsMultipleBlocksPerTimestampUnderLoad asserts that once the chain allows it, a
// loaded sequencer puts more than one block on a single timestamp, and that the chain it produces
// obeys the multi-blocks rules.
func TestSequencerBuildsMultipleBlocksPerTimestampUnderLoad(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newMultiBlockSystem(t)

	sys.StartTransferLoad(t)
	activation := sys.L2Network.AwaitMultiBlockActivation(t)

	first, last := sys.SequencerEL.WaitForSiblingBlocks(2, siblingTimeout)
	t.Require().Equal(first.Time, last.Time, "the run must share one timestamp")

	// Check the rules over a stretch of loaded chain rather than over the group alone: a two-block
	// window holds no timestamp step and cannot reach the group limit. The range starts at the
	// activation block, which stands by itself on its own timestamp and so is a group boundary.
	head := sys.SequencerEL.WaitForHeadPastTime(eth.Unsafe, last.Time+loadObservationWindow, siblingTimeout)
	sys.SequencerEL.VerifyTimestampGroups(activation.Number, head.Number, sys.RollupConfig())
}

// TestIdleSequencerBuildsOneBlockPerBlockTime asserts that allowing block groups does not change an
// idle chain: with nothing to include, the sequencer still produces exactly one block per block
// time after the activation.
func TestIdleSequencerBuildsOneBlockPerBlockTime(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newMultiBlockSystem(t)

	activation := sys.L2Network.AwaitMultiBlockActivation(t)
	head := sys.SequencerEL.WaitForHeadPastTime(eth.Unsafe, activation.Time+idleObservationWindow, siblingTimeout)

	sys.SequencerEL.VerifyNoSiblingBlocks(activation.Number, head.Number, sys.RollupConfig())
}

// TestNoSiblingBlocksBeforeMultiBlockActivation asserts that the activation timestamp gates block
// groups: under the same load that produces groups later, every block up to and including the first
// block at the activation timestamp stands alone on its own timestamp.
func TestNoSiblingBlocksBeforeMultiBlockActivation(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := newMultiBlockSystem(t)

	sys.StartTransferLoad(t)
	// An idle chain trivially has no siblings, so the load has to reach the chain before the
	// activation for the pre-activation blocks to prove anything. Assert that rather than hope for
	// it: a block carrying several transfers is unambiguous evidence the spam arrived in time.
	loaded := sys.SequencerEL.WaitForGasUsed(eth.Unsafe, 5*params.TxGas, loadTimeout)
	activation := sys.L2Network.AwaitMultiBlockActivation(t)
	t.Require().Less(loaded.NumberU64(), activation.Number,
		"the load must reach the chain before activation, or the pre-activation blocks prove nothing")

	// Siblings are allowed strictly after the activation timestamp, so the activation block itself
	// is still the last block that has to stand alone.
	sys.SequencerEL.VerifyNoSiblingBlocks(0, activation.Number, sys.RollupConfig())
}
