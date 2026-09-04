package conductor

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// [Category: conductor rpc]
// This test covers conductor RPC behaviors that are not exercised by acceptance tests.
func TestSequencerFailover_ConductorRPC(t *testing.T) {
	t.Skip("Flaky test tracked in ethereum-optimism/optimism#22094")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	sys, conductors, cleanup := setupSequencerFailoverTest(t)
	defer cleanup()

	// SequencerHealthy, Leader, AddServerAsVoter are used in setup already.

	// Test ClusterMembership
	t.Log("Testing ClusterMembership")
	c1 := conductors[Sequencer1Name]
	c2 := conductors[Sequencer2Name]
	c3 := conductors[Sequencer3Name]
	membership, err := c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(membership.Servers), "Expected 3 members in cluster")
	ids := make([]string, 0)
	for _, member := range membership.Servers {
		ids = append(ids, member.ID)
		require.Equal(t, consensus.Voter, member.Suffrage, "Expected all members to be voters")
	}
	sort.Strings(ids)
	require.Equal(t, []string{Sequencer1Name, Sequencer2Name, Sequencer3Name}, ids, "Expected all sequencers to be in cluster")

	// Test Active & Pause & Resume & Stop
	t.Log("Testing Active & Pause & Resume")
	active, err := c1.client.Active(ctx)
	require.NoError(t, err)
	require.True(t, active, "Expected conductor to be active")

	err = c1.client.Pause(ctx)
	require.NoError(t, err)
	active, err = c1.client.Active(ctx)
	require.NoError(t, err)
	require.False(t, active, "Expected conductor to be paused")

	err = c1.client.Resume(ctx)
	require.NoError(t, err)
	active, err = c1.client.Active(ctx)
	require.NoError(t, err)
	require.True(t, active, "Expected conductor to be active")

	t.Log("Testing LeaderWithID")
	leader1, err := c1.client.LeaderWithID(ctx)
	require.NoError(t, err)
	leader2, err := c2.client.LeaderWithID(ctx)
	require.NoError(t, err)
	leader3, err := c3.client.LeaderWithID(ctx)
	require.NoError(t, err)
	require.Equal(t, leader1.ID, leader2.ID, "Expected leader ID to be the same")
	require.Equal(t, leader1.ID, leader3.ID, "Expected leader ID to be the same")

	_, leader := findLeader(t, conductors)

	// Test AddServerAsNonvoter, do not start a new sequencer just for this purpose, use Sequencer3's rpc to start conductor.
	// This is fine as this mainly tests conductor's ability to add itself into the raft consensus cluster as a nonvoter.
	t.Log("Testing AddServerAsNonvoter")
	nonvoter, err := retry.Do[*conductor](ctx, maxSetupRetries, retryStrategy, func() (*conductor, error) {
		return setupConductor(
			t, VerifierName, t.TempDir(),
			sys.RollupEndpoint(Sequencer3Name).RPC(),
			sys.NodeEndpoint(Sequencer3Name).RPC(),
			false,
			true,
			*sys.RollupConfig,
		)
	})
	require.NoError(t, err)
	defer func() {
		err = nonvoter.service.Stop(ctx)
		require.NoError(t, err)
	}()

	err = leader.client.AddServerAsNonvoter(ctx, VerifierName, nonvoter.ConsensusEndpoint(), 0)
	require.NoError(t, err, "Expected leader to add non-voter")
	membership, err = leader.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, len(membership.Servers), "Expected 4 members in cluster")
	require.Equal(t, consensus.Nonvoter, membership.Servers[3].Suffrage, "Expected last member to be non-voter")

	t.Log("Testing RemoveServer, call remove on follower, expected to fail")
	lid, leader := findLeader(t, conductors)
	_, follower := findFollower(t, conductors)
	err = follower.client.RemoveServer(ctx, lid, membership.Version)
	require.ErrorContains(t, err, "node is not the leader", "Expected follower to fail to remove leader")
	membership, err = c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, len(membership.Servers), "Expected 4 members in cluster")

	t.Log("Testing RemoveServer, call remove on leader, expect non-voter to be removed")
	err = leader.client.RemoveServer(ctx, VerifierName, membership.Version)
	require.NoError(t, err, "Expected leader to remove non-voter")
	membership, err = c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(membership.Servers), "Expected 2 members in cluster after removal")
	require.NotContains(t, memberIDs(membership), VerifierName, "Expected follower to be removed from cluster")

	// Testing the stop api
	t.Log("Testing Stop API")
	err = c1.client.Stop(ctx)
	require.NoError(t, err)
	_, err = c1.client.Stopped(ctx)
	require.Error(t, err, "Expected no connection to the conductor since it's stopped")
}
