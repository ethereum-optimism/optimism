package conductor

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/raft"
	"github.com/pkg/errors"

	"github.com/ethereum-optimism/optimism/op-conductor/client"
	"github.com/ethereum-optimism/optimism/op-conductor/consensus"
	"github.com/ethereum-optimism/optimism/op-conductor/health"
	conductorrpc "github.com/ethereum-optimism/optimism/op-conductor/rpc"
	opp2p "github.com/ethereum-optimism/optimism/op-node/p2p"
	"github.com/ethereum-optimism/optimism/op-service/cliapp"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
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

	// do not rely on the default context, use a dedicated context for shutdown.
	oc.shutdownCtx, oc.shutdownCancel = context.WithCancel(context.Background())

	err := oc.init(ctx)
	if err != nil {
		log.Error("failed to initialize OpConductor", "err", err)
		// ensure we always close the resources if we fail to initialize the conductor.
		closeErr := oc.Stop(ctx)
		if closeErr != nil {
			err = multierror.Append(err, closeErr)
		}
		return nil, err
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
	if err := c.initRPCServer(ctx); err != nil {
		return errors.Wrap(err, "failed to initialize rpc server")
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
	c.leaderUpdateCh = c.cons.LeaderCh()
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
	c.healthUpdateCh = c.hmon.Subscribe()

	return nil
}

func (oc *OpConductor) initRPCServer(ctx context.Context) error {
	server := oprpc.NewServer(
		oc.cfg.RPC.ListenAddr,
		oc.cfg.RPC.ListenPort,
		oc.version,
		oprpc.WithLogger(oc.log),
	)
	api := conductorrpc.NewAPIBackend(oc.log, oc)
	server.AddAPI(rpc.API{
		Namespace: conductorrpc.RPCNamespace,
		Version:   oc.version,
		Service:   api,
	})
	oc.rpcServer = server
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

	healthUpdateCh <-chan bool
	leaderUpdateCh <-chan bool
	actionFn       func() // actionFn defines the action to be executed to bring the sequencer to the desired state.

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

	rpcServer *oprpc.Server
}

var _ cliapp.Lifecycle = (*OpConductor)(nil)

// Start implements cliapp.Lifecycle.
func (oc *OpConductor) Start(ctx context.Context) error {
	oc.log.Info("starting OpConductor")

	if err := oc.hmon.Start(); err != nil {
		return errors.Wrap(err, "failed to start health monitor")
	}

	oc.log.Info("starting JSON-RPC server")
	if err := oc.rpcServer.Start(); err != nil {
		return errors.Wrap(err, "failed to start JSON-RPC server")
	}

	oc.wg.Add(1)
	go oc.loop()

	oc.log.Info("OpConductor started")
	return nil
}

// Stop implements cliapp.Lifecycle.
func (oc *OpConductor) Stop(ctx context.Context) error {
	if oc.Stopped() {
		oc.log.Info("OpConductor already stopped")
		return nil
	}

	oc.log.Info("stopping OpConductor")
	var result *multierror.Error

	// close control loop
	oc.shutdownCancel()
	oc.wg.Wait()

	if oc.rpcServer != nil {
		if err := oc.rpcServer.Stop(); err != nil {
			result = multierror.Append(result, errors.Wrap(err, "failed to stop rpc server"))
		}
	}

	// stop health check
	if oc.hmon != nil {
		if err := oc.hmon.Stop(); err != nil {
			result = multierror.Append(result, errors.Wrap(err, "failed to stop health monitor"))
		}
	}

	if oc.cons != nil {
		if err := oc.cons.Shutdown(); err != nil {
			result = multierror.Append(result, errors.Wrap(err, "failed to shutdown consensus"))
		}
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

func (oc *OpConductor) HTTPEndpoint() string {
	if oc.rpcServer == nil {
		return ""
	}
	return fmt.Sprintf("http://%s", oc.rpcServer.Endpoint())
}

// Leader returns true if OpConductor is the leader.
func (oc *OpConductor) Leader(_ context.Context) bool {
	return oc.cons.Leader()
}

// LeaderWithID returns the current leader's server ID and address.
func (oc *OpConductor) LeaderWithID(_ context.Context) (string, string) {
	return oc.cons.LeaderWithID()
}

// AddServerAsVoter adds a server as a voter to the cluster.
func (oc *OpConductor) AddServerAsVoter(_ context.Context, id string, addr string) error {
	return oc.cons.AddVoter(id, addr)
}

// AddServerAsNonvoter adds a server as a non-voter to the cluster. non-voter will not participate in leader election.
func (oc *OpConductor) AddServerAsNonvoter(_ context.Context, id string, addr string) error {
	return oc.cons.AddNonVoter(id, addr)
}

// RemoveServer removes a server from the cluster.
func (oc *OpConductor) RemoveServer(_ context.Context, id string) error {
	return oc.cons.RemoveServer(id)
}

// TransferLeader transfers leadership to another server.
func (oc *OpConductor) TransferLeader(_ context.Context) error {
	return oc.cons.TransferLeader()
}

// TransferLeaderToServer transfers leadership to a specific server.
func (oc *OpConductor) TransferLeaderToServer(_ context.Context, id string, addr string) error {
	return oc.cons.TransferLeaderTo(id, addr)
}

// CommitUnsafePayload commits a unsafe payload (lastest head) to the cluster FSM.
func (oc *OpConductor) CommitUnsafePayload(_ context.Context, payload *eth.ExecutionPayload) error {
	return oc.cons.CommitUnsafePayload(payload)
}

// SequencerHealthy returns true if sequencer is healthy.
func (oc *OpConductor) SequencerHealthy(_ context.Context) bool {
	return oc.healthy.Load()
}

func (oc *OpConductor) loop() {
	defer oc.wg.Done()

	for {
		select {
		// We process status update (health, leadership) first regardless of the paused state.
		// This way we could properly bring the sequencer to the desired state when resumed.
		case healthy := <-oc.healthUpdateCh:
			oc.handleHealthUpdate(healthy)
		case leader := <-oc.leaderUpdateCh:
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

	var err error
	// exhaust all cases below for completeness, 3 state, 8 cases.
	switch status := struct{ leader, healthy, active bool }{oc.leader.Load(), oc.healthy.Load(), oc.seqActive.Load()}; {
	case !status.leader && !status.healthy && !status.active:
		// if follower is not healthy and not sequencing, just log an error
		oc.log.Error("server (follower) is not healthy", "server", oc.cons.ServerID())
	case !status.leader && !status.healthy && status.active:
		// sequencer is not leader, not healthy, but it is sequencing, stop it
		err = oc.stopSequencer()
	case !status.leader && status.healthy && !status.active:
		// normal follower, do nothing
	case !status.leader && status.healthy && status.active:
		// stop sequencer, this happens when current server steps down as leader.
		err = oc.stopSequencer()
	case status.leader && !status.healthy && !status.active:
		// transfer leadership to another node
		err = oc.transferLeader()
	case status.leader && !status.healthy && status.active:
		var result *multierror.Error
		// Try to stop sequencer first, but since sequencer is not healthy, we may not be able to stop it.
		// In this case, it's fine to continue to try to transfer leadership to another server. This is safe because
		// 1. if leadership transfer succeeded, then we'll retry and enter case !status.leader && status.healthy && status.active, which will try to stop sequencer.
		// 2. even if the retry continues to fail and current server stays in active sequencing mode, it would be safe because our hook in op-node will prevent it from committing any new blocks to the network via p2p (if it's not leader any more)
		if e := oc.stopSequencer(); e != nil {
			result = multierror.Append(result, e)
		}
		// try to transfer leadership to another server despite if sequencer is stopped or not. There are 4 scenarios here:
		// 1. [sequencer stopped, leadership transfer succeeded] which is the happy case and we handed over sequencing to another server.
		// 2. [sequencer stopped, leadership transfer failed] we'll enter into case status.leader && !status.healthy && !status.active and retry transfer leadership.
		// 3. [sequencer active, leadership transfer succeeded] we'll enter into case !status.leader && status.healthy && status.active and retry stop sequencer.
		// 4. [sequencer active, leadership transfer failed] we're in the same state and will retry here again.
		if e := oc.transferLeader(); e != nil {
			result = multierror.Append(result, e)
		}
		err = result.ErrorOrNil()
	case status.leader && status.healthy && !status.active:
		// start sequencer
		err = oc.startSequencer()
	case status.leader && status.healthy && status.active:
		// normal leader, do nothing
	}

	if err != nil {
		oc.log.Error("failed to execute step, queueing another one to retry", "err", err)
		// randomly sleep for 0-200ms to avoid excessive retry
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
		oc.queueAction()
	}
}

// transferLeader tries to transfer leadership to another server.
func (oc *OpConductor) transferLeader() error {
	// TransferLeader here will do round robin to try to transfer leadership to the next healthy node.
	err := oc.cons.TransferLeader()
	if err == nil {
		oc.leader.Store(false)
		return nil // success
	}

	switch {
	case errors.Is(err, raft.ErrNotLeader):
		// This node is not the leader, do nothing.
		oc.log.Warn("cannot transfer leadership since current server is not the leader")
		return nil
	default:
		oc.log.Error("failed to transfer leadership", "err", err)
		return err
	}
}

func (oc *OpConductor) stopSequencer() error {
	oc.log.Info("stopping sequencer", "server", oc.cons.ServerID(), "leader", oc.leader.Load(), "healthy", oc.healthy.Load(), "active", oc.seqActive.Load())

	if _, err := oc.ctrl.StopSequencer(context.Background()); err != nil {
		return errors.Wrap(err, "failed to stop sequencer")
	}
	oc.seqActive.Store(false)
	return nil
}

func (oc *OpConductor) startSequencer() error {
	oc.log.Info("starting sequencer", "server", oc.cons.ServerID(), "leader", oc.leader.Load(), "healthy", oc.healthy.Load(), "active", oc.seqActive.Load())

	// When starting sequencer, we need to make sure that the current node has the latest unsafe head from the consensus protocol
	// If not, then we wait for the unsafe head to catch up or gossip it to op-node manually from op-conductor.
	unsafeInCons := oc.cons.LatestUnsafePayload()
	if unsafeInCons == nil {
		return errors.New("failed to get latest unsafe block from consensus")
	}
	unsafeInNode, err := oc.ctrl.LatestUnsafeBlock(context.Background())
	if err != nil {
		return errors.Wrap(err, "failed to get latest unsafe block from EL during startSequencer phase")
	}

	if unsafeInCons.BlockHash != unsafeInNode.Hash() {
		oc.log.Warn(
			"latest unsafe block in consensus is not the same as the one in op-node",
			"consensus_hash", unsafeInCons.BlockHash,
			"consensus_block_num", unsafeInCons.BlockNumber,
			"node_hash", unsafeInNode.Hash(),
			"node_block_num", unsafeInNode.NumberU64(),
		)

		if uint64(unsafeInCons.BlockNumber)-unsafeInNode.NumberU64() == 1 {
			// tries to post the unsafe head to op-node when head is only 1 block behind (most likely due to gossip delay)
			if err = oc.ctrl.PostUnsafePayload(context.Background(), unsafeInCons); err != nil {
				oc.log.Error("failed to post unsafe head payload to op-node", "err", err)
			}
		}
		return ErrUnsafeHeadMismarch // return error to allow retry
	}

	if err := oc.ctrl.StartSequencer(context.Background(), unsafeInCons.BlockHash); err != nil {
		return errors.Wrap(err, "failed to start sequencer")
	}

	oc.seqActive.Store(true)
	return nil
}
