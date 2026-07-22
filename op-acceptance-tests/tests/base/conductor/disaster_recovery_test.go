package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestDisasterRecoveryLeaderOverride verifies the disaster-recovery escape
// hatch: when the conductor cluster loses Raft quorum no sequencer may start
// through the normal leadership path, but an operator can override leadership
// on a surviving node and force it to resume sequencing.
func TestDisasterRecoveryLeaderOverride(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.VerifyOneActiveSequencer()
	followers := sys.Conductors.Followers()
	survivor, casualty := followers[0], followers[1]

	// The disaster: two of the three conductors die, so the survivor can never
	// win an election, and the active sequencer's node dies with its
	// conductor, halting the chain. The casualty conductor goes first so the
	// still-live cluster cannot elect it in between.
	casualty.Stop()
	leader.Stop()
	leader.Sequencer().Stop()

	// Without Raft leadership the survivor's node refuses to sequence.
	survivor.Sequencer().VerifyStartSequencerRefused("sequencer is not the leader")

	// The operator override forces the survivor into non-HA mode: its node
	// stops consulting the conductor and its conductor reports leadership to
	// downstream users (e.g. proxied batcher traffic).
	survivor.Sequencer().OverrideLeader()
	survivor.OverrideLeader(true)
	survivor.Sequencer().StartSequencer()

	// The chain is live again under the surviving sequencer, and the
	// overridden conductor serves proxied traffic as if it were the leader.
	survivor.Sequencer().AdvancedUnsafe(2, 30)
	survivor.VerifyProxyServesSequencerAPIs()
}
