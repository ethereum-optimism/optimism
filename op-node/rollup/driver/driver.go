package driver

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	gosync "sync"

	"github.com/ethereum-optimism/optimism/op-node/metrics/metered"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/confdepth"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/finality"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sequencing"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum/go-ethereum/params"
)

// NewDriver composes an events handler that tracks L1 state, triggers L2 Derivation, and optionally sequences new L2 blocks.
func NewDriver(
	sys event.Registry,
	drain Drain,
	driverCfg *Config,
	cfg *rollup.Config,
	l1ChainConfig *params.ChainConfig,
	depSet derive.DependencySet,
	l2 L2Chain,
	l1 L1Chain,
	upstreamFollowSource UpstreamFollowSource,
	l1Blobs derive.L1BlobsFetcher,
	altSync AltSync,
	network Network,
	log log.Logger,
	metrics Metrics,
	sequencerStateListener sequencing.SequencerStateListener,
	safeHeadListener rollup.SafeHeadListener,
	syncCfg *sync.Config,
	sequencerConductor conductor.SequencerConductor,
	altDA AltDAIface,
	indexingMode bool,
	superAuthority rollup.SuperAuthority,
) *Driver {
	driverCtx, driverCancel := context.WithCancel(context.Background())

	statusTracker := status.NewStatusTracker(log, metrics)
	sys.Register("status", statusTracker)

	l1Tracker := status.NewL1Tracker(l1)

	l1 = metered.NewMeteredL1Fetcher(l1Tracker, metrics)
	verifConfDepth := confdepth.NewConfDepth(driverCfg.VerifierConfDepth, statusTracker.L1Head, l1)

	ec := engine.NewEngineController(driverCtx, l2, log, metrics, cfg, syncCfg, indexingMode, l1, sys.Register("engine-controller", nil), superAuthority)
	// TODO(#17115): Refactor dependency cycles
	ec.SetCrossUpdateHandler(statusTracker)

	var finalizer Finalizer
	if cfg.AltDAEnabled() {
		finalizer = finality.NewAltDAFinalizer(driverCtx, log, cfg, driverCfg.Finalizer, indexingMode, l1, altDA, ec)
	} else {
		finalizer = finality.NewFinalizer(driverCtx, log, cfg, driverCfg.Finalizer, indexingMode, l1, ec)
	}
	sys.Register("finalizer", finalizer)

	attrHandler := attributes.NewAttributesHandler(log, cfg, driverCtx, l2, ec)
	sys.Register("attributes-handler", attrHandler)

	derivationPipeline := derive.NewDerivationPipeline(log, cfg, depSet, verifConfDepth, l1Blobs, altDA, l2, metrics, indexingMode, l1ChainConfig)

	pipelineDeriver := derive.NewPipelineDeriver(driverCtx, derivationPipeline)
	sys.Register("pipeline", pipelineDeriver)

	// Connect components that need force reset notifications to the engine controller
	ec.SetAttributesResetter(attrHandler)
	ec.SetPipelineResetter(pipelineDeriver)

	schedDeriv := NewStepSchedulingDeriver(log)
	sys.Register("step-scheduler", schedDeriv)

	syncDeriver := &SyncDeriver{
		Derivation:          derivationPipeline,
		SafeHeadNotifs:      safeHeadListener,
		Engine:              ec,
		SyncCfg:             syncCfg,
		Config:              cfg,
		L1:                  l1,
		L1Tracker:           l1Tracker,
		L2:                  l2,
		Log:                 log,
		Ctx:                 driverCtx,
		ManagedBySupervisor: indexingMode,
		StepDeriver:         schedDeriv,
	}
	// TODO(#16917) Remove Event System Refactor Comments
	//  Couple SyncDeriver and EngineController for event refactoring
	//  Couple EngDeriver and NewAttributesHandler for event refactoring
	ec.SyncDeriver = syncDeriver
	sys.Register("sync", syncDeriver)
	sys.Register("engine", ec)

	var sequencer sequencing.SequencerIface
	if driverCfg.SequencerEnabled {
		asyncGossiper := async.NewAsyncGossiper(driverCtx, network, log, metrics)
		attrBuilder := derive.NewFetchingAttributesBuilder(cfg, l1ChainConfig, depSet, l1, l2)
		sequencerConfDepth := confdepth.NewConfDepth(driverCfg.SequencerConfDepth, statusTracker.L1Head, l1)
		findL1Origin := sequencing.NewL1OriginSelector(driverCtx, log, cfg, sequencerConfDepth)
		sys.Register("origin-selector", findL1Origin)

		// Connect origin selector to the engine controller for force reset notifications
		ec.SetOriginSelectorResetter(findL1Origin)

		sequencer = sequencing.NewSequencer(driverCtx, log, cfg, driverCfg.SequencerSealingDuration, attrBuilder, findL1Origin,
			sequencerStateListener, sequencerConductor, asyncGossiper, metrics, ec)
		sys.Register("sequencer", sequencer)
	} else {
		sequencer = sequencing.DisabledSequencer{}
	}

	driverEmitter := sys.Register("driver", nil)
	driver := &Driver{
		StatusTracker:        statusTracker,
		Finalizer:            finalizer,
		SyncDeriver:          syncDeriver,
		sched:                schedDeriv,
		emitter:              driverEmitter,
		drain:                drain,
		stateReq:             make(chan chan struct{}),
		forceReset:           make(chan chan struct{}, 10),
		driverConfig:         driverCfg,
		syncConfig:           syncCfg,
		driverCtx:            driverCtx,
		driverCancel:         driverCancel,
		log:                  log,
		sequencer:            sequencer,
		metrics:              metrics,
		altSync:              altSync,
		upstreamFollowSource: upstreamFollowSource,
	}

	return driver
}

