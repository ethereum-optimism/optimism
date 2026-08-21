package conductor

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// TestConductorProxyServesOnlyLeader verifies the conductor RPC proxy
// contract that batchers and proposers rely on to follow the active
// sequencer: the leader's conductor forwards execution, rollup, and admin
// requests to its sequencer, while a follower's conductor refuses them.
func TestConductorProxyServesOnlyLeader(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.AwaitOneActiveSequencer()
	follower := sys.Conductors.Without(leader)[0]

	// The preset's batcher talks only to conductor proxy endpoints. Require it
	// to publish blocks produced after this point and advance the safe chain.
	targetSafe := leader.Sequencer().HeadBlockRef(safety.LocalUnsafe).Number + 2
	leader.Sequencer().Reached(safety.LocalUnsafe, targetSafe, 30)
	leader.Sequencer().Reached(safety.LocalSafe, targetSafe, 60)

	var active bool
	var block map[string]any
	probes := []struct {
		method     string
		result     any
		args       func() []any
		eventually bool
		check      func()
	}{
		{
			method: "admin_sequencerActive",
			result: &active,
			check: func() {
				t.Require().True(active, "expected leader's sequencer to be active")
			},
		},
		{
			method: "optimism_syncStatus",
			result: new(json.RawMessage),
		},
		{
			method: "eth_getBlockByNumber",
			result: &block,
			args:   func() []any { return []any{"latest", false} },
			check: func() {
				t.Require().Contains(block, "number")
			},
		},
		{
			method:     "optimism_outputAtBlock",
			result:     new(json.RawMessage),
			args:       func() []any { return []any{block["number"]} },
			eventually: true,
		},
		{
			method: "optimism_rollupConfig",
			result: new(json.RawMessage),
		},
	}
	for _, probe := range probes {
		var args []any
		if probe.args != nil {
			args = probe.args()
		}

		callLeader := func() error {
			return leader.CallProxy(probe.result, probe.method, args...)
		}
		var err error
		if probe.eventually {
			err = retry.Do0(t.Ctx(), 120, retry.Fixed(500*time.Millisecond), callLeader)
		} else {
			err = callLeader()
		}
		t.Require().NoErrorf(err,
			"expected conductor %s to proxy %s to its sequencer", leader, probe.method)
		if probe.check != nil {
			probe.check()
		}

		var followerResult json.RawMessage
		err = follower.CallProxy(&followerResult, probe.method, args...)
		t.Require().ErrorContainsf(err, "refusing to proxy request to non-leader sequencer",
			"expected conductor %s to refuse proxying %s", follower, probe.method)
	}
}
