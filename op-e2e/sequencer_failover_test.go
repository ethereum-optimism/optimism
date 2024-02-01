package op_e2e

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

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
// In this test, we test all rpcs exposed by conductor.
func TestSequencerFailover_ConductorRPC(t *testing.T) {
	ctx := context.Background()
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	// SequencerHealthy, Leader, AddServerAsVoter are used in setup already.

	// Test ClusterMembership
	t.Log("Testing ClusterMembership")
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

	// Test LeaderWithID
	t.Log("Testing LeaderWithID")
	leader1, err := c1.client.LeaderWithID(ctx)
	require.NoError(t, err)
	leader2, err := c2.client.LeaderWithID(ctx)
	require.NoError(t, err)
	leader3, err := c3.client.LeaderWithID(ctx)
	require.NoError(t, err)
	require.Equal(t, leader1.ID, leader2.ID, "Expected leader ID to be the same")
	require.Equal(t, leader1.ID, leader3.ID, "Expected leader ID to be the same")

	// Test TransferLeader & TransferLeaderToServer
	t.Log("Testing TransferLeader")
	lid, leader := findLeader(t, conductors)
	err = leader.client.TransferLeader(ctx)
	require.NoError(t, err, "Expected leader to transfer leadership")
	require.NoError(t, waitForLeadershipChange(t, leader, false))
	newLeader, err := leader.client.LeaderWithID(ctx)
	require.NoError(t, err)
	isLeader, err := leader.client.Leader(ctx)
	require.NoError(t, err)
	require.False(t, isLeader, "Expected leader to transfer leadership")
	require.NotEqual(t, lid, newLeader.ID, "Expected leader to change")
	require.NoError(t, waitForSequencerStatusChange(t, sys.RollupClient(newLeader.ID), true))

	// old leader now became follower, we're trying to transfer leadership directly back to it.
	t.Log("Testing TransferLeaderToServer")
	fid, follower := lid, leader
	_, leader = findLeader(t, conductors)
	err = leader.client.TransferLeaderToServer(ctx, fid, follower.ConsensusEndpoint())
	require.NoError(t, err, "Expected leader to transfer leadership to follower")
	require.NoError(t, waitForLeadershipChange(t, follower, true))
	require.NoError(t, waitForSequencerStatusChange(t, sys.RollupClient(fid), true))
	leader = follower

	// Test AddServerAsNonvoter, do not start a new sequencer just for this purpose, use Sequencer3's rpc to start conductor.
	// This is fine as this mainly tests conductor's ability to add itself into the raft consensus cluster as a nonvoter.
	t.Log("Testing AddServerAsNonvoter")
	nonvoter := setupConductor(
		t, VerifierName, t.TempDir(),
		sys.RollupEndpoint(Sequencer3Name),
		sys.NodeEndpoint(Sequencer3Name),
		findAvailablePort(t),
		false,
		*sys.RollupConfig,
	)

	err = leader.client.AddServerAsNonvoter(ctx, VerifierName, nonvoter.ConsensusEndpoint())
	require.NoError(t, err, "Expected leader to add non-voter")
	membership, err = leader.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, len(membership), "Expected 4 members in cluster")
	require.Equal(t, consensus.Nonvoter, membership[3].Suffrage, "Expected last member to be non-voter")

	t.Log("Testing RemoveServer")
	lid, leader = findLeader(t, conductors)
	fid, follower = findFollower(t, conductors)
	err = follower.client.RemoveServer(ctx, lid)
	require.ErrorContains(t, err, "node is not the leader", "Expected follower to fail to remove leader")
	membership, err = c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 4, len(membership), "Expected 4 members in cluster")

	err = leader.client.RemoveServer(ctx, VerifierName)
	require.NoError(t, err, "Expected leader to remove non-voter")
	membership, err = c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(membership), "Expected 2 members in cluster after removal")
	require.NotContains(t, membership, VerifierName, "Expected follower to be removed from cluster")

	err = leader.client.RemoveServer(ctx, fid)
	require.NoError(t, err, "Expected leader to remove follower")
	membership, err = c1.client.ClusterMembership(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(membership), "Expected 2 members in cluster after removal")
	require.NotContains(t, membership, fid, "Expected follower to be removed from cluster")
}

