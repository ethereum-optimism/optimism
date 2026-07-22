package conductor

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
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

	leader := sys.Conductors.AwaitLeader()
	member := sys.Conductors.Without(leader)[0]
	before := leader.FetchClusterMembership()

	leader.RemoveFromCluster(member)
	removed := leader.FetchClusterMembership()
	t.Require().Len(removed.Servers, len(before.Servers)-1)
	t.Require().ElementsMatch(withoutMember(before.Servers, member.String()), removed.Servers)

	leader.AddVoterToCluster(member)
	after := leader.FetchClusterMembership()
	t.Require().Len(after.Servers, len(before.Servers))
	t.Require().Equal(consensus.Voter, serverInfo(t, after, member.String()).Suffrage)
	t.Require().ElementsMatch(serverIDs(before), serverIDs(after))
}

// TestConductorRejectsStaleMembershipVersion verifies the optimistic
// concurrency guard on membership changes: a change submitted against an
// outdated configuration version is refused and leaves the membership
// untouched.
func TestConductorRejectsStaleMembershipVersion(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sysgo.SkipOnKonaNode(t, "kona-node conductor support is tracked by #21906")

	sys := presets.NewMinimalWithConductors(t)

	leader := sys.Conductors.AwaitLeader()
	member := sys.Conductors.Without(leader)[0]

	membership := leader.FetchClusterMembership()
	staleVersion := membership.Version - 1

	err := leader.RemoveServer(member.String(), staleVersion)
	t.Require().ErrorContainsf(err, "configuration changed since",
		"expected removal of %s with a stale configuration version to be refused", member)
	t.Require().Equalf(membership, leader.FetchClusterMembership(),
		"membership must be unchanged after refused removal of %s", member)

	err = leader.AddServerAsNonvoter("replacement", member.Escape().ConsensusEndpoint(), staleVersion)
	t.Require().ErrorContains(err, "configuration changed since",
		"expected non-voter addition with a stale configuration version to be refused")
	t.Require().Equal(membership, leader.FetchClusterMembership(),
		"membership must be unchanged after refused non-voter addition")
}

func serverIDs(membership *consensus.ClusterMembership) []string {
	ids := make([]string, 0, len(membership.Servers))
	for _, server := range membership.Servers {
		ids = append(ids, server.ID)
	}
	return ids
}

func withoutMember(servers []consensus.ServerInfo, id string) []consensus.ServerInfo {
	out := make([]consensus.ServerInfo, 0, len(servers)-1)
	for _, server := range servers {
		if server.ID != id {
			out = append(out, server)
		}
	}
	return out
}

func serverInfo(t devtest.T, membership *consensus.ClusterMembership, id string) consensus.ServerInfo {
	for _, server := range membership.Servers {
		if server.ID == id {
			return server
		}
	}
	t.Require().FailNowf("missing cluster member", "no member %q in %v", id, membership.Servers)
	return consensus.ServerInfo{}
}