type Driver struct {
	StatusTracker SyncStatusTracker
	Finalizer     Finalizer

	SyncDeriver *SyncDeriver

	sched *StepSchedulingDeriver

	emitter event.Emitter
	drain   Drain

	// Requests to block the event loop for synchronous execution to avoid reading an inconsistent state
	stateReq chan chan struct{}

	// Upon receiving a channel in this channel, the derivation pipeline is forced to be reset.
	// It tells the caller that the reset occurred by closing the passed in channel.
	forceReset chan chan struct{}

	// Driver config: verifier and sequencer settings.
	// May not be modified after starting the Driver.
	driverConfig *Config

	syncConfig *sync.Config

	// Interface to signal the L2 block range to sync.
	altSync AltSync

	sequencer sequencing.SequencerIface

	metrics Metrics
	log     log.Logger

	wg gosync.WaitGroup

	driverCtx    context.Context
	driverCancel context.CancelFunc

	upstreamFollowSource UpstreamFollowSource
}

// Start starts up the state loop.
// The loop will have been started iff err is not nil.
func (s *Driver) Start() error {
	s.log.Info("Starting driver", "sequencerEnabled", s.driverConfig.SequencerEnabled,
		"sequencerStopped", s.driverConfig.SequencerStopped, "recoverMode", s.driverConfig.RecoverMode)
	if s.driverConfig.SequencerEnabled {
		if s.driverConfig.RecoverMode {
			s.log.Warn("sequencer is in recover mode")
			s.sequencer.SetRecoverMode(true)
		}
		if err := s.sequencer.SetMaxSafeLag(s.driverCtx, s.driverConfig.SequencerMaxSafeLag); err != nil {
			return fmt.Errorf("failed to set sequencer max safe lag: %w", err)
		}
		if err := s.sequencer.Init(s.driverCtx, !s.driverConfig.SequencerStopped); err != nil {
			return fmt.Errorf("persist initial sequencer state: %w", err)
		}
	}

	s.wg.Add(1)
	go s.eventLoop()

	return nil
}

func (s *Driver) Close() error {
	s.driverCancel()
	s.wg.Wait()
	s.sequencer.Close()
	return nil
}

