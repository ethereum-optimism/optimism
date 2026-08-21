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

// Without returns the set without the given conductor, e.g. the survivors of
// an injected failure.
func (s ConductorSet) Without(exclude *Conductor) ConductorSet {
	out := make(ConductorSet, 0, len(s))
	for _, c := range s {
		if c != exclude {
			out = append(out, c)
		}
	}
	return out
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

// awaitLeadership waits until the cluster agrees on exactly one Raft leader.
func (s ConductorSet) awaitLeadership() *Conductor {
	c := s.common()
	var leader *Conductor
	err := retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		l, _, err := s.leaderAndFollowers()
		if err != nil {
			c.log.Info("Waiting for conductor cluster to settle on a single Raft leader", "err", err)
			return err
		}
		leader = l
		return nil
	})
	c.require.NoError(err, "conductor cluster did not settle on a single Raft leader")
	return leader
}

// AwaitLeader waits until the cluster agrees on exactly one Raft leader and
// returns that conductor.
func (s ConductorSet) AwaitLeader() *Conductor {
	return s.awaitLeadership()
}

// AwaitOneActiveSequencer waits until the cluster has exactly one Raft leader,
// the leader's sequencer is active, and every follower's sequencer is inactive.
// It returns the leader's conductor.
func (s ConductorSet) AwaitOneActiveSequencer() *Conductor {
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
	c.log.Info("Conductor cluster reached exactly one active sequencer", "leader", leader)
	return leader
}

type Conductor struct {
	commonImpl
	inner     stack.Conductor
	sequencer *L2CLNode
}

func NewConductor(inner stack.Conductor, sequencer *L2CLNode) *Conductor {
	common := commonFromT(inner.T())
	common.require.NotNil(sequencer, "conductor %s has no sequencer node", inner.Name())
	return &Conductor{
		commonImpl: common,
		inner:      inner,
		sequencer:  sequencer,
	}
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

// clusterMemberInfo returns the Raft ServerInfo of the cluster member with the
// given ID (conductor names double as Raft server IDs in the presets).
func (c *Conductor) clusterMemberInfo(id string) consensus.ServerInfo {
	membership := c.FetchClusterMembership()
	for _, member := range membership.Servers {
		if member.ID == id {
			return member
		}
	}
	c.require.FailNowf("unknown cluster member", "no member %q in cluster membership %v", id, membership.Servers)
	return consensus.ServerInfo{}
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

// awaitClusterMembership waits until the presence of the given server ID in
// the cluster membership matches want.
func (c *Conductor) awaitClusterMembership(id string, want bool) {
	err := retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
		defer cancel()
		membership, err := c.inner.RpcAPI().ClusterMembership(ctx)
		if err != nil {
			return err
		}
		present := false
		for _, member := range membership.Servers {
			if member.ID == id {
				present = true
				break
			}
		}
		if present != want {
			c.log.Info("Waiting for cluster membership", "conductor", c, "member", id, "want", want, "present", present)
			return fmt.Errorf("membership of %s is %v, want %v", id, present, want)
		}
		return nil
	})
	c.require.NoErrorf(err, "cluster membership never reflected member %s present=%v", id, want)
}

// RemoveServer removes a server using the given Raft configuration version.
func (c *Conductor) RemoveServer(id string, version uint64) error {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	return c.inner.RpcAPI().RemoveServer(ctx, id, version)
}

// AddServerAsVoter adds a server using the given Raft configuration version.
func (c *Conductor) AddServerAsVoter(id, addr string, version uint64) error {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	return c.inner.RpcAPI().AddServerAsVoter(ctx, id, addr, version)
}

// AddServerAsNonvoter adds a server using the given Raft configuration version.
func (c *Conductor) AddServerAsNonvoter(id, addr string, version uint64) error {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	return c.inner.RpcAPI().AddServerAsNonvoter(ctx, id, addr, version)
}

// RemoveFromCluster removes the target conductor from the Raft cluster, using
// the current configuration version, and waits for the change to take effect.
func (c *Conductor) RemoveFromCluster(target *Conductor) {
	c.log.Info("Removing conductor from cluster", "leader", c, "target", target)
	membership := c.FetchClusterMembership()
	err := c.RemoveServer(target.String(), membership.Version)
	c.require.NoErrorf(err, "failed to remove %s from the cluster", target)
	c.awaitClusterMembership(target.String(), false)
	c.log.Info("Removed conductor from cluster", "leader", c, "target", target)
}

// AddVoterToCluster adds the target conductor to the Raft cluster as a voter,
// using the current configuration version, and waits for the change to take
// effect.
func (c *Conductor) AddVoterToCluster(target *Conductor) {
	c.log.Info("Adding conductor to cluster as voter", "leader", c, "target", target)
	membership := c.FetchClusterMembership()
	err := c.AddServerAsVoter(target.String(), target.Escape().ConsensusEndpoint(), membership.Version)
	c.require.NoErrorf(err, "failed to add %s to the cluster as voter", target)
	c.awaitClusterMembership(target.String(), true)
	c.log.Info("Added conductor to cluster as voter", "leader", c, "target", target)
}