// [Category: Sequencer Failover]
// In this test, we test that the sequencer can successfully failover to a new sequencer once active sequencer goes down.
func TestSequencerFailover_ActiveSequencerDown(t *testing.T) {
	ctx := context.Background()
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	// find leader, stop sequencer completely
	lid, _ := findLeader(t, conductors)
	err := sys.RollupNodes[lid].Stop(ctx)
	require.NoError(t, err, "Expected leader to stop")
	require.NoError(t, waitForHealthChange(t, conductors[lid], false), "Expected leader to become unhealthy")
	require.NoError(t, waitForLeadershipChange(t, conductors[lid], false), "Expected leader to lose leadership")

	t.Log("ensure there's only 1 leader and active sequencer")
	ensureOnlyOneLeader(t, sys, conductors)
}

// [Category: Sequencer Failover]
// In this test, we test that one follower goes down does not affect the cluster.
func TestSequencerFailover_FollowerDown(t *testing.T) {
	ctx := context.Background()
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	// find follower, stop sequencer completely
	fid, _ := findFollower(t, conductors)
	err := sys.RollupNodes[fid].Stop(ctx)
	require.NoError(t, err, "Expected follower to stop")
	require.NoError(t, waitForHealthChange(t, conductors[fid], false), "Expected follower to become unhealthy")

	t.Log("ensure there's only 1 leader and active sequencer")
	ensureOnlyOneLeader(t, sys, conductors)
}

// [Category: Sequencer Failover]
// In this test, we test that when conductor failed on active sequencer A, we'll be able to
// 1. start sequencing on another sequencer B.
// 2. current sequencer A won't cause unsafe reorg despite the fact that it stays in active sequencing mode.
func TestSequencerFailover_ActiveSequencerConductorDown(t *testing.T) {
	ctx := context.Background()
	sys, conductors := setupSequencerFailoverTest(t)
	defer sys.Close()

	t.Log("find leader, stop conductor")
	lid, leader := findLeader(t, conductors)
	require.NoError(t, leader.service.Stop(ctx))
	active, err := sys.RollupClient(lid).SequencerActive(ctx)
	require.NoError(t, err)
	require.True(t, active, "Expect sequencer to stay in active mode")

	t.Log("ensure there's only 1 leader and active sequencer")
	ensureOnlyOneLeader(t, sys, conductors)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		t.Log("ensure original leader unsafe head won't proceed")

		base, err := sys.Clients[lid].BlockNumber(ctx)
		require.NoError(t, err)
		for i := 0; i < 4; i++ {
			now, err := sys.Clients[lid].BlockNumber(ctx)
			require.NoError(t, err)
			require.Equal(t, base, now, "Expect unsafe head to stay the same")
			ss, err := sys.RollupClient(lid).SyncStatus(ctx)
			require.NoError(t, err)
			require.Equal(t, base, ss.UnsafeL2.Number, "Expect unsafe head to stay the same")
			time.Sleep(time.Duration(sys.RollupCfg().BlockTime * uint64(time.Second)))
		}
	}()

	for name, con := range conductors {
		if name == lid {
			continue
		}

		wg.Add(1)
		go func(seq string, c *conductor) {
			defer wg.Done()
			t.Log("ensure the other sequencer's unsafe head is still progressing")

			// There could be a chance that newly taken over sequencer hasn't been able to sequence to tip yet.
			// We call SequencerHealthy here because its monitor contains chain progression check.
			require.NoError(t, waitForHealthChange(t, c, true))
			for i := 0; i < 4; i++ {
				healthy, err := c.client.SequencerHealthy(ctx)
				require.NoError(t, err)
				require.True(t, healthy, "Expect sequencer to stay healthy")
				time.Sleep(time.Duration(sys.RollupCfg().BlockTime * uint64(time.Second)))
				fmt.Println("current time", time.Now())
			}

			base, err := sys.RollupClient(lid).SyncStatus(ctx)
			require.NoError(t, err)
			now, err := sys.RollupClient(seq).SyncStatus(ctx)
			require.NoError(t, err)
			require.Greater(t, now.UnsafeL2.Number, base.UnsafeL2.Number, "Expect unsafe head to progress")
		}(name, con)
	}

	wg.Wait()
}
