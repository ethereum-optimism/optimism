package dsl

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

type ConductorSet []*Conductor

func NewConductorSet(inner []stack.Conductor) ConductorSet {
	conductors := make([]*Conductor, len(inner))
	for i, c := range inner {
		conductors[i] = NewConductor(c)
	}
	return conductors
}

type Conductor struct {
	commonImpl
	inner stack.Conductor
}

func NewConductor(inner stack.Conductor) *Conductor {
	return &Conductor{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (c *Conductor) String() string {
	return c.inner.ID().String()
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
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	leader, err := c.inner.RpcAPI().Leader(ctx)
	c.require.NoError(err, "Failed to check if conductor is leader")
	c.log.Info("Checked if conductor is leader", "leader", leader)
	return leader
}

func (c *Conductor) TransferLeadershipTo(targetLeaderInfo consensus.ServerInfo) {
	c.log.Debug("Transferring leadership to target leader", "targetLeaderID", targetLeaderInfo.ID, "targetLeaderAddr", targetLeaderInfo.Addr)
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().TransferLeaderToServer(ctx, targetLeaderInfo.ID, targetLeaderInfo.Addr)
	c.require.NoError(err, "Failed to transfer leadership to target leader", "targetLeaderID", targetLeaderInfo.ID)
	c.log.Info("Transferred leadership to target leader", "targetLeaderID", targetLeaderInfo.ID)
}