// the eventLoop responds to L1 changes and internal timers to produce L2 blocks.
func (s *Driver) eventLoop() {
	defer s.wg.Done()
	s.log.Info("State loop started")
	defer s.log.Info("State loop returned")

	defer s.driverCancel()

	// reqStep requests a derivation step nicely, with a delay if this is a reattempt, or not at all if we already scheduled a reattempt.
	reqStep := func() {
		s.sched.RequestStep(s.driverCtx, false)
	}

	// We call reqStep right away to finish syncing to the tip of the chain if we're behind.
	// reqStep will also be triggered when the L1 head moves forward or if there was a reorg on the
	// L1 chain that we need to handle.
	reqStep()

	sequencerTimer := time.NewTimer(0)
	var sequencerCh <-chan time.Time
	var prevTime time.Time
	// planSequencerAction updates the sequencerTimer with the next action, if any.
	// The sequencerCh is nil (indefinitely blocks on read) if no action needs to be performed,
	// or set to the timer channel if there is an action scheduled.
	planSequencerAction := func() {
		nextAction, ok := s.sequencer.NextAction()
		if !ok {
			if sequencerCh != nil {
				s.log.Info("Sequencer paused until new events")
			}
			sequencerCh = nil
			return
		}
		// avoid unnecessary timer resets
		if nextAction == prevTime {
			return
		}
		prevTime = nextAction
		sequencerCh = sequencerTimer.C
		if len(sequencerCh) > 0 { // empty if not already drained before resetting
			<-sequencerCh
		}
		delta := time.Until(nextAction)
		s.log.Info("Scheduled sequencer action", "delta", delta)
		sequencerTimer.Reset(delta)
	}

	// Create a ticker to check if there is a gap in the engine queue. Whenever
	// there is, we send requests to sync source to retrieve the missing payloads.
	syncCheckInterval := time.Duration(s.SyncDeriver.Config.BlockTime) * time.Second * 2
	altSyncTicker := time.NewTicker(syncCheckInterval)
	defer altSyncTicker.Stop()

	lastUnsafeL2 := s.SyncDeriver.Engine.UnsafeL2Head()

	followSource := s.SyncDeriver.SyncCfg.FollowSourceEnabled()

	resetAltSync := func(newHead eth.L2BlockRef, derivationReady bool) {
		s.log.Debug(
			"altSyncTicker reset",
			"head", newHead,
			"lastUnsafeL2", lastUnsafeL2,
			"derivationReady", derivationReady,
			"followSource", followSource,
		)
		lastUnsafeL2 = newHead
		altSyncTicker.Reset(syncCheckInterval)
	}

	// upstreamSyncTickerC drives the upstreamSyncTicker, which periodically reconciles
	// the state against upstream sources when derivation is disabled (unsafeOnly).
	//
	// In this mode, the node does not derive from L1; instead, it uses L1 as a mandatory
	// upstream anchor for its unsafe head, and imports safe/finalized state
	// from an external source. Since the normal derivation pipeline is inactive, reorg
	// detection must be performed here instead.
	var upstreamSyncTickerC <-chan time.Time
	if followSource {
		upstreamSyncTickerCheckInterval := time.Duration(s.SyncDeriver.Config.BlockTime) * time.Second * 2
		upstreamSyncTicker := time.NewTicker(upstreamSyncTickerCheckInterval)
		upstreamSyncTickerC = upstreamSyncTicker.C
		defer upstreamSyncTicker.Stop()
	}

	for {
		if s.driverCtx.Err() != nil { // don't try to schedule/handle more work when we are closing.
			return
		}

		planSequencerAction()

		head := s.SyncDeriver.Engine.UnsafeL2Head()
		derivationReady := s.SyncDeriver.Derivation.DerivationReady()

		if lastUnsafeL2 != head {
			// Unsafe head changed: reset alt-sync to avoid redundant L2 requests while syncing.
			resetAltSync(head, derivationReady)
		} else if !followSource && !derivationReady {
			// Derivation enabled but not yet ready: reset alt-sync while it catches up.
			resetAltSync(head, derivationReady)
		}

		select {
		case <-sequencerCh:
			s.emitter.Emit(s.driverCtx, sequencing.SequencerActionEvent{})
		case <-altSyncTicker.C:
			// Check if there is a gap in the current unsafe payload queue.
			ctx, cancel := context.WithTimeout(s.driverCtx, time.Second*2)
			err := s.checkForGapInUnsafeQueue(ctx)
			cancel()
			if err != nil {
				s.log.Warn("failed to check for unsafe L2 blocks to sync", "err", err)
			}
		case <-upstreamSyncTickerC:
			s.followUpstream()
		case <-s.sched.NextDelayedStep():
			s.sched.AttemptStep(s.driverCtx)
		case <-s.sched.NextStep():
			s.sched.AttemptStep(s.driverCtx)
		case respCh := <-s.stateReq:
			respCh <- struct{}{}
		case respCh := <-s.forceReset:
			s.log.Warn("Derivation pipeline is manually reset")
			s.SyncDeriver.Derivation.Reset()
			s.metrics.RecordPipelineReset()
			close(respCh)
		case <-s.drain.Await():
			if err := s.drain.Drain(); err != nil {
				if s.driverCtx.Err() != nil {
					return
				} else {
					s.log.Error("unexpected error from event-draining", "err", err)
					s.emitter.Emit(s.driverCtx, rollup.CriticalErrorEvent{
						Err: fmt.Errorf("unexpected error: %w", err),
					})
				}
			}
		case <-s.driverCtx.Done():
			return
		}
	}
}

