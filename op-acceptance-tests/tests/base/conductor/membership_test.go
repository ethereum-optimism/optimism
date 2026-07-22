package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestConductorClusterMembershipChanges verifies an operator can take a
// conductor out of the Raft cluster and add it back — the flow used when
// replacing a sequencer machine — with the cluster membership reflecting each
// change.
func TestConductorClusterMembershipChanges(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.Leader()
	member := sys.Conductors.Followers()[0]

	leader.RemoveFromCluster(member)
	leader.AddVoterToCluster(member)
}

// TestConductorRejectsStaleMembershipVersion verifies the optimistic
// concurrency guard on membership changes: a change submitted against an
// outdated configuration version is refused and leaves the membership
// untouched.
func TestConductorRejectsStaleMembershipVersion(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.Leader()
	member := sys.Conductors.Followers()[0]

	leader.VerifyMembershipChangeRejectsStaleVersion(member)
}
