package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestFailoverOnActiveSequencerFailure verifies the HA failover guarantee:
// when the active sequencer's node dies, its conductor detects the failure via
// health monitoring and hands leadership to a healthy cluster member, whose
// sequencer takes over block production.
func TestFailoverOnActiveSequencerFailure(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalWithConductors(t, presets.WithConductorFastHealthChecks())

	failedLeader := sys.Conductors.AwaitOneActiveSequencer()
	failedLeader.Sequencer().Stop()

	failedLeader.AwaitNotLeader()
	survivors := sys.Conductors.Without(failedLeader)
	newLeader := survivors.AwaitOneActiveSequencer()
	t.Require().Same(newLeader, sys.Conductors.AwaitLeader())
	newLeader.AwaitSequencerHealthy()
}
