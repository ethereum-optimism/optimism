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

	initialLeader := sys.Conductors.AwaitOneActiveSequencer()
	leader := initialLeader
	for _, target := range sys.Conductors.Without(initialLeader) {
		leader.TransferLeadershipTo(target)
		leader = target
	}
	leader.TransferLeadershipTo(initialLeader)
}

// TestLeadershipTransferSelectsAnotherActiveSequencer verifies an untargeted
// transfer moves leadership and sequencing to another healthy voter.
func TestLeadershipTransferSelectsAnotherActiveSequencer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.AwaitOneActiveSequencer()
	next := leader.TransferLeadership(sys.Conductors)
	t.Require().NotSame(leader, next)
}