// ResetDerivationPipeline forces a reset of the derivation pipeline.
// It waits for the reset to occur. It simply unblocks the caller rather
// than fully cancelling the reset request upon a context cancellation.
func (s *Driver) ResetDerivationPipeline(ctx context.Context) error {
	respCh := make(chan struct{}, 1)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.forceReset <- respCh:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-respCh:
			return nil
		}
	}
}

func (s *Driver) StartSequencer(ctx context.Context, blockHash common.Hash) error {
	return s.sequencer.Start(ctx, blockHash)
}

func (s *Driver) StopSequencer(ctx context.Context) (common.Hash, error) {
	return s.sequencer.Stop(ctx)
}

func (s *Driver) SequencerActive(ctx context.Context) (bool, error) {
	return s.sequencer.Active(), nil
}

func (s *Driver) OverrideLeader(ctx context.Context) error {
	return s.sequencer.OverrideLeader(ctx)
}

func (s *Driver) ConductorEnabled(ctx context.Context) (bool, error) {
	return s.sequencer.ConductorEnabled(ctx), nil
}

func (s *Driver) SetRecoverMode(ctx context.Context, mode bool) error {
	s.sequencer.SetRecoverMode(mode)
	return nil
}

// SyncStatus blocks the driver event loop and captures the syncing status.
func (s *Driver) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return s.StatusTracker.SyncStatus(), nil
}

// BlockRefWithStatus blocks the driver event loop and captures the syncing status,
// along with an L2 block reference by number consistent with that same status.
// If the event loop is too busy and the context expires, a context error is returned.
func (s *Driver) BlockRefWithStatus(ctx context.Context, num uint64) (eth.L2BlockRef, *eth.SyncStatus, error) {
	resp := s.StatusTracker.SyncStatus()
	if resp.FinalizedL2.Number >= num { // If finalized, we are certain it does not reorg, and don't have to lock.
		ref, err := s.SyncDeriver.L2.L2BlockRefByNumber(ctx, num)
		return ref, resp, err
	}
	wait := make(chan struct{})
	select {
	case s.stateReq <- wait:
		resp := s.StatusTracker.SyncStatus()
		ref, err := s.SyncDeriver.L2.L2BlockRefByNumber(ctx, num)
		<-wait
		return ref, resp, err
	case <-ctx.Done():
		return eth.L2BlockRef{}, nil, ctx.Err()
	}
}

// checkForGapInUnsafeQueue checks if there is a gap in the unsafe queue and attempts to retrieve the missing payloads
func (s *Driver) checkForGapInUnsafeQueue(ctx context.Context) error {
	start := s.SyncDeriver.Engine.UnsafeL2Head()
	payload, end := s.SyncDeriver.Engine.PeekUnsafePayload()

	if s.syncConfig.SyncModeReqResp {
		if end == (eth.L2BlockRef{}) {
			s.log.Debug("requesting rrsync with open-end range", "start", start)
			return s.altSync.RequestL2Range(ctx, start, eth.L2BlockRef{})
		} else if end.Number > start.Number+1 {
			s.log.Debug("requesting rrsync missing unsafe L2 block range", "start", start, "end", end, "size", end.Number-start.Number)
			return s.altSync.RequestL2Range(ctx, start, end)
		}
	} else {
		if end == (eth.L2BlockRef{}) {
			s.log.Debug("checkForGapInUnsafeQueue: no unsafe payload in queue", "start", start)
			return nil
		} else if end.Number > start.Number+1 {
			s.log.Info("requesting engine missing unsafe L2 block range", "start", start, "end", end, "size", end.Number-start.Number)
			err := s.SyncDeriver.Engine.InsertUnsafePayload(ctx, payload, end)
			if err != nil {
				s.log.Error("failed to insert unsafe payload", "err", err)
			}
			return err
		}
	}

	return nil
}

