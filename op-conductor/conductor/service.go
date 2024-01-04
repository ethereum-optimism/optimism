package conductor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"

	"github.com/ethereum-optimism/optimism/op-conductor/client"
	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-conductor/health"
	opp2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

var (
	ErrResumeTimeout      = errors.New("timeout to resume conductor")
	ErrPauseTimeout       = errors.New("timeout to pause conductor")
	ErrUnsafeHeadMismarch = errors.New("unsafe head mismatch")
)

// New creates a new OpConductor instance.
func New(ctx context.Context, cfg *Config, log log.Logger, version string) (*OpConductor, error) {
	return NewOpConductor(ctx, cfg, log, version, nil, nil, nil)
}

// NewOpConductor creates a new OpConductor instance.
func NewOpConductor(
	ctx context.Context,
	cfg *Config,
	log log.Logger,
	version string,
	ctrl client.SequencerControl,
	cons consensus.Consensus,
	hmon health.HealthMonitor,
) (*OpConductor, error) {
	if err := cfg.Check(); err != nil {
		return nil, errors.Wrap(err, "invalid config")
	}

	oc := &OpConductor{
		log:          log,
		version:      version,
		cfg:          cfg,
		pauseCh:      make(chan struct{}),
		pauseDoneCh:  make(chan struct{}),
		resumeCh:     make(chan struct{}),
		resumeDoneCh: make(chan struct{}),
		actionCh:     make(chan struct{}, 1),
		ctrl:         ctrl,
		cons:         cons,
		hmon:         hmon,
	}
	oc.actionFn = oc.action

	// explicitly set all atomic.Bool values
	oc.leader.Store(false)    // upon start, it should not be the leader unless specified otherwise by raft bootstrap, in that case, it'll receive a leadership update from consensus.
	oc.healthy.Store(true)    // default to healthy unless reported otherwise by health monitor.
	oc.seqActive.Store(false) // explicitly set to false by default, the real value will be reported after sequencer control initialization.
	oc.paused.Store(cfg.Paused)
	oc.stopped.Store(false)

	err := oc.init(ctx)
	if err != nil {
		log.Error("failed to initialize OpConductor", "err", err)
		// ensure we always close the resources if we fail to initialize the conductor.
		if closeErr := oc.Stop(ctx); closeErr != nil {
			return nil, multierror.Append(err, closeErr)
		}
	}

	return oc, nil
}

func (c *OpConductor) init(ctx context.Context) error {
	c.log.Info("initializing OpConductor", "version", c.version)
	if err := c.initSequencerControl(ctx); err != nil {
		return errors.Wrap(err, "failed to initialize sequencer control")
	}
	if err := c.initConsensus(ctx); err != nil {
		return errors.Wrap(err, "failed to initialize consensus")
	}
	if err := c.initHealthMonitor(ctx); err != nil {
		return errors.Wrap(err, "failed to initialize health monitor")
	}
	return nil
}

func (c *OpConductor) initSequencerControl(ctx context.Context) error {
	if c.ctrl != nil {
		return nil
	}

	ec, err := opclient.NewRPC(ctx, c.log, c.cfg.ExecutionRPC)
	if err != nil {
		return errors.Wrap(err, "failed to create geth rpc client")
	}
	execCfg := sources.L2ClientDefaultConfig(&c.cfg.RollupCfg, true)
	// TODO: Add metrics tracer here. tracked by https://github.com/ethereum-optimism/protocol-quest/issues/45
	exec, err := sources.NewEthClient(ec, c.log, nil, &execCfg.EthClientConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create geth client")
	}

	nc, err := opclient.NewRPC(ctx, c.log, c.cfg.NodeRPC)
	if err != nil {
		return errors.Wrap(err, "failed to create node rpc client")
	}
	node := sources.NewRollupClient(nc)
	c.ctrl = client.NewSequencerControl(exec, node)

	active, err := c.ctrl.SequencerActive(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to get sequencer active status")
	}
	c.seqActive.Store(active)

	return nil
}

func (c *OpConductor) initConsensus(ctx context.Context) error {
	if c.cons != nil {
		return nil
	}

	serverAddr := fmt.Sprintf("%s:%d", c.cfg.ConsensusAddr, c.cfg.ConsensusPort)
	cons, err := consensus.NewRaftConsensus(c.log, c.cfg.RaftServerID, serverAddr, c.cfg.RaftStorageDir, c.cfg.RaftBootstrap, &c.cfg.RollupCfg)
	if err != nil {
		return errors.Wrap(err, "failed to create raft consensus")
	}
	c.cons = cons
	return nil
}

func (c *OpConductor) initHealthMonitor(ctx context.Context) error {
	if c.hmon != nil {
		return nil
	}

	nc, err := opclient.NewRPC(ctx, c.log, c.cfg.NodeRPC)
	if err != nil {
		return errors.Wrap(err, "failed to create node rpc client")
	}
	node := sources.NewRollupClient(nc)

	pc, err := rpc.DialContext(ctx, c.cfg.NodeRPC)
	if err != nil {
		return errors.Wrap(err, "failed to create p2p rpc client")
	}
	p2p := opp2p.NewClient(pc)

	c.hmon = health.NewSequencerHealthMonitor(
		c.log,
		c.cfg.HealthCheck.Interval,
		c.cfg.HealthCheck.SafeInterval,
		c.cfg.HealthCheck.MinPeerCount,
		&c.cfg.RollupCfg,
		node,
		p2p,
	)

	return nil
}

