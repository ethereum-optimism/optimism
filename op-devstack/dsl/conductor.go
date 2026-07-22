package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// conductorSettleAttempts bounds the polling loops that wait for cluster-wide
// conditions (leadership, sequencer active-state). Polls are 2s apart.
const conductorSettleAttempts = 30

type ConductorSet []*Conductor

// common returns the shared test plumbing of the set. The preset always
// constructs non-empty sets; an empty set is a wiring bug, not a test failure.
func (s ConductorSet) common() commonImpl {
	if len(s) == 0 {
		panic("empty conductor set: preset wiring is broken")
	}
	return s[0].commonImpl
}

// leaderAndFollowers samples Raft leadership across the set once, requiring
// exactly one leader. It returns errors instead of asserting so polling
// callers can treat transient RPC failures and unsettled elections as retries.
func (s ConductorSet) leaderAndFollowers() (*Conductor, []*Conductor, error) {
	var leader *Conductor
	followers := make([]*Conductor, 0, len(s))
	for _, c := range s {
		isLeader, err := c.isLeader()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to check leadership of %s: %w", c, err)
		}
		if !isLeader {
			followers = append(followers, c)
			continue
		}
		if leader != nil {
			return nil, nil, fmt.Errorf("multiple Raft leaders: %s and %s", leader, c)
		}
		leader = c
	}
	if leader == nil {
		return nil, nil, fmt.Errorf("no Raft leader among %d conductors", len(s))
	}
	return leader, followers, nil
}

// VerifyOneActiveSequencer waits until the cluster has exactly one Raft leader
// and verifies that only the leader's sequencer is active — the core HA
// guarantee that exactly one node sequences at any time. It returns the
// leader's conductor.
func (s ConductorSet) VerifyOneActiveSequencer() *Conductor {
	c := s.common()
	var leader *Conductor
	err := retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		l, followers, err := s.leaderAndFollowers()
		if err != nil {
			c.log.Info("Waiting for conductor cluster to settle on a single Raft leader", "err", err)
			return err
		}
		active, err := l.Sequencer().sequencerActive()
		if err != nil {
			return err
		}
		if !active {
			c.log.Info("Waiting for leader's sequencer to become active", "leader", l)
			return fmt.Errorf("leader %s sequencer is not active", l)
		}
		for _, f := range followers {
			active, err := f.Sequencer().sequencerActive()
			if err != nil {
				return err
			}
			if active {
				c.log.Info("Waiting for follower's sequencer to become inactive", "follower", f)
				return fmt.Errorf("follower %s sequencer is active", f)
			}
		}
		leader = l
		return nil
	})
	c.require.NoError(err, "expected exactly one active sequencer, the Raft leader's")
	c.log.Info("Verified exactly one active sequencer", "leader", leader)
	return leader
}

type Conductor struct {
	commonImpl
	inner     stack.Conductor
	sequencer *L2CLNode
}

func NewConductor(inner stack.Conductor) *Conductor {
	return &Conductor{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

// AttachSequencer links the L2CL node whose sequencer this conductor manages.
// Preset wiring calls this once at construction time.
func (c *Conductor) AttachSequencer(cl *L2CLNode) {
	c.sequencer = cl
}

// Sequencer returns the L2CL node whose sequencer this conductor manages.
func (c *Conductor) Sequencer() *L2CLNode {
	c.require.NotNil(c.sequencer, "conductor %s has no sequencer node attached", c)
	return c.sequencer
}

func (c *Conductor) String() string {
	return c.inner.Name()
}

func (c *Conductor) Escape() stack.Conductor {
	return c.inner
}

func (c *Conductor) FetchClusterMembership() *consensus.ClusterMembership {
	c.log.Debug("Fetching cluster membership")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	clusterMembership, err := retry.Do(ctx, 2, retry.Fixed(500*time.Millisecond), func() (*consensus.ClusterMembership, error) {
		clusterMembership, err := c.inner.RpcAPI().ClusterMembership(c.ctx)
		return clusterMembership, err
	})
	c.require.NoError(err, "Failed to fetch cluster membership")
	c.log.Info("Fetched cluster membership",
		"clusterMembership", clusterMembership)
	return clusterMembership
}

func (c *Conductor) FetchLeader() *consensus.ServerInfo {
	c.log.Debug("Fetching leader information")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	leaderInfo, err := retry.Do[*consensus.ServerInfo](ctx, 2, retry.Fixed(500*time.Millisecond), func() (*consensus.ServerInfo, error) {
		leaderInfo, err := c.inner.RpcAPI().LeaderWithID(c.ctx)
		return leaderInfo, err
	})
	c.require.NoError(err, "Failed to fetch leader information")
	c.log.Info("Fetched leader information",
		"leaderInfo", leaderInfo)
	return leaderInfo
}

func (c *Conductor) FetchSequencerHealthy() bool {
	c.log.Debug("Fetching sequencer healthy status")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	healthy, err := c.inner.RpcAPI().SequencerHealthy(ctx)
	c.require.NoError(err, "Failed to fetch sequencer healthy status")
	c.log.Info("Fetched sequencer healthy status", "healthy", healthy)
	return healthy
}

func (c *Conductor) FetchPaused() bool {
	c.log.Debug("Fetching paused status")
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	paused, err := c.inner.RpcAPI().Paused(ctx)
	c.require.NoError(err, "Failed to fetch paused status")
	c.log.Info("Fetched paused status", "paused", paused)
	return paused
}

func (c *Conductor) IsLeader() bool {
	c.log.Debug("Checking if conductor is leader")
	leader, err := c.isLeader()
	c.require.NoError(err, "Failed to check if conductor is leader")
	c.log.Info("Checked if conductor is leader", "leader", leader)
	return leader
}

// isLeader reports Raft leadership and returns the RPC error, if any. Internal
// callers in retry loops use this so a transient RPC failure counts as a retry
// rather than an instant FailNow.
func (c *Conductor) isLeader() (bool, error) {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	return c.inner.RpcAPI().Leader(ctx)
}

func (c *Conductor) TransferLeadershipTo(targetLeaderInfo consensus.ServerInfo) {
	c.log.Debug("Transferring leadership to target leader", "targetLeaderID", targetLeaderInfo.ID, "targetLeaderAddr", targetLeaderInfo.Addr)
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().TransferLeaderToServer(ctx, targetLeaderInfo.ID, targetLeaderInfo.Addr)
	c.require.NoError(err, "Failed to transfer leadership to target leader", "targetLeaderID", targetLeaderInfo.ID)
	c.log.Info("Transferred leadership to target leader", "targetLeaderID", targetLeaderInfo.ID)
}
