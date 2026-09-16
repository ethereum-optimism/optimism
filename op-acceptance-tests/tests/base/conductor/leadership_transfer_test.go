package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
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
		sys.Conductors.TransferLeadershipTo(leader, target)
		leader = target
	}
	sys.Conductors.TransferLeadershipTo(leader, initialLeader)
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

// TestUnsafeChainAdvancesAfterLeadershipTransfer verifies that block
// production survives a leadership transfer: the unsafe chain keeps advancing
// under the new leader and every sequencer node converges on the same
// canonical chain.
func TestUnsafeChainAdvancesAfterLeadershipTransfer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.AwaitLeader()
	target := sys.Conductors.Without(leader)[0]
	sys.Conductors.TransferLeadershipTo(leader, target)

	// A handful of blocks is enough to prove sustained production by the new
	// leader rather than a single lucky head advance.
	const attempts = 30
	advanceChecks := make([]dsl.CheckFunc, 0, len(sys.Conductors))
	for _, conductor := range sys.Conductors {
		advanceChecks = append(advanceChecks, conductor.Sequencer().AdvancedFn(safety.LocalUnsafe, 3, attempts))
	}
	dsl.CheckAll(t, advanceChecks...)

	// Check convergence only after every node has advanced. Running these
	// checks in parallel with the advancement checks would let them pass at the
	// shared pre-transfer head without verifying any newly produced block.
	ref := sys.Conductors[0].Sequencer()
	convergenceChecks := make([]dsl.CheckFunc, 0, len(sys.Conductors)-1)
	for _, conductor := range sys.Conductors[1:] {
		convergenceChecks = append(convergenceChecks, conductor.Sequencer().InSyncFn(ref, safety.LocalUnsafe, attempts))
	}
	dsl.CheckAll(t, convergenceChecks...)
}