func (s *Driver) OnUnsafeL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
	s.SyncDeriver.OnUnsafeL2Payload(ctx, payload)
}

// followUpstream reconciles the local engine state with upstream sources when
// derivation is disabled (UnsafeOnly).
//
// In this mode, the driver does not derive L2 from L1. Instead, it:
// Uses the followTracker to fetch external safe / finalized / CurrentL1,
// validates that the external state is sane (e.g. finalized is not ahead
// of safe), and then updates the engine via FollowSource.
//
// This function is intended to be called periodically by a ticker and is a
// no-op while derivation is enabled or the EL is still performing its initial
// sync.
func (s *Driver) followUpstream() {
	if !s.syncConfig.FollowSourceEnabled() {
		return
	}
	if s.SyncDeriver.Engine.IsEngineInitialELSyncing() {
		// Do not interfere with initial EL Sync and wait until it is done
		return
	}
	status, err := s.upstreamFollowSource.GetFollowStatus(s.driverCtx)
	if err != nil {
		s.log.Warn("Follow Upstream: Failed to fetch status", "err", err)
		return
	}
	s.log.Info("Follow Upstream", "eSafe", status.SafeL2, "eFinalized", status.FinalizedL2, "eCurrentL1", status.CurrentL1)
	if status.FinalizedL2.Number > status.SafeL2.Number {
		s.log.Warn("Follow Upstream: Invalid external state, finalized is ahead of safe", "safe", status.SafeL2.Number, "finalized", status.FinalizedL2.Number)
		return
	}

	eSafeL1Origin, err := s.upstreamFollowSource.L1BlockRefByNumber(s.driverCtx, status.SafeL2.L1Origin.Number)
	if err != nil {
		s.log.Warn("Follow Upstream: Failed to look up L1 origin of external safe head", "err", err)
		return
	}
	if eSafeL1Origin.Hash != status.SafeL2.L1Origin.Hash {
		s.log.Warn(
			"Follow Upstream: Invalid external safe: L1 origin of external safe head mismatch",
			"actual", eSafeL1Origin,
			"expected", status.SafeL2.L1Origin,
		)
		return
	}

	eFinalizedL1Origin, err := s.upstreamFollowSource.L1BlockRefByNumber(s.driverCtx, status.FinalizedL2.L1Origin.Number)
	if err != nil {
		s.log.Warn("Follow Upstream: Failed to look up L1 origin of external finalized head", "err", err)
		return
	}
	if eFinalizedL1Origin.Hash != status.FinalizedL2.L1Origin.Hash {
		s.log.Warn(
			"Follow Upstream: Invalid external finalized: L1 origin of external finalized head mismatch",
			"actual", eFinalizedL1Origin,
			"expected", status.FinalizedL2.L1Origin,
		)
		return
	}

	if (status.CurrentL1 == eth.L1BlockRef{}) {
		s.log.Debug("Follow Upstream: CurrentL1 not available")
	} else {
		eCurrentL1, err := s.upstreamFollowSource.L1BlockRefByNumber(s.driverCtx, status.CurrentL1.Number)
		if err != nil {
			s.log.Warn("Follow Upstream: Failed to look up external currentL1", "err", err)
			return
		}
		if eCurrentL1.Hash != status.CurrentL1.Hash {
			s.log.Warn(
				"Follow Upstream: Invalid external CurrentL1: L1 head mismatch",
				"actual", eCurrentL1,
				"expected", status.CurrentL1,
			)
			return
		}

		s.log.Debug("Follow Upstream: Inject L1 Info", "currentL1", status.CurrentL1)
		s.emitter.Emit(s.driverCtx, derive.DeriverL1StatusEvent{Origin: status.CurrentL1})
	}
	// Only reach this point if all L1 checks passed
	s.SyncDeriver.Engine.FollowSource(status.SafeL2, status.FinalizedL2)
}
