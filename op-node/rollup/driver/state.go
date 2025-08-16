package driver

import (
	"context"
	"errors"
	"fmt"
	gosync "sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/clsync"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sequencing"
	"github.com/ethereum-optimism/optimism/op-node/rollup/status"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// Deprecated: use eth.SyncStatus instead.
type SyncStatus = eth.SyncStatus

type Driver struct {
	StatusTracker SyncStatusTracker
	Finalizer     Finalizer

	*SyncDeriver

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
	syncCheckInterval := time.Duration(s.Config.BlockTime) * time.Second * 2
	altSyncTicker := time.NewTicker(syncCheckInterval)
	defer altSyncTicker.Stop()
	lastUnsafeL2 := s.Engine.UnsafeL2Head()

	for {
		if s.driverCtx.Err() != nil { // don't try to schedule/handle more work when we are closing.
			return
		}

		planSequencerAction()

		// If the engine is not ready, or if the L2 head is actively changing, then reset the alt-sync:
		// there is no need to request L2 blocks when we are syncing already.
		if head := s.Engine.UnsafeL2Head(); head != lastUnsafeL2 || !s.Derivation.DerivationReady() {
			lastUnsafeL2 = head
			altSyncTicker.Reset(syncCheckInterval)
		}

		select {
		case <-sequencerCh:
			s.Emitter.Emit(s.driverCtx, sequencing.SequencerActionEvent{})
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
			s.Derivation.Reset()
			s.metrics.RecordPipelineReset()
			close(respCh)
		case <-s.drain.Await():
			if err := s.drain.Drain(); err != nil {
				if s.driverCtx.Err() != nil {
					return
				} else {
					s.log.Error("unexpected error from event-draining", "err", err)
					s.Emitter.Emit(s.driverCtx, rollup.CriticalErrorEvent{
						Err: fmt.Errorf("unexpected error: %w", err),
					})
				}
			}
		case <-s.driverCtx.Done():
			return
		}
	}
}

type SyncDeriver struct {
	// The derivation pipeline is reset whenever we reorg.
	// The derivation pipeline determines the new l2Safe.
	Derivation DerivationPipeline

	SafeHeadNotifs rollup.SafeHeadListener // notified when safe head is updated

	CLSync CLSync

	// The engine controller is used by the sequencer & Derivation components.
	// We will also use it for EL sync in a future PR.
	Engine *engine.EngineController

	// Sync Mod Config
	SyncCfg *sync.Config

	Config *rollup.Config

	L1 L1Chain
	// Track L1 view when new unsafe L1 block is observed
	L1Tracker *status.L1Tracker
	L2        L2Chain

	Emitter event.Emitter

	Log log.Logger

	Ctx context.Context

	// When in interop, and managed by an op-supervisor,
	// the node performs a reset based on the instructions of the op-supervisor.
	ManagedBySupervisor bool

	StepDeriver StepDeriver
}

func (s *SyncDeriver) AttachEmitter(em event.Emitter) {
	s.Emitter = em
}

func (s *SyncDeriver) OnL1Unsafe(ctx context.Context) {
	// a new L1 head may mean we have the data to not get an EOF again.
	s.StepDeriver.RequestStep(ctx, false)
}

func (s *SyncDeriver) OnL1Finalized(ctx context.Context) {
	// On "safe" L1 blocks: no step, justified L1 information does not do anything for L2 derivation or status.
	// On "finalized" L1 blocks: we may be able to mark more L2 data as finalized now.
	s.StepDeriver.RequestStep(ctx, false)
}

func (s *SyncDeriver) OnEvent(ctx context.Context, ev event.Event) bool {
	// TODO(#16917) Remove Event System Refactor Comments
	//  ELSyncStartedEvent is removed and OnELSyncStarted is synchronously called at EngineController
	//  ReceivedBlockEvent is removed and OnUnsafeL2Payload is synchronously called at NewBlockReceiver
	//  L1UnsafeEvent is removed and OnL1Unsafe is synchronously called at L1Handler
	//  FinalizeL1Event is removed and OnL1Finalized is synchronously called at L1Handler
	switch x := ev.(type) {
	case StepEvent:
		s.SyncStep()
	case rollup.ResetEvent:
		s.onResetEvent(ctx, x)
	case rollup.L1TemporaryErrorEvent:
		s.Log.Warn("L1 temporary error", "err", x.Err)
		s.StepDeriver.RequestStep(ctx, false)
	case rollup.EngineTemporaryErrorEvent:
		s.Log.Warn("Engine temporary error", "err", x.Err)
		// Make sure that for any temporarily failed attributes we retry processing.
		// This will be triggered by a step. After appropriate backoff.
		s.StepDeriver.RequestStep(ctx, false)
	case engine.EngineResetConfirmedEvent:
		s.onEngineConfirmedReset(ctx, x)
	case derive.DeriverIdleEvent:
		// Once derivation is idle the system is healthy
		// and we can wait for new inputs. No backoff necessary.
		s.StepDeriver.ResetStepBackoff(ctx)
	case derive.DeriverMoreEvent:
		// If there is more data to process,
		// continue derivation quickly
		s.StepDeriver.RequestStep(ctx, true)
	case engine.SafeDerivedEvent:
		s.onSafeDerivedBlock(ctx, x)
	case derive.ProvideL1Traversal:
		s.StepDeriver.RequestStep(ctx, false)
	default:
		return false
	}
	return true
}

func (s *SyncDeriver) OnUnsafeL2Payload(ctx context.Context, envelope *eth.ExecutionPayloadEnvelope) {
	// If we are doing CL sync or done with engine syncing, fallback to the unsafe payload queue & CL P2P sync.
	if s.SyncCfg.SyncMode == sync.CLSync || !s.Engine.IsEngineSyncing() {
		s.Log.Info("Optimistically queueing unsafe L2 execution payload", "id", envelope.ExecutionPayload.ID())
		s.Emitter.Emit(ctx, clsync.ReceivedUnsafePayloadEvent{Envelope: envelope})
	} else if s.SyncCfg.SyncMode == sync.ELSync {
		ref, err := derive.PayloadToBlockRef(s.Config, envelope.ExecutionPayload)
		if err != nil {
			s.Log.Info("Failed to turn execution payload into a block ref", "id", envelope.ExecutionPayload.ID(), "err", err)
			return
		}
		if ref.Number <= s.Engine.UnsafeL2Head().Number {
			return
		}
		s.Log.Info("Optimistically inserting unsafe L2 execution payload to drive EL sync", "id", envelope.ExecutionPayload.ID())
		if err := s.Engine.InsertUnsafePayload(s.Ctx, envelope, ref); err != nil {
			s.Log.Warn("Failed to insert unsafe payload for EL sync", "id", envelope.ExecutionPayload.ID(), "err", err)
		}
	}
}

func (s *SyncDeriver) onSafeDerivedBlock(ctx context.Context, x engine.SafeDerivedEvent) {
	if s.SafeHeadNotifs != nil && s.SafeHeadNotifs.Enabled() {
		if err := s.SafeHeadNotifs.SafeHeadUpdated(x.Safe, x.Source.ID()); err != nil {
			// At this point our state is in a potentially inconsistent state as we've updated the safe head
			// in the execution client but failed to post process it. Reset the pipeline so the safe head rolls back
			// a little (it always rolls back at least 1 block) and then it will retry storing the entry
			s.Emitter.Emit(ctx, rollup.ResetEvent{
				Err: fmt.Errorf("safe head notifications failed: %w", err),
			})
		}
	}
}

func (s *SyncDeriver) OnELSyncStarted() {
	// The EL sync may progress the safe head in the EL without deriving those blocks from L1
	// which means the safe head db will miss entries so we need to remove all entries to avoid returning bad data
	s.Log.Warn("Clearing safe head db because EL sync started")
	if s.SafeHeadNotifs != nil {
		if err := s.SafeHeadNotifs.SafeHeadReset(eth.L2BlockRef{}); err != nil {
			s.Log.Error("Failed to notify safe-head reset when optimistically syncing")
		}
	}
}

func (s *SyncDeriver) onEngineConfirmedReset(ctx context.Context, x engine.EngineResetConfirmedEvent) {
	// If the listener update fails, we return,
	// and don't confirm the engine-reset with the derivation pipeline.
	// The pipeline will re-trigger a reset as necessary.
	if s.SafeHeadNotifs != nil {
		if err := s.SafeHeadNotifs.SafeHeadReset(x.CrossSafe); err != nil {
			s.Log.Error("Failed to warn safe-head notifier of safe-head reset", "safe", x.CrossSafe)
			return
		}
		if s.SafeHeadNotifs.Enabled() && x.CrossSafe.ID() == s.Config.Genesis.L2 {
			// The rollup genesis block is always safe by definition. So if the pipeline resets this far back we know
			// we will process all safe head updates and can record genesis as always safe from L1 genesis.
			// Note that it is not safe to use cfg.Genesis.L1 here as it is the block immediately before the L2 genesis
			// but the contracts may have been deployed earlier than that, allowing creating a dispute game
			// with a L1 head prior to cfg.Genesis.L1
			l1Genesis, err := s.L1.L1BlockRefByNumber(s.Ctx, 0)
			if err != nil {
				s.Log.Error("Failed to retrieve L1 genesis, cannot notify genesis as safe block", "err", err)
				return
			}
			if err := s.SafeHeadNotifs.SafeHeadUpdated(x.CrossSafe, l1Genesis.ID()); err != nil {
				s.Log.Error("Failed to notify safe-head listener of safe-head", "err", err)
				return
			}
		}
	}
	s.Log.Info("Confirming pipeline reset")
	s.Emitter.Emit(ctx, derive.ConfirmPipelineResetEvent{})
}

func (s *SyncDeriver) onResetEvent(ctx context.Context, x rollup.ResetEvent) {
	if s.ManagedBySupervisor {
		s.Log.Warn("Encountered reset when managed by op-supervisor, waiting for op-supervisor", "err", x.Err)
		// IndexingMode will pick up the ResetEvent
		return
	}
	// If the system corrupts, e.g. due to a reorg, simply reset it
	s.Log.Warn("Deriver system is resetting", "err", x.Err)
	s.Emitter.Emit(ctx, engine.ResetEngineRequestEvent{})
	s.StepDeriver.RequestStep(ctx, false)
}

func (s *SyncDeriver) tryBackupUnsafeReorg() {
	// If we don't need to call FCU to restore unsafeHead using backupUnsafe, keep going b/c
	// this was a no-op(except correcting invalid state when backupUnsafe is empty but TryBackupUnsafeReorg called).
	fcuCalled, err := s.Engine.TryBackupUnsafeReorg(s.Ctx)
	// Dealing with legacy here: it used to skip over the error-handling if fcuCalled was false.
	// But that combination is not actually a code-path in TryBackupUnsafeReorg.
	// We should drop fcuCalled, and make the function emit events directly,
	// once there are no more synchronous callers.
	if !fcuCalled && err != nil {
		s.Log.Crit("unexpected TryBackupUnsafeReorg error after no FCU call", "err", err)
	}
	if err != nil {
		// If we needed to perform a network call, then we should yield even if we did not encounter an error.
		if errors.Is(err, derive.ErrReset) {
			s.Emitter.Emit(s.Ctx, rollup.ResetEvent{Err: err})
		} else if errors.Is(err, derive.ErrTemporary) {
			s.Emitter.Emit(s.Ctx, rollup.EngineTemporaryErrorEvent{Err: err})
		} else {
			s.Emitter.Emit(s.Ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected TryBackupUnsafeReorg error type: %w", err),
			})
		}
	}
}