// Stop shuts down the conductor service via its admin RPC and waits until the
// conductor stops serving RPC. The service cannot be restarted; tests use this
// to take cluster members out, e.g. to simulate loss of Raft quorum.
func (c *Conductor) Stop() {
	c.log.Info("Stopping conductor", "conductor", c)
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().Stop(ctx)
	c.require.NoErrorf(err, "failed to stop conductor %s", c)
	err = retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		if _, err := c.isLeader(); err == nil {
			return fmt.Errorf("conductor %s is still serving RPC after stop", c)
		}
		return nil
	})
	c.require.NoErrorf(err, "conductor %s never stopped serving RPC", c)
	c.log.Info("Stopped conductor", "conductor", c)
}

// OverrideLeader forces this conductor to report itself as leader regardless
// of actual Raft state, or clears the override again with false. This is the
// disaster-recovery escape hatch used when the cluster cannot form a quorum.
func (c *Conductor) OverrideLeader(override bool) {
	c.log.Info("Setting conductor leader override", "conductor", c, "override", override)
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().OverrideLeader(ctx, override)
	c.require.NoErrorf(err, "failed to set leader override of conductor %s to %v", c, override)
	c.log.Info("Set conductor leader override", "conductor", c, "override", override)
}

// LeaderOverridden reports whether this conductor's leader override is set.
func (c *Conductor) LeaderOverridden() bool {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	overridden, err := c.inner.RpcAPI().LeaderOverridden(ctx)
	c.require.NoErrorf(err, "failed to fetch leader override of conductor %s", c)
	return overridden
}

// CallProxy calls a sequencer API through this conductor's RPC proxy.
func (c *Conductor) CallProxy(result any, method string, args ...any) error {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	return c.inner.ProxyRPC().CallContext(ctx, result, method, args...)
}

// waitForLeadership waits until this conductor's Raft leadership matches want.
func (c *Conductor) waitForLeadership(want bool) {
	err := retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		leader, err := c.isLeader()
		if err != nil {
			return err
		}
		if leader != want {
			c.log.Info("Waiting for conductor leadership state", "conductor", c, "want", want, "current", leader)
			return fmt.Errorf("conductor %s leadership is %v, want %v", c, leader, want)
		}
		return nil
	})
	c.require.NoErrorf(err, "conductor %s never reached leadership=%v", c, want)
}

// AwaitNotLeader waits until this conductor no longer reports Raft leadership.
func (c *Conductor) AwaitNotLeader() {
	c.waitForLeadership(false)
}

// sequencerHealthy reports this conductor's view of its sequencer's health and
// returns the RPC error, if any, so polling callers can retry on transient
// failures.
func (c *Conductor) sequencerHealthy() (bool, error) {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	return c.inner.RpcAPI().SequencerHealthy(ctx)
}

// AwaitSequencerHealthy waits until this conductor reports its sequencer as
// healthy. Leadership changes may cause brief unhealthiness; this rides those
// out.
func (c *Conductor) AwaitSequencerHealthy() {
	err := retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		healthy, err := c.sequencerHealthy()
		if err != nil {
			return err
		}
		if !healthy {
			c.log.Info("Waiting for sequencer to become healthy", "conductor", c)
			return fmt.Errorf("conductor %s reports unhealthy sequencer", c)
		}
		return nil
	})
	c.require.NoErrorf(err, "conductor %s never reported a healthy sequencer", c)
	c.log.Info("Sequencer is healthy", "conductor", c)
}

// TransferLeadership transfers Raft leadership to an unspecified eligible
// voter and waits for the cluster to settle on a different active sequencer.
func (c *Conductor) TransferLeadership(cluster ConductorSet) *Conductor {
	c.log.Info("Transferring leadership", "from", c)
	c.waitForLeadership(true)
	c.Sequencer().AwaitSequencerActive()

	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().TransferLeader(ctx)
	c.require.NoErrorf(err, "failed to transfer leadership from %s", c)

	var next *Conductor
	err = retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		leader, _, err := cluster.leaderAndFollowers()
		if err != nil {
			return err
		}
		if leader == c {
			return fmt.Errorf("conductor %s is still the leader", c)
		}
		next = leader
		return nil
	})
	c.require.NoErrorf(err, "conductor %s never transferred leadership", c)
	cluster.AwaitOneActiveSequencer()
	c.AwaitSequencerHealthy()
	next.AwaitSequencerHealthy()
	return next
}

// TransferLeadershipTo safely transfers Raft leadership and sequencing from
// this conductor to the target. It waits for the source to lead and sequence,
// the target to be healthy, and both the Raft and sequencer state transitions
// to complete before returning.
func (c *Conductor) TransferLeadershipTo(target *Conductor) {
	c.log.Info("Transferring leadership", "from", c, "to", target)
	c.waitForLeadership(true)
	c.Sequencer().AwaitSequencerActive()
	target.AwaitSequencerHealthy()

	info := c.clusterMemberInfo(target.String())
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().TransferLeaderToServer(ctx, info.ID, info.Addr)
	c.require.NoErrorf(err, "failed to transfer leadership from %s to %s", c, target)

	target.waitForLeadership(true)
	c.waitForLeadership(false)
	target.Sequencer().AwaitSequencerActive()
	c.Sequencer().AwaitSequencerInactive()
	c.AwaitSequencerHealthy()
	target.AwaitSequencerHealthy()
	c.log.Info("Transferred leadership", "from", c, "to", target)
}