// OpConductor represents a full conductor instance and its resources, it does:
//  1. performs health checks on sequencer
//  2. participate in consensus protocol for leader election
//  3. and control sequencer state based on leader, sequencer health and sequencer active status.
//
// OpConductor has three states:
//  1. running: it is running normally, which executes control loop and participates in leader election.
//  2. paused: control loop (sequencer start/stop) is paused, but it still participates in leader election, and receives health updates.
//  3. stopped: it is stopped, which means it is not participating in leader election and control loop. OpConductor cannot be started again from stopped mode.
type OpConductor struct {
	log     log.Logger
	version string
	cfg     *Config

	ctrl client.SequencerControl
	cons consensus.Consensus
	hmon health.HealthMonitor

	leader    atomic.Bool
	healthy   atomic.Bool
	seqActive atomic.Bool

	actionFn func() // actionFn defines the action to be executed to bring the sequencer to the desired state.

	wg             sync.WaitGroup
	pauseCh        chan struct{}
	pauseDoneCh    chan struct{}
	resumeCh       chan struct{}
	resumeDoneCh   chan struct{}
	actionCh       chan struct{}
	paused         atomic.Bool
	stopped        atomic.Bool
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
}

var _ cliapp.Lifecycle = (*OpConductor)(nil)

// Start implements cliapp.Lifecycle.
func (oc *OpConductor) Start(ctx context.Context) error {
	oc.log.Info("starting OpConductor")

	if err := oc.hmon.Start(); err != nil {
		return errors.Wrap(err, "failed to start health monitor")
	}

	oc.shutdownCtx, oc.shutdownCancel = context.WithCancel(ctx)
	oc.wg.Add(1)
	go oc.loop()

	oc.log.Info("OpConductor started")
	return nil
}

// Stop implements cliapp.Lifecycle.
func (oc *OpConductor) Stop(ctx context.Context) error {
	oc.log.Info("stopping OpConductor")

	var result *multierror.Error

	// close control loop
	oc.shutdownCancel()
	oc.wg.Wait()

	// stop health check
	if err := oc.hmon.Stop(); err != nil {
		result = multierror.Append(result, errors.Wrap(err, "failed to stop health monitor"))
	}

	if err := oc.cons.Shutdown(); err != nil {
		result = multierror.Append(result, errors.Wrap(err, "failed to shutdown consensus"))
	}

	if result.ErrorOrNil() != nil {
		oc.log.Error("failed to stop OpConductor", "err", result.ErrorOrNil())
		return result.ErrorOrNil()
	}

	oc.stopped.Store(true)
	oc.log.Info("OpConductor stopped")
	return nil
}

// Stopped implements cliapp.Lifecycle.
func (oc *OpConductor) Stopped() bool {
	return oc.stopped.Load()
}

// Pause pauses the control loop of OpConductor, but still allows it to participate in leader election.
func (oc *OpConductor) Pause(ctx context.Context) error {
	select {
	case oc.pauseCh <- struct{}{}:
		<-oc.pauseDoneCh
		return nil
	case <-ctx.Done():
		return ErrPauseTimeout
	}
}

// Resume resumes the control loop of OpConductor.
func (oc *OpConductor) Resume(ctx context.Context) error {
	select {
	case oc.resumeCh <- struct{}{}:
		<-oc.resumeDoneCh
		return nil
	case <-ctx.Done():
		return ErrResumeTimeout
	}
}

// Paused returns true if OpConductor is paused.
func (oc *OpConductor) Paused() bool {
	return oc.paused.Load()
}

func (oc *OpConductor) loop() {
	defer oc.wg.Done()
	healthUpdate := oc.hmon.Subscribe()
	leaderUpdate := oc.cons.LeaderCh()

	for {
		select {
		// We process status update (health, leadership) first regardless of the paused state.
		// This way we could properly bring the sequencer to the desired state when resumed.
		case healthy := <-healthUpdate:
			oc.handleHealthUpdate(healthy)
		case leader := <-leaderUpdate:
			oc.handleLeaderUpdate(leader)
		case <-oc.pauseCh:
			oc.paused.Store(true)
			oc.pauseDoneCh <- struct{}{}
		case <-oc.resumeCh:
			oc.paused.Store(false)
			oc.resumeDoneCh <- struct{}{}
			// queue an action to make sure sequencer is in the desired state after resume.
			oc.queueAction()
		case <-oc.shutdownCtx.Done():
			return
		// Handle control action last, so that when executing the action, we have the latest status and bring the sequencer to the desired state.
		case <-oc.actionCh:
			oc.actionFn()
		}
	}
}

func (oc *OpConductor) queueAction() {
	select {
	case oc.actionCh <- struct{}{}:
	default:
		// do nothing if there's an action queued already, this is fine because whenever an action is executed,
		// it is guaranteed to have the latest status and bring the sequencer to the desired state.
	}
}

// handleLeaderUpdate handles leadership update from consensus.
func (oc *OpConductor) handleLeaderUpdate(leader bool) {
	oc.log.Info("Leadership status changed", "server", oc.cons.ServerID(), "leader", leader)

	oc.leader.Store(leader)
	oc.queueAction()
}

// handleHealthUpdate handles health update from health monitor.
func (oc *OpConductor) handleHealthUpdate(healthy bool) {
	if !healthy {
		oc.log.Error("Sequencer is unhealthy", "server", oc.cons.ServerID())
	}

	if healthy != oc.healthy.Load() {
		oc.healthy.Store(healthy)
		oc.queueAction()
	}
}

// action tries to bring the sequencer to the desired state, a retry will be queued if any action failed.
func (oc *OpConductor) action() {
	if oc.Paused() {
		return
	}

	// TODO: (https://github.com/ethereum-optimism/protocol-quest/issues/47) implement
}
