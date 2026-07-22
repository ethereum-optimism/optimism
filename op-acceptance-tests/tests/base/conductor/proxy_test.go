package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestConductorProxyServesOnlyLeader verifies the conductor RPC proxy
// contract that batchers and proposers rely on to follow the active
// sequencer: the leader's conductor forwards execution, rollup, and admin
// requests to its sequencer, while a follower's conductor refuses them.
func TestConductorProxyServesOnlyLeader(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.Leader()
	follower := sys.Conductors.Followers()[0]

	leader.VerifyProxyServesSequencerAPIs()
	follower.VerifyProxyRefusesSequencerAPIs()
}
