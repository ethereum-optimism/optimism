package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-conductor-mon/pkg/config"
	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum/go-ethereum/log"
)

type Poller struct {
	nodesConfig map[string]*config.NodeConfig
	config      *config.Config

	mutex sync.Mutex
	state map[string]*NodeState

	cancelFunc context.CancelFunc
}

type NodeState struct {
	// conductor status
	paused  bool
	stopped bool
	active  bool

	// sequencer status
	healthy bool
	leader  bool

	// raft status
	leaderWithID      *consensus.ServerInfo
	clusterMembership []*consensus.ServerInfo

	updatedAt time.Time
}

func New(
	config *config.Config,
	nodesConfig map[string]*config.NodeConfig) *Poller {
	poller := &Poller{
		nodesConfig: nodesConfig,
		config:      config,

		state: make(map[string]*NodeState),
	}
	return poller
}

func (p *Poller) Start(ctx context.Context) {
	networkCtx, cancelFunc := context.WithCancel(ctx)
	p.cancelFunc = cancelFunc

	schedule(networkCtx, p.config.PollInterval, p.Tick)
}

func (p *Poller) Shutdown() {
	if p.cancelFunc != nil {
		p.cancelFunc()
	}
}

func (p *Poller) Tick(ctx context.Context) {
	log.Debug("tick")

	// clean up expired state
	p.cleanup(ctx)

	// poll members for current state
	p.poll(ctx)

	// report state to metrics
	p.reportMetrics(ctx)

	log.Debug("tick done")
}
