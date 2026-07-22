package node

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	node_utils "github.com/ethereum-optimism/optimism/rust/kona/tests/node/utils"
)

// TestConductorLeadershipTransfer rotates Raft leadership through every
// conductor in the cluster and verifies that the active sequencer follows
// leadership on each transfer.
func TestConductorLeadershipTransfer(gt *testing.T) {
	t := devtest.SerialT(gt)

	sys := node_utils.NewMixedOpKonaWithConductors(t)

	for _, conductors := range sys.ConductorSets {
		leader := conductors.Leader()
		// Rotate through every follower and finally back to the original
		// leader, so each cluster member leads once.
		ring := append(conductors.Followers(), leader)
		for _, target := range ring {
			leader.TransferLeadershipTo(target)
			target.VerifySequencerActive()
			leader.VerifySequencerInactive()
			leader = target
		}
	}
}
