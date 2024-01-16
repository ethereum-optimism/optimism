package op_e2e

import (
	"context"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
)

// [Category: Initial Setup]
// In this test, we test that we can successfully setup a working cluster.
func TestSequencerFailover_SetupCluster(t *testing.T) {
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	require.Equal(t, 3, len(conductors), "Expected 3 conductors")
	for _, con := range conductors {
		require.NotNil(t, con, "Expected conductor to be non-nil")
	}
}

// [Category: conductor rpc]
// In this test, we test all the rpcs exposed by conductor.
func TestSequencerFailover_ConductorRPC(t *testing.T) {
	ctx := context.Background()
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	// SequencerHealthy, Leader, AddServerAsVoter are used in setup already.

	// Test ClusterMembership
	c1 := conductors[Sequencer1Name]
	c2 := conductors[Sequencer2Name]
	c3 := conductors[Sequencer3Name]
	membership, err := c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(membership), "Expected 3 members in cluster")
	ids := make([]string, 0)
	for _, member := range membership {
		ids = append(ids, member.ID)
		require.Equal(t, consensus.Voter, member.Suffrage, "Expected all members to be voters")
	}
	sort.Strings(ids)
	require.Equal(t, []string{Sequencer1Name, Sequencer2Name, Sequencer3Name}, ids, "Expected all sequencers to be in cluster")

	// Test Active & Pause & Resume
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

	// TestLeaderWithID
	leader1, err := c1.client.LeaderWithID(ctx)
	require.NoError(t, err)
	leader2, err := c2.client.LeaderWithID(ctx)
	require.NoError(t, err)
	leader3, err := c3.client.LeaderWithID(ctx)
	require.NoError(t, err)
	require.Equal(t, leader1.ID, leader2.ID, "Expected leader ID to be the same")
	require.Equal(t, leader1.ID, leader3.ID, "Expected leader ID to be the same")

	// RemoveServer
	lid, leader := findLeader(t, conductors)
	fid, follower := findFollower(t, conductors)

	err = follower.client.RemoveServer(ctx, lid)
	require.ErrorContains(t, err, "node is not the leader", "Expected follower to fail to remove leader")
	membership, err = c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(membership), "Expected 3 members in cluster")

	err = leader.client.RemoveServer(ctx, fid)
	require.NoError(t, err, "Expected leader to remove follower")
	membership, err = c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(membership), "Expected 2 members in cluster after removal")
	require.NotContains(t, membership, fid, "Expected follower to be removed from cluster")

	// TransferLeader & TransferLeaderToServer
	// pause all conductors for now because no sequencer is committing unsafe payload now, thus causing infinite leadership transfer loop.
	// PR is work in progress here:
	// https://github.com/ethereum-optimism/optimism/pull/8894
	// TODO: (https://github.com/ethereum-optimism/protocol-quest/issues/85) remove this once above PR is merged.
	for _, con := range conductors {
		err = con.client.Pause(ctx)
		require.NoError(t, err)
	}

	err = leader.client.TransferLeader(ctx)
	require.NoError(t, err, "Expected leader to transfer leadership")
	require.NoError(t, waitForLeadershipChange(t, leader, false))
	newLeader, err := leader.client.LeaderWithID(ctx)
	require.NoError(t, err)
	isLeader, err := leader.client.Leader(ctx)
	require.NoError(t, err)
	require.False(t, isLeader, "Expected leader to transfer leadership")
	require.NotEqual(t, lid, newLeader.ID, "Expected leader to change")

	// old leader now became follower, we're trying to transfer leadership directly back to it.
	fid, follower = lid, leader
	_, leader = findLeader(t, conductors)
	err = leader.client.TransferLeaderToServer(ctx, fid, follower.ConsensusEndpoint())
	require.NoError(t, err, "Expected leader to transfer leadership to follower")
	require.NoError(t, waitForLeadershipChange(t, follower, true))
}
