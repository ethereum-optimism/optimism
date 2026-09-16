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

// TestDisasterRecoveryLeaderOverride verifies the disaster-recovery escape
// hatch: when the conductor cluster loses Raft quorum no sequencer may start
// through the normal leadership path, but an operator can override leadership
// on a surviving node and force it to resume sequencing.
func TestDisasterRecoveryLeaderOverride(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.AwaitOneActiveSequencer()
	followers := sys.Conductors.Without(leader)
	survivor, casualty := followers[0], followers[1]

	// The disaster: two of the three conductors die, so the survivor can never
	// win an election, and the active sequencer's node dies with its
	// conductor, halting the chain. The casualty conductor goes first so the
	// still-live cluster cannot elect it in between.
	casualty.Stop()
	leader.Stop()
	leader.Sequencer().Stop()

	// Without Raft leadership the survivor's node refuses to sequence.
	unsafe := survivor.Sequencer().HeadBlockRef(safety.LocalUnsafe)
	err := survivor.Sequencer().StartSequencerAt(unsafe.Hash)
	t.Require().ErrorContains(err, "sequencer is not the leader")

	// The operator override forces the survivor into non-HA mode: its node
	// stops consulting the conductor and its conductor reports leadership to
	// downstream users (e.g. proxied batcher traffic).
	survivor.Sequencer().OverrideLeader()
	survivor.OverrideLeader(true)
	t.Require().True(survivor.LeaderOverridden())
	survivor.Sequencer().StartSequencer()

	// The chain is live again under the surviving sequencer, and the
	// overridden conductor serves admin, rollup, and execution traffic as if
	// it were the leader.
	survivor.Sequencer().AdvancedUnsafe(2, 30)

	var active bool
	t.Require().NoError(survivor.CallProxy(&active, "admin_sequencerActive"))
	t.Require().True(active, "expected overridden sequencer to be active")

	var syncStatus json.RawMessage
	t.Require().NoError(survivor.CallProxy(&syncStatus, "optimism_syncStatus"))

	var block map[string]any
	t.Require().NoError(survivor.CallProxy(&block, "eth_getBlockByNumber", "latest", false))
	t.Require().Contains(block, "number")
	blockNumber := block["number"]

	var output json.RawMessage
	err = retry.Do0(t.Ctx(), 120, retry.Fixed(500*time.Millisecond), func() error {
		return survivor.CallProxy(&output, "optimism_outputAtBlock", blockNumber)
	})
	t.Require().NoError(err, "expected output at the overridden sequencer's latest block")

	var rollupConfig json.RawMessage
	t.Require().NoError(survivor.CallProxy(&rollupConfig, "optimism_rollupConfig"))

	// Clearing the override makes the surviving non-Raft-leader refuse every
	// proxied API family again.
	survivor.OverrideLeader(false)
	t.Require().False(survivor.LeaderOverridden())
	probes := []struct {
		method string
		args   []any
	}{
		{method: "admin_sequencerActive"},
		{method: "optimism_syncStatus"},
		{method: "eth_getBlockByNumber", args: []any{"latest", false}},
		{method: "optimism_outputAtBlock", args: []any{blockNumber}},
		{method: "optimism_rollupConfig"},
	}
	for _, probe := range probes {
		var result json.RawMessage
		err = survivor.CallProxy(&result, probe.method, probe.args...)
		t.Require().ErrorContainsf(err, "refusing to proxy request to non-leader sequencer",
			"expected overridden conductor to refuse %s after clearing its override", probe.method)
	}
}
