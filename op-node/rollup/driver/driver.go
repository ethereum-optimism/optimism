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
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
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
)

// NewDriver composes an events handler that tracks L1 state, triggers L2 Derivation, and optionally sequences new L2 blocks.
func NewDriver(
	sys event.Registry,
	drain Drain,
	driverCfg *Config,
	cfg *rollup.Config,
	depSet derive.DependencySet,
	l2 L2Chain,
	l1 L1Chain,
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
) *Driver {
	driverCtx, driverCancel := context.WithCancel(context.Background())

	statusTracker := status.NewStatusTracker(log, metrics)
	sys.Register("status", statusTracker)

	l1Tracker := status.NewL1Tracker(l1)

	l1 = metered.NewMeteredL1Fetcher(l1Tracker, metrics)
	verifConfDepth := confdepth.NewConfDepth(driverCfg.VerifierConfDepth, statusTracker.L1Head, l1)

	ec := engine.NewEngineController(driverCtx, l2, log, metrics, cfg, syncCfg, sys.Register("engine-controller", nil))
	// TODO(#17115): Refactor dependency cycles
	ec.SetCrossUpdateHandler(statusTracker)

	engineReset := engine.NewEngineResetDeriver(driverCtx, log, cfg, l1, l2, syncCfg)
	engineReset.SetEngController(ec)
	sys.Register("engine-reset", engineReset)

	clSync := clsync.NewCLSync(log, cfg, metrics, ec) // alt-sync still uses cl-sync state to determine what to sync to
	sys.Register("cl-sync", clSync)

	var finalizer Finalizer
	if cfg.AltDAEnabled() {
		finalizer = finality.NewAltDAFinalizer(driverCtx, log, cfg, l1, altDA, ec)
	} else {
		finalizer = finality.NewFinalizer(driverCtx, log, cfg, l1, ec)
	}
	sys.Register("finalizer", finalizer)

	attrHandler := attributes.NewAttributesHandler(log, cfg, driverCtx, l2, ec)
	sys.Register("attributes-handler", attrHandler)

	derivationPipeline := derive.NewDerivationPipeline(log, cfg, depSet, verifConfDepth, l1Blobs, altDA, l2, metrics, indexingMode)

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
		CLSync:              clSync,
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
		attrBuilder := derive.NewFetchingAttributesBuilder(cfg, depSet, l1, l2)
		sequencerConfDepth := confdepth.NewConfDepth(driverCfg.SequencerConfDepth, statusTracker.L1Head, l1)
		findL1Origin := sequencing.NewL1OriginSelector(driverCtx, log, cfg, sequencerConfDepth)
		sys.Register("origin-selector", findL1Origin)

		// Connect origin selector to the engine controller for force reset notifications
		ec.SetOriginSelectorResetter(findL1Origin)

		sequencer = sequencing.NewSequencer(driverCtx, log, cfg, attrBuilder, findL1Origin,
			sequencerStateListener, sequencerConductor, asyncGossiper, metrics, ec)
		sys.Register("sequencer", sequencer)
	} else {
		sequencer = sequencing.DisabledSequencer{}
	}

	driverEmitter := sys.Register("driver", nil)
	driver := &Driver{
		StatusTracker: statusTracker,
		Finalizer:     finalizer,
		SyncDeriver:   syncDeriver,
		sched:         schedDeriv,
		emitter:       driverEmitter,
		drain:         drain,
		stateReq:      make(chan chan struct{}),
		forceReset:    make(chan chan struct{}, 10),
		driverConfig:  driverCfg,
		driverCtx:     driverCtx,
		driverCancel:  driverCancel,
		log:           log,
		sequencer:     sequencer,
		metrics:       metrics,
		altSync:       altSync,
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

	// Interface to signal the L2 block range to sync.
	altSync AltSync

	sequencer sequencing.SequencerIface

	metrics Metrics
	log     log.Logger

	wg gosync.WaitGroup

	driverCtx    context.Context
	driverCancel context.CancelFunc
}

// Start starts up the state loop.
// The loop will have been started iff err is not nil.
func (s *Driver) Start() error {
	log.Info("Starting driver", "sequencerEnabled", s.driverConfig.SequencerEnabled,
		"sequencerStopped", s.driverConfig.SequencerStopped, "recoverMode", s.driverConfig.RecoverMode)
	if s.driverConfig.SequencerEnabled {
		if s.driverConfig.RecoverMode {
			log.Warn("sequencer is in recover mode")
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

	for {
		if s.driverCtx.Err() != nil { // don't try to schedule/handle more work when we are closing.
			return
		}

		planSequencerAction()

		// If the engine is not ready, or if the L2 head is actively changing, then reset the alt-sync:
		// there is no need to request L2 blocks when we are syncing already.
		if head := s.SyncDeriver.Engine.UnsafeL2Head(); head != lastUnsafeL2 || !s.SyncDeriver.Derivation.DerivationReady() {
			lastUnsafeL2 = head
			altSyncTicker.Reset(syncCheckInterval)
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

// checkForGapInUnsafeQueue checks if there is a gap in the unsafe queue and attempts to retrieve the missing payloads from an alt-sync method.
// WARNING: This is only an outgoing signal, the blocks are not guaranteed to be retrieved.
// Results are received through OnUnsafeL2Payload.
func (s *Driver) checkForGapInUnsafeQueue(ctx context.Context) error {
	start := s.SyncDeriver.Engine.UnsafeL2Head()
	end := s.SyncDeriver.CLSync.LowestQueuedUnsafeBlock()
	// Check if we have missing blocks between the start and end. Request them if we do.
	if end == (eth.L2BlockRef{}) {
		s.log.Debug("requesting sync with open-end range", "start", start)
		return s.altSync.RequestL2Range(ctx, start, eth.L2BlockRef{})
	} else if end.Number > start.Number+1 {
		s.log.Debug("requesting missing unsafe L2 block range", "start", start, "end", end, "size", end.Number-start.Number)
		return s.altSync.RequestL2Range(ctx, start, end)
	}
	return nil
}

func (s *Driver) OnUnsafeL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) {
	s.SyncDeriver.OnUnsafeL2Payload(ctx, payload)
}
