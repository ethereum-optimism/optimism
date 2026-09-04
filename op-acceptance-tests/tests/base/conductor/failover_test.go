package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
)

// TestFailoverOnActiveSequencerFailure verifies the HA failover guarantee:
// when the active sequencer's node dies, its conductor detects the failure via
// health monitoring and hands leadership to a healthy cluster member, whose
// sequencer takes over block production.
func TestFailoverOnActiveSequencerFailure(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t, presets.WithConductorFastHealthChecks())

	failedLeader := sys.Conductors.AwaitOneActiveSequencer()
	preFailureHead := failedLeader.Sequencer().HeadBlockRef(safety.LocalUnsafe)
	failedLeader.Sequencer().Stop()

	failedLeader.AwaitNotLeader()
	survivors := sys.Conductors.Without(failedLeader)
	newLeader := survivors.AwaitOneActiveSequencer()
	t.Require().Same(newLeader, sys.Conductors.AwaitLeader())

	// Cross the pre-failure head, then require fresh production after takeover.
	// This distinguishes a genuinely live replacement from one that only
	// reports SequencerActive while its chain is stalled.
	newLeader.Sequencer().Reached(safety.LocalUnsafe, preFailureHead.Number+1, 30)
	newLeader.Sequencer().AdvancedUnsafe(2, 30)

	// The surviving follower must receive the replacement sequencer's new
	// unsafe blocks and remain on the same canonical chain.
	for _, follower := range survivors.Without(newLeader) {
		follower.Sequencer().InSync(newLeader.Sequencer(), safety.LocalUnsafe, 30)
	}
}
