package monitor

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor-mon/pkg/config"
	"github.com/ethereum-optimism/optimism/op-conductor-mon/pkg/metrics"
	"github.com/ethereum-optimism/optimism/op-conductor-mon/pkg/metrics/opconductor_client"
	"github.com/ethereum/go-ethereum/log"
)

func (p *Poller) cleanup(ctx context.Context) {
	defer p.mutex.Unlock()
	p.mutex.Lock()
	for nodeName, nodeState := range p.state {
		if time.Since(nodeState.updatedAt) > p.config.NodeStateExpiration {
			log.Warn("node state expired",
				"node", nodeName,
				"updated_at", nodeState.updatedAt)
			delete(p.state, nodeName)
		}
	}
}

func (p *Poller) poll(ctx context.Context) {
	for nodeName, nodeConfig := range p.nodesConfig {
		p.pollNode(ctx, nodeName, nodeConfig)
	}
}

func (p *Poller) pollNode(ctx context.Context, nodeName string, nodeConfig *config.NodeConfig) {
	log.Debug("polling node",
		"name", nodeName,
		"rpc", nodeConfig.RPCAddress)

	client, err := opconductor_client.New(ctx, p.config, nodeName, nodeConfig.RPCAddress)
	if err != nil {
		return
	}

	// conductor status
	paused, err := client.Paused(ctx)
	if err != nil {
		log.Error("cant get paused",
			"node", nodeName,
			"err", err)
		return
	}
	log.Debug("got paused", "node", nodeName, "paused", paused)

	stopped, err := client.Stopped(ctx)
	if err != nil {
		log.Error("cant get stopped",
			"node", nodeName,
			"err", err)
		return
	}
	log.Debug("got stopped", "node", nodeName, "stopped", stopped)

	active, err := client.Active(ctx)
	if err != nil {
		log.Error("cant get active",
			"node", nodeName,
			"err", err)
		return
	}
	log.Debug("got active", "node", nodeName, "active", active)

	// sequencer status
	healthy, err := client.SequencerHealthy(ctx)
	if err != nil {
		log.Error("cant get sequencer healthy",
			"node", nodeName,
			"err", err)
		return
	}
	log.Debug("got sequencer healthy", "node", nodeName, "healthy", healthy)

	leader, err := client.Leader(ctx)
	if err != nil {
		log.Error("cant get leader",
			"node", nodeName,
			"err", err)
		return
	}
	log.Debug("got leader", "node", nodeName, "leader", leader)

	// raft status
	leaderWithID, err := client.LeaderWithID(ctx)
	if err != nil {
		log.Error("cant get leader with id",
			"node", nodeName,
			"err", err)
		return
	}
	log.Debug("got leader with id", "node", nodeName, "leader_with_id", leaderWithID)

	clusterMembership, err := client.ClusterMembership(ctx)
	if err != nil {
		log.Error("cant get cluster membership",
			"node", nodeName,
			"err", err)
		return
	}
	log.Debug("got cluster membership", "node", nodeName, "cluster_membership", clusterMembership)

	// update node state
	nodeState := &NodeState{
		paused:  paused,
		stopped: stopped,
		active:  active,

		healthy: healthy,
		leader:  leader,

		leaderWithID:      leaderWithID,
		clusterMembership: clusterMembership,

		updatedAt: time.Now(),
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.state[nodeName] = nodeState
}

func (p *Poller) reportMetrics(ctx context.Context) {
	log.Debug("report metrics",
		"state_len", len(p.state))

	for nodeName, nodeState := range p.state {
		p.reportNodeMetrics(ctx, nodeName, nodeState)
	}
}

func (p *Poller) reportNodeMetrics(ctx context.Context, name string, state *NodeState) {
	log.Debug("report node metrics",
		"node", name)

	// conductor status
	metrics.RecordNodeState(name, "paused", state.paused)
	metrics.RecordNodeState(name, "stopped", state.stopped)
	metrics.RecordNodeState(name, "active", state.active)

	// sequencer status
	metrics.RecordNodeState(name, "healthy", state.healthy)
	metrics.RecordNodeState(name, "leader", state.leader)

	// raft status
	metrics.ReportNodeLeader(name, state.leaderWithID.ID)
	metrics.ReportClusterMembershipCount(name, len(state.clusterMembership))
}