// SyncStep performs the sequence of encapsulated syncing steps.
// Warning: this sequence will be broken apart as outlined in op-node derivers design doc.
func (s *SyncDeriver) SyncStep() {
	s.Log.Debug("Sync process step")

	s.tryBackupUnsafeReorg()

	s.Engine.TryUpdateEngine(s.Ctx)

	if s.Engine.IsEngineSyncing() {
		// The pipeline cannot move forwards if doing EL sync.
		s.Log.Debug("Rollup driver is backing off because execution engine is syncing.",
			"unsafe_head", s.Engine.UnsafeL2Head())
		s.StepDeriver.ResetStepBackoff(s.Ctx)
		return
	}

	// Any now processed forkchoice updates will trigger CL-sync payload processing, if any payload is queued up.

	// Since we don't force attributes to be processed at this point,
	// we cannot safely directly trigger the derivation, as that may generate new attributes that
	// conflict with what attributes have not been applied yet.
	// Instead, we request the engine to repeat where its pending-safe head is at.
	// Upon the pending-safe signal the attributes deriver can then ask the pipeline
	// to generate new attributes, if no attributes are known already.
	s.Emitter.Emit(s.Ctx, engine.PendingSafeRequestEvent{})

	// If interop is configured, we have to run the engine events,
	// to ensure cross-L2 safety is continuously verified against the interop-backend.
	if s.Config.InteropTime != nil && !s.ManagedBySupervisor {
		s.Emitter.Emit(s.Ctx, engine.CrossUpdateRequestEvent{})
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
		ref, err := s.L2.L2BlockRefByNumber(ctx, num)
		return ref, resp, err
	}
	wait := make(chan struct{})
	select {
	case s.stateReq <- wait:
		resp := s.StatusTracker.SyncStatus()
		ref, err := s.L2.L2BlockRefByNumber(ctx, num)
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
	start := s.Engine.UnsafeL2Head()
	end := s.CLSync.LowestQueuedUnsafeBlock()
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
