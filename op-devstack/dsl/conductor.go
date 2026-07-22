package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	safety "github.com/ethereum-optimism/optimism/op-service/eth/safety"
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

// awaitLeadership waits until the cluster agrees on exactly one Raft leader.
func (s ConductorSet) awaitLeadership() (*Conductor, []*Conductor) {
	c := s.common()
	var leader *Conductor
	var followers []*Conductor
	err := retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		l, f, err := s.leaderAndFollowers()
		if err != nil {
			c.log.Info("Waiting for conductor cluster to settle on a single Raft leader", "err", err)
			return err
		}
		leader, followers = l, f
		return nil
	})
	c.require.NoError(err, "conductor cluster did not settle on a single Raft leader")
	return leader, followers
}

// Leader waits until the cluster agrees on exactly one Raft leader and returns
// that conductor.
func (s ConductorSet) Leader() *Conductor {
	leader, _ := s.awaitLeadership()
	return leader
}

// Followers waits until the cluster agrees on exactly one Raft leader and
// returns all other conductors.
func (s ConductorSet) Followers() []*Conductor {
	_, followers := s.awaitLeadership()
	return followers
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

// VerifyUnsafeChainAdvancesAndConverges waits until every conductor-managed
// sequencer node advances its unsafe head by at least delta blocks, and until
// all nodes agree on the canonical unsafe chain. Together these prove the
// active sequencer keeps producing blocks and its peers follow the same chain,
// e.g. after a leadership change.
func (s ConductorSet) VerifyUnsafeChainAdvancesAndConverges(delta uint64) {
	c := s.common()
	c.log.Info("Verifying the unsafe chain advances on all sequencer nodes and converges", "delta", delta)
	advanceChecks := make([]CheckFunc, 0, len(s))
	for _, con := range s {
		advanceChecks = append(advanceChecks, con.Sequencer().AdvancedFn(safety.LocalUnsafe, delta, conductorSettleAttempts))
	}
	CheckAll(c.t, advanceChecks...)

	// Check convergence only after every node has advanced. Running these
	// checks in parallel with the advancement checks would let them pass at the
	// shared pre-transfer head without verifying any newly produced block.
	ref := s[0].Sequencer()
	convergenceChecks := make([]CheckFunc, 0, len(s)-1)
	for _, con := range s[1:] {
		convergenceChecks = append(convergenceChecks, con.Sequencer().InSyncFn(ref, safety.LocalUnsafe, conductorSettleAttempts))
	}
	CheckAll(c.t, convergenceChecks...)
	c.log.Info("Verified the unsafe chain advances on all sequencer nodes and converges", "delta", delta)
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

// sequencerHealthy reports this conductor's view of its sequencer's health and
// returns the RPC error, if any, so polling callers can retry on transient
// failures.
func (c *Conductor) sequencerHealthy() (bool, error) {
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	return c.inner.RpcAPI().SequencerHealthy(ctx)
}

// VerifySequencerHealthy waits until this conductor reports its sequencer as
// healthy. Leadership changes may cause brief unhealthiness; this rides those
// out.
func (c *Conductor) VerifySequencerHealthy() {
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
	c.log.Info("Verified sequencer is healthy", "conductor", c)
}

// VerifySequencerActive waits until the sequencer managed by this conductor
// reports that it is actively sequencing.
func (c *Conductor) VerifySequencerActive() {
	c.verifySequencerActive(true)
}

// VerifySequencerInactive waits until the sequencer managed by this conductor
// reports that it has stopped sequencing.
func (c *Conductor) VerifySequencerInactive() {
	c.verifySequencerActive(false)
}

func (c *Conductor) verifySequencerActive(want bool) {
	err := retry.Do0(c.ctx, conductorSettleAttempts, retry.Fixed(2*time.Second), func() error {
		active, err := c.Sequencer().sequencerActive()
		if err != nil {
			return err
		}
		if active != want {
			c.log.Info("Waiting for sequencer active-state", "conductor", c, "want", want, "current", active)
			return fmt.Errorf("sequencer of %s active is %v, want %v", c, active, want)
		}
		return nil
	})
	c.require.NoErrorf(err, "sequencer of conductor %s never reached active=%v", c, want)
	c.log.Info("Verified sequencer active-state", "conductor", c, "active", want)
}

// TransferLeadershipTo transfers Raft leadership from this conductor to the
// target conductor. It waits for the preconditions op-conductor requires (this
// conductor leads and actively sequences, the target is healthy), then
// performs the transfer and waits until the target has taken over leadership.
//
// Sequencer active-state transitions are deliberately not asserted here so
// tests can verify them explicitly; see VerifySequencerActive and
// VerifySequencerInactive.
func (c *Conductor) TransferLeadershipTo(target *Conductor) {
	c.log.Info("Transferring leadership", "from", c, "to", target)
	c.waitForLeadership(true)
	c.VerifySequencerActive()
	target.VerifySequencerHealthy()

	info := c.clusterMemberInfo(target.String())
	ctx, cancel := context.WithTimeout(c.ctx, DefaultTimeout)
	defer cancel()
	err := c.inner.RpcAPI().TransferLeaderToServer(ctx, info.ID, info.Addr)
	c.require.NoErrorf(err, "failed to transfer leadership from %s to %s", c, target)

	target.waitForLeadership(true)
	c.waitForLeadership(false)
	c.log.Info("Transferred leadership", "from", c, "to", target)
}
