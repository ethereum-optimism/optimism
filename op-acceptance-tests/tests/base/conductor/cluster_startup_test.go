package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestConductorClusterStartsWithOneActiveSequencer verifies the core HA
// startup guarantee: a freshly formed conductor cluster elects a single Raft
// leader and only that leader's sequencer produces blocks; the follower
// sequencers stay stopped.
func TestConductorClusterStartsWithOneActiveSequencer(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	sys.Conductors.VerifyOneActiveSequencer()
}
