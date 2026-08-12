package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
)

// TestConductorClusterStartsWithOneActiveSequencer verifies the core HA
// startup guarantee: a freshly formed conductor cluster elects a single Raft
// leader and only that leader's sequencer produces blocks; the follower
// sequencers stay stopped.
func TestConductorClusterStartsWithOneActiveSequencer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimalWithConductors(t)

	sys.Conductors.AwaitOneActiveSequencer()
}
