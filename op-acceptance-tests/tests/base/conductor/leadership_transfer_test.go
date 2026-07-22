package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestLeadershipTransferMovesActiveSequencer verifies that manually
// transferring Raft leadership also moves the active sequencer: the old
// leader's sequencer stops sequencing and the transfer target's starts.
func TestLeadershipTransferMovesActiveSequencer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.Leader()
	target := sys.Conductors.Followers()[0]

	leader.TransferLeadershipTo(target)

	target.VerifySequencerActive()
	leader.VerifySequencerInactive()
}

// TestUnsafeChainAdvancesAfterLeadershipTransfer verifies that block
// production survives a leadership transfer: the unsafe chain keeps advancing
// under the new leader and every sequencer node converges on the same
// canonical chain.
func TestUnsafeChainAdvancesAfterLeadershipTransfer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.Leader()
	target := sys.Conductors.Followers()[0]
	leader.TransferLeadershipTo(target)

	// A handful of blocks is enough to prove sustained production by the new
	// leader rather than a single lucky head advance.
	sys.Conductors.VerifyUnsafeChainAdvancesAndConverges(3)
}
