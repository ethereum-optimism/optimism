package sequencing

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/protolambda/ctxlock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// defaultSealingDuration defines the expected time it takes to seal the block
const defaultSealingDuration = 50 * time.Millisecond

var (
	ErrSequencerAlreadyStarted = errors.New("sequencer already running")
	ErrSequencerAlreadyStopped = errors.New("sequencer not running")
)

type L1OriginSelectorIface interface {
	FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error)
	SetRecoverMode(bool)
}

type Metrics interface {
	SetSequencerState(active bool)
	RecordSequencerInconsistentL1Origin(from eth.BlockID, to eth.BlockID)
	RecordSequencerReset()
	RecordSequencingError()
}

type SequencerStateListener interface {
	SequencerStarted() error
	SequencerStopped() error
}

type AsyncGossiper interface {
	Gossip(payload *eth.ExecutionPayloadEnvelope)
	Get() *eth.ExecutionPayloadEnvelope
	Clear()
	Stop()
	Start()
}

type BuildingState struct {
	Onto eth.L2BlockRef
	Info eth.PayloadInfo

	Started time.Time

	// Set once known
	Ref eth.L2BlockRef
}

// Sequencer implements the sequencing interface of the driver: it starts and completes block building jobs.
type Sequencer struct {
	mu ctxlock.Lock

	// closed when driver system closes, to interrupt any ongoing API calls etc.
	ctx context.Context

	log             log.Logger
	rollupCfg       *rollup.Config
	spec            *rollup.ChainSpec
	sealingDuration time.Duration

	maxSafeLag atomic.Uint64

	// stalledByMaxSafeLag tracks whether the sequencer was stalled specifically
	// by the maxSafeLag check, to avoid incorrectly resuming when nextActionArmed
	// was set to false for unrelated reasons (e.g., pipeline reset, L1-derivation backoff).
	stalledByMaxSafeLag bool

	recoverMode atomic.Bool

	// active identifies whether the sequencer is running.
	// This is an atomic value, so it can be read without locking the whole sequencer.
	active atomic.Bool

	// listener for sequencer-state changes. Blocking, may error.
	// May be used to ensure sequencer-state is accurately persisted.
	listener SequencerStateListener

	conductor conductor.SequencerConductor

	asyncGossip AsyncGossiper

	emitter event.Emitter

	eng SequencerEngine

	attrBuilder      derive.AttributesBuilder
	l1OriginSelector L1OriginSelectorIface

	metrics Metrics

	// timeNow enables sequencer testing to mock the time
	timeNow func() time.Time

	// nextAction is when the next sequencing action should be performed
	nextAction      time.Time
	nextActionArmed bool
	// awaitingResetConfirm parks the sequencer between a reset signal and its
	// confirmation. The engine emits the rewound forkchoice update before
	// EngineResetConfirmedEvent, and that update must not re-arm the schedule:
	// the confirmation carries a deliberate one-block delay that keeps a reset
	// loop from running hot.
	awaitingResetConfirm bool

	building   BuildingState
	lastSealed eth.L2BlockRef
	unsafeHead eth.L2BlockRef
	// safeHead is the safe head as of the last ingested forkchoice update,
	// used to enforce maxSafeLag on the direct-call scheduling path.
	safeHead eth.L2BlockRef

	headKnownCh chan struct{}

	// inbox is an ordered log of external events, appended by the event loop
	// via OnEvent and replayed by the sequencer goroutine via drainInbox.
	inboxMu sync.Mutex
	inbox   []event.Event

	// wakeCh coalesces wake-up signals (cap 1) for the sequencer goroutine,
	// so inbox appends and RPC start/stop interrupt its timer wait.
	wakeCh chan struct{}

	// toBlockRef converts a payload to a block-ref, and is only configurable for test-purposes
	toBlockRef func(rollupCfg *rollup.Config, payload *eth.ExecutionPayload) (eth.L2BlockRef, error)

	catchupSchedule *CatchupSchedule
}

var _ SequencerIface = (*Sequencer)(nil)

func NewSequencer(driverCtx context.Context, log log.Logger, rollupCfg *rollup.Config,
	sealingDuration time.Duration,
	attributesBuilder derive.AttributesBuilder,
	l1OriginSelector L1OriginSelectorIface,
	listener SequencerStateListener,
	conductor conductor.SequencerConductor,
	asyncGossip AsyncGossiper,
	metrics Metrics,
	eng SequencerEngine,
) *Sequencer {
	return NewSequencerWithClock(driverCtx, log, rollupCfg, sealingDuration, attributesBuilder, l1OriginSelector,
		listener, conductor, asyncGossip, metrics, eng, nil, clock.SystemClock)
}

func NewSequencerWithClock(driverCtx context.Context, log log.Logger, rollupCfg *rollup.Config,
	sealingDuration time.Duration,
	attributesBuilder derive.AttributesBuilder,
	l1OriginSelector L1OriginSelectorIface,
	listener SequencerStateListener,
	conductor conductor.SequencerConductor,
	asyncGossip AsyncGossiper,
	metrics Metrics,
	eng SequencerEngine,
	catchupSchedule *CatchupSchedule,
	runtimeClock clock.Clock,
) *Sequencer {
	if sealingDuration <= 0 {
		sealingDuration = defaultSealingDuration
	}
	return &Sequencer{
		ctx:              driverCtx,
		log:              log,
		rollupCfg:        rollupCfg,
		spec:             rollup.NewChainSpec(rollupCfg),
		sealingDuration:  sealingDuration,
		listener:         listener,
		conductor:        conductor,
		asyncGossip:      asyncGossip,
		attrBuilder:      attributesBuilder,
		l1OriginSelector: l1OriginSelector,
		metrics:          metrics,
		eng:              eng,
		timeNow:          runtimeClock.Now,
		wakeCh:           make(chan struct{}, 1),
		toBlockRef:       derive.PayloadToBlockRef,
		catchupSchedule:  catchupSchedule,
	}
}

func (s *Sequencer) AttachEmitter(em event.Emitter) {
	s.emitter = em
}

// OnEvent implements event.Deriver. It never blocks and never mutates
// sequencer state: accepted events are appended to the inbox log and
// replayed in arrival order on the sequencer goroutine.
func (s *Sequencer) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case engine.ForkchoiceUpdateInitEvent:
		s.ingest(engine.ForkchoiceUpdateEvent(x))
	case engine.PayloadSuccessEvent:
		// Only the block hash is used. Ingesting the event as-is would keep a
		// whole block, with all its transactions, alive in the inbox.
		s.ingest(payloadSuccess{blockHash: x.Envelope.ExecutionPayload.BlockHash})
	case engine.ForkchoiceUpdateEvent,
		rollup.ResetEvent,
		engine.EngineResetConfirmedEvent,
		rollup.EngineTemporaryErrorEvent,
		engine.BuildStartedEvent:
		s.ingest(x)
	default:
		return false
	}
	return true
}

// payloadSuccess is the inbox form of engine.PayloadSuccessEvent, reduced to
// the only field the sequencer reads.
type payloadSuccess struct{ blockHash common.Hash }

func (payloadSuccess) String() string { return "payload-success" }

// ingest appends an event to the inbox and wakes the sequencer goroutine.
// A head update replaces the last inbox entry if that entry is also a head
// update that does not advance past the new one: nothing sits between
// adjacent entries, so relative order with other event kinds is unaffected.
// A rewinding head update (block replacement, backup-unsafe restore) is
// appended instead, so replay still drops build jobs onto the rewound head.
func (s *Sequencer) ingest(ev event.Event) {
	s.inboxMu.Lock()
	defer s.wake() // deferred LIFO: runs after the unlock below
	defer s.inboxMu.Unlock()
	if fcu, isHead := ev.(engine.ForkchoiceUpdateEvent); isHead && len(s.inbox) > 0 {
		if last, lastIsHead := s.inbox[len(s.inbox)-1].(engine.ForkchoiceUpdateEvent); lastIsHead &&
			fcu.UnsafeL2Head.Number >= last.UnsafeL2Head.Number {
			s.inbox[len(s.inbox)-1] = ev
			return
		}
	}
	s.inbox = append(s.inbox, ev)
}

// wake signals the sequencer goroutine to re-plan its schedule. Non-blocking.
func (s *Sequencer) wake() {
	select {
	case s.wakeCh <- struct{}{}:
	default:
	}
}

// drainInbox replays queued external events in arrival order.
// Must only be called from the sequencer goroutine.
func (s *Sequencer) drainInbox() {
	s.inboxMu.Lock()
	events := s.inbox
	s.inbox = nil
	s.inboxMu.Unlock()
	if len(events) == 0 {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	preTime := s.nextAction
	preOk := s.nextActionArmed
	for _, ev := range events {
		s.handleIngest(ev)
	}
	if s.nextActionArmed != preOk || s.nextAction != preTime {
		s.log.Debug("Sequencer action schedule changed",
			"time", s.nextAction, "wait", s.nextAction.Sub(s.timeNow()), "armed", s.nextActionArmed)
	}
}

// handleIngest dispatches a single inbox event to its handler.
// Must be called with the sequencer lock held.
func (s *Sequencer) handleIngest(ev event.Event) {
	switch x := ev.(type) {
	case engine.BuildStartedEvent:
		s.onBuildStarted(x)
	case payloadSuccess:
		s.onPayloadSuccess(x)
	case rollup.EngineTemporaryErrorEvent:
		s.onEngineTemporaryError(x)
	case rollup.ResetEvent:
		s.onReset(x)
	case engine.EngineResetConfirmedEvent:
		s.onEngineResetConfirmedEvent(x)
	case engine.ForkchoiceUpdateEvent:
		s.onForkchoiceUpdate(x)
	}
}

// RunLoop runs the sequencer scheduling loop: it replays ingested events and
// runs the next sequencer action when its scheduled time arrives, until ctx
// is canceled. Event ingestion and RPC start/stop interrupt the wait via the
// wake channel.
func (s *Sequencer) RunLoop(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		s.drainInbox()

		// Re-plan from post-drain state. A nil channel blocks forever,
		// disarming the timer case while no action is scheduled.
		var timerC <-chan time.Time
		if nextAction, ok := s.NextAction(); ok {
			// Same clock the due-check below uses, so an early fire re-arms on
			// the true remaining wait instead of busy-looping.
			timer.Reset(nextAction.Sub(s.timeNow()))
			timerC = timer.C
		} else {
			timer.Stop()
		}

		select {
		case <-ctx.Done():
			return
		case <-s.wakeCh:
			// loop around: drain the inbox and re-plan
		case <-timerC:
			s.drainInbox() // events may have arrived while we were sleeping
			// Only act on a deadline that is actually due. The drain may have
			// pushed it out (engine backoff, post-reset delay), and the timer
			// itself can fire early: deadlines are wall-clock values while the
			// sleep is monotonic, so a backward clock step arrives here as an
			// early fire. Either way, loop around and re-plan on the remainder.
			if next, ok := s.NextAction(); !ok || next.After(s.timeNow()) {
				s.log.Debug("Sequencer deadline not due yet, re-planning", "next", next)
				continue
			}
			s.RunAction()
		}
	}
}

func (s *Sequencer) onBuildStarted(x engine.BuildStartedEvent) {
	if x.DerivedFrom != (eth.L1BlockRef{}) {
		// If we are adding new blocks onto the tip of the chain, derived from L1,
		// then don't try to build on top of it immediately, as sequencer.
		s.log.Warn("Detected new block-building from L1 derivation, avoiding sequencing for now.",
			"build_job", x.Info.ID, "build_timestamp", x.Info.Timestamp,
			"parent", x.Parent, "derived_from", x.DerivedFrom)
		s.nextActionArmed = false
		return
	}
	// Sequencer-originated builds are handled inline via direct engine calls.
	s.log.Debug("Ignoring build-started event of sequencer-originated block", "payloadID", x.Info.ID)
}

func (s *Sequencer) handleInvalid() {
	s.metrics.RecordSequencingError()
	s.building = BuildingState{}
	// A block we sealed may be discarded here (invalid or denied insertion), and
	// will then never become the head. Reconcile the sealed marker so Stop, which
	// waits for the head to catch up to it, is not left waiting for a dead block.
	s.lastSealed = s.unsafeHead
	s.asyncGossip.Clear()
	// upon error, retry after one block worth of time
	blockTime := time.Duration(s.rollupCfg.BlockTime) * time.Second
	s.nextAction = s.timeNow().Add(blockTime)
	s.nextActionArmed = s.active.Load()
}

func (s *Sequencer) onPayloadSuccess(x payloadSuccess) {
	// Sequencer-originated payloads are processed inline via direct engine calls.
	// For derivation-originated payloads (e.g. replacement blocks) we still clear
	// the async-gossip buffer, so the sequencer cannot reuse a stale payload after
	// a chain reset.
	if s.building.Ref != (eth.L2BlockRef{}) && s.building.Ref.Hash != x.blockHash {
		return // not relevant to us
	}
	s.building = BuildingState{}
	s.asyncGossip.Clear()
}

// RunAction performs one sequencer action: it processes a payload left in the
// async-gossip buffer, seals the pending block-building job, or starts a new
// build. Engine interactions happen as direct synchronous calls, not events.
func (s *Sequencer) RunAction() {
	s.drainInbox() // decide from up-to-date state

	s.mu.Lock()
	defer s.mu.Unlock()

	preTime := s.nextAction
	preOk := s.nextActionArmed
	defer func() {
		if s.nextActionArmed != preOk || s.nextAction != preTime {
			s.log.Debug("Sequencer action schedule changed",
				"time", s.nextAction, "wait", s.nextAction.Sub(s.timeNow()), "armed", s.nextActionArmed)
		}
	}()

	s.log.Debug("Sequencer action")
	if !s.active.Load() {
		s.log.Debug("Ignoring stale sequencer action while inactive")
		// Every exit must leave the schedule changed or disarmed, so the loop
		// never re-fires an unchanged deadline. Start/Stop normally keep these
		// two in step; this makes it structural rather than incidental.
		s.nextActionArmed = false
		return
	}
	// The inbox drain above may have parked the schedule (reset, derivation
	// build, engine backoff, maxSafeLag stall). The timer fire that triggered
	// this action predates that state: honor the post-drain decision.
	if !s.nextActionArmed {
		s.log.Debug("Ignoring sequencer action, no action scheduled after inbox replay")
		return
	}
	payload := s.asyncGossip.Get()
	if payload != nil {
		if s.building.Info.ID == (eth.PayloadID{}) {
			s.log.Warn("Found reusable payload from async gossiper, and no block was being built. Reusing payload.",
				"hash", payload.ExecutionPayload.BlockHash,
				"number", uint64(payload.ExecutionPayload.BlockNumber),
				"parent", payload.ExecutionPayload.ParentHash)
		}
		ref, err := s.toBlockRef(s.rollupCfg, payload.ExecutionPayload)
		if err != nil {
			s.log.Error("Payload from async-gossip buffer could not be turned into block-ref", "err", err)
			// Treat like an invalid payload: drop it and retry with a fresh
			// build after a backoff. No event echo re-arms this failure.
			s.handleInvalid()
			return
		}
		s.log.Info("Resuming sequencing with previously async-gossip confirmed payload",
			"payload", payload.ExecutionPayload.ID())
		// The payload is known and was already gossiped: it must have been sealed
		// before a temporary error. Retry processing to make it canonical.
		// Park the schedule first: on a temporary error the engine's
		// EngineTemporaryErrorEvent echoes back through the mailbox and re-arms
		// the backoff; re-firing before that echo would spin on the engine.
		s.nextActionArmed = false
		if err := s.eng.ProcessPayload(s.ctx, payload, ref, time.Time{}); err != nil {
			if errors.Is(err, engine.ErrStaleBuild) {
				// The chain moved past the buffered payload; it is worthless now.
				// Nothing of ours is outstanding anymore: don't leave Stop
				// waiting for the dropped block to become the head.
				s.building = BuildingState{}
				s.lastSealed = s.unsafeHead
				s.asyncGossip.Clear()
				if ref.ParentHash != s.unsafeHead.Hash {
					// The payload was already stale against the head we know, so
					// the forkchoice update the engine just requested carries that
					// same head and will not re-plan anything. Re-plan here, or the
					// sequencer parks forever waiting for an echo it has had.
					s.scheduleNextAction(s.unsafeHead)
				}
				// Otherwise the engine moved during the call: its forkchoice update
				// names a head we have not seen yet and re-arms us on arrival.
			} else if errors.Is(err, engine.ErrPayloadDenied) || errors.Is(err, engine.ErrPayloadInvalid) {
				s.handleInvalid()
			}
			return
		}
		s.building = BuildingState{}
		s.asyncGossip.Clear()
		s.scheduleNextAction(ref)
		return
	}
	if s.building.Info != (eth.PayloadInfo{}) {
		// We should not repeat the seal request.
		s.nextActionArmed = false
		sealResult, err := s.eng.SealBuild(s.ctx, s.building.Info, s.building.Started)
		if err != nil {
			if errors.Is(err, engine.ErrStaleBuild) {
				// A competing block landed while we were building; drop the job
				// without committing or gossiping the stale sibling. Parked until
				// the engine-requested forkchoice update arrives.
				s.log.Warn("Dropping stale sealed block, chain moved past it", "payloadID", s.building.Info.ID)
				s.building = BuildingState{}
				return
			}
			// Restart building on seal errors (expired or invalid), this way we get
			// a block we should be able to seal (smaller, since we adapt build time).
			s.handleInvalid()
			return
		}
		envelope := sealResult.Envelope
		s.log.Info("Sequencer sealed block", "payloadID", s.building.Info.ID,
			"block", envelope.ExecutionPayload.ID(),
			"parent", envelope.ExecutionPayload.ParentID(),
			"txs", len(envelope.ExecutionPayload.Transactions),
			"time", uint64(envelope.ExecutionPayload.Timestamp))

		// generous timeout, the conductor is important
		commitCtx, cancel := context.WithTimeout(s.ctx, time.Second*30)
		defer cancel()
		if err := s.conductor.CommitUnsafePayload(commitCtx, envelope); err != nil {
			s.emitter.Emit(s.ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("failed to commit unsafe payload to conductor: %w", err),
			})
			return
		}

		// Begin gossiping as soon as possible.
		// asyncGossip.Clear() is called once the payload is successfully inserted below,
		// or when a later action hits a non-temporary error.
		s.asyncGossip.Gossip(envelope)
		s.building.Ref = sealResult.Ref
		s.lastSealed = sealResult.Ref
		// Now after having gossiped the block, try to put it in our own canonical chain.
		if err := s.eng.ProcessPayload(s.ctx, envelope, sealResult.Ref, s.building.Started); err != nil {
			if errors.Is(err, engine.ErrStaleBuild) {
				// A competing block landed after we sealed and gossiped; the
				// gossiped sibling lost. Drop it rather than reorging back to it.
				// Parked until the engine-requested forkchoice update arrives.
				// Reset lastSealed so Stop does not wait for the dropped
				// block to ever become the head.
				s.log.Warn("Dropping stale processed block, chain moved past it", "block", sealResult.Ref)
				s.building = BuildingState{}
				s.lastSealed = s.unsafeHead
				s.asyncGossip.Clear()
			} else if errors.Is(err, engine.ErrPayloadDenied) || errors.Is(err, engine.ErrPayloadInvalid) {
				s.handleInvalid()
			}
			return
		}
		s.log.Info("Sequencer inserted block",
			"block", sealResult.Ref, "parent", envelope.ExecutionPayload.ParentID())
		s.building = BuildingState{}
		// The payload was already published upon sealing.
		// Now that we have processed it ourselves we don't need it anymore.
		s.asyncGossip.Clear()
		s.scheduleNextAction(sealResult.Ref)
	} else if s.building == (BuildingState{}) {
		// If we have not started building anything, start building.
		s.startBuildingBlock()
	}
}

func (s *Sequencer) onEngineTemporaryError(x rollup.EngineTemporaryErrorEvent) {
	if s.building == (BuildingState{}) {
		s.log.Debug("Engine reported temporary error while building state is empty", "err", x.Err)
	}
	s.log.Error("Engine failed temporarily, backing off sequencer", "err", x.Err)
	if errors.Is(x.Err, engine.ErrEngineSyncing) { // if it is syncing we can back off by more
		s.nextAction = s.timeNow().Add(30 * time.Second)
	} else {
		s.nextAction = s.timeNow().Add(time.Second)
	}
	// A reset in flight keeps the sequencer parked: the engine has not rewound
	// yet, so re-arming here would resume building on the pre-reset head. The
	// confirmation carries the cool-down, and a backoff must not pre-empt it.
	s.nextActionArmed = s.active.Load() && !s.awaitingResetConfirm
	// Re-check the lag bound: this is the only ingest-path re-arm, so it is the
	// only way a maxSafeLag-stalled sequencer can be released early.
	s.evalMaxSafeLag()
	// We don't explicitly cancel block building jobs upon temporary errors: we may still finish the block (if any).
	// Any unfinished block building work eventually times out, and will be cleaned up that way.
	// Note that this only applies to temporary errors upon starting a block-building job.
	// If the engine errors upon sealing, the seal-error handling restarts the build with fresh attributes.

	// If we don't have an ID of a job to resume, then start over.
	// (s.building.Onto would be set if we emitted BuildStart already)
	if s.building.Info == (eth.PayloadInfo{}) {
		s.building = BuildingState{}
	}
}

func (s *Sequencer) onReset(x rollup.ResetEvent) {
	s.log.Error("Sequencer encountered reset signal, aborting work", "err", x.Err)
	s.metrics.RecordSequencerReset()
	// try to cancel any ongoing payload building job
	if s.building.Info != (eth.PayloadInfo{}) {
		s.emitter.Emit(s.ctx, engine.BuildCancelEvent{Info: s.building.Info})
	}
	s.building = BuildingState{}
	// A block we sealed may be discarded by this reset, and the head is rewound
	// again right after, so clear the marker rather than pointing it at the head.
	s.lastSealed = eth.L2BlockRef{}
	s.stalledByMaxSafeLag = false
	// no action to perform until we get a reset-confirmation
	s.nextActionArmed = false
	s.awaitingResetConfirm = true
}

func (s *Sequencer) onEngineResetConfirmedEvent(engine.EngineResetConfirmedEvent) {
	s.awaitingResetConfirm = false
	s.nextActionArmed = s.active.Load()
	// Before sequencing we can wait a block,
	// assuming the execution-engine just churned through some work for the reset.
	// This will also prevent any potential reset-loop from running too hot.
	s.nextAction = s.timeNow().Add(time.Second * time.Duration(s.rollupCfg.BlockTime))
	// The reset delivered fresh forkchoice state just before this confirmation;
	// re-check the lag bound rather than unconditionally re-arming, so a still
	// badly lagging sequencer does not build one extra block before the next
	// forkchoice update re-stalls it. (onReset cleared stalledByMaxSafeLag.)
	s.evalMaxSafeLag()
	s.log.Info("Engine reset confirmed, sequencer may continue", "armed", s.nextActionArmed)
}

func (s *Sequencer) onForkchoiceUpdate(x engine.ForkchoiceUpdateEvent) {
	s.log.Debug("Sequencer is processing forkchoice update", "unsafe", x.UnsafeL2Head, "prev_unsafe", s.unsafeHead)

	s.safeHead = x.SafeL2Head
	if !s.active.Load() {
		s.setUnsafeHead(x.UnsafeL2Head)
		return
	}
	// Drop a stale block-building job if the chain no longer sits on its parent.
	// The head can move backwards (block replacement, backup-unsafe restore, an
	// engine reset), so compare identity, not height.
	if s.building != (BuildingState{}) && s.building.Onto.Hash != x.UnsafeL2Head.Hash {
		s.log.Debug("Dropping stale/completed block-building job",
			"state", s.building.Onto, "unsafe", x.UnsafeL2Head)
		// The cleared state stops the next sequencer action from sealing the stale build job.
		s.building = BuildingState{}
	}
	if x.UnsafeL2Head.Hash != s.unsafeHead.Hash && !s.awaitingResetConfirm {
		// Any change of head — including a rewind — needs a fresh plan on top of
		// it. Re-planning only on a higher block number would leave the sequencer
		// parked forever after a rewind, because the engine rejects builds and
		// seals that no longer extend its head and the recovering forkchoice
		// update is the rewound one.
		s.scheduleNextAction(x.UnsafeL2Head)
	} else {
		s.setUnsafeHead(x.UnsafeL2Head)
		// Re-evaluate the stall even when the head is unchanged: the resume
		// trigger is the safe head catching up, which arrives on exactly such
		// forkchoice updates. (scheduleNextAction evaluates it itself.)
		s.evalMaxSafeLag()
	}
}

// scheduleNextAction arms the sequencer to build the next block on top of the
// given new head, leaving spare time if the head is fresh enough.
// Called after a block insertion, either from RunAction (direct-call path)
// or from onForkchoiceUpdate (event path).
func (s *Sequencer) scheduleNextAction(newHead eth.L2BlockRef) {
	s.nextActionArmed = true
	now := s.timeNow()
	if s.catchupSchedule != nil {
		deadline, _, err := s.catchupSchedule.BuildDeadline(newHead, s.sealingDuration)
		if err != nil {
			s.log.Error("Invalid checkpoint-relative catch-up schedule", "err", err, "head", newHead)
			s.metrics.RecordSequencingError()
			s.nextActionArmed = false
			s.setUnsafeHead(newHead)
			return
		}
		if deadline.Before(now) {
			deadline = now
		}
		s.nextAction = deadline
		s.setUnsafeHead(newHead)
		s.evalMaxSafeLag()
		return
	}
	blockTime := time.Duration(s.rollupCfg.BlockTime) * time.Second
	payloadTime := time.Unix(int64(newHead.Time+s.rollupCfg.BlockTime), 0)
	remainingTime := payloadTime.Sub(now)
	if remainingTime > blockTime {
		// if we have too much time, then wait before starting the build
		s.nextAction = payloadTime.Add(-blockTime)
	} else {
		// otherwise start instantly
		s.nextAction = now
	}
	s.setUnsafeHead(newHead)
	// The stall check must run after arming (nextActionArmed=true above), so it
	// can override it. Evaluating here, not just on forkchoice updates, keeps
	// the lag bound enforced on the direct-call path too: the sequencer must
	// not outrun it while the stalling forkchoice echo is still queued.
	s.evalMaxSafeLag()
}

// evalMaxSafeLag stalls block production while the safe head (as of the last
// ingested forkchoice update) lags the unsafe head by more than the configured
// bound, and resumes it when the safe head catches up.
// Must run after any schedule arming so the stall can override it.
func (s *Sequencer) evalMaxSafeLag() {
	if maxSafeLag := s.maxSafeLag.Load(); maxSafeLag > 0 {
		if s.safeHead.Number+maxSafeLag <= s.unsafeHead.Number {
			if !s.stalledByMaxSafeLag {
				s.log.Warn("sequencer has fallen behind safe head by more than lag, stalling",
					"unsafe", s.unsafeHead, "safe", s.safeHead, "max_lag", maxSafeLag)
			}
			s.nextActionArmed = false
			s.stalledByMaxSafeLag = true
		} else if s.stalledByMaxSafeLag {
			// Safe head has caught up after a maxSafeLag stall, resume sequencing.
			// Only resume if we were stalled by maxSafeLag specifically, to avoid
			// interfering with other nextActionArmed=false states (reset, L1-derivation backoff, etc).
			s.log.Info("safe head caught up, resuming sequencing",
				"unsafe", s.unsafeHead, "safe", s.safeHead, "max_lag", maxSafeLag)
			s.stalledByMaxSafeLag = false
			s.nextActionArmed = s.active.Load()
			s.nextAction = s.timeNow()
		}
	} else if s.stalledByMaxSafeLag {
		// maxSafeLag was disabled at runtime (set to 0) while stalled; resume immediately.
		s.log.Info("maxSafeLag disabled, resuming sequencing")
		s.stalledByMaxSafeLag = false
		s.nextActionArmed = s.active.Load()
		s.nextAction = s.timeNow()
	}
}

func (s *Sequencer) setUnsafeHead(head eth.L2BlockRef) {
	s.unsafeHead = head
	if s.headKnownCh != nil {
		close(s.headKnownCh)
		s.headKnownCh = nil
	}
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (s *Sequencer) startBuildingBlock() {
	ctx := s.ctx
	l2Head := s.unsafeHead

	// If we do not have data to know what to build on, then request a forkchoice update.
	// Park until the head update arrives through the mailbox; re-firing on the
	// unchanged schedule would hot-loop forkchoice requests while the engine
	// state is still unknown (e.g. before the initial engine reset completes).
	if l2Head == (eth.L2BlockRef{}) {
		s.nextActionArmed = false
		s.eng.RequestForkchoiceUpdate(s.ctx)
		return
	}

	recoverMode := s.recoverMode.Load()

	// Figure out which L1 origin block we're going to be building on top of.
	l1Origin, err := s.l1OriginSelector.FindL1Origin(ctx, l2Head)
	switch {
	case err == nil:
	case errors.Is(err, ErrInvalidL1Origin), errors.Is(err, ErrNextL1OriginOrphaned):
		s.metrics.RecordSequencerInconsistentL1Origin(l2Head.L1Origin, l1Origin.ID())
		// Park: the ResetEvent echoes back through the mailbox (onReset keeps us
		// parked until the reset is confirmed).
		s.nextActionArmed = false
		s.emitter.Emit(s.ctx, rollup.ResetEvent{
			Err: fmt.Errorf("cannot build new L2 block with L1 origin %s (parent L1 %s) on current L2 head %s with L1 origin %s",
				l1Origin, l1Origin.ParentHash, l2Head, l2Head.L1Origin),
		})
		return
	case errors.Is(err, ErrNextL1OriginRequired):
		fallthrough
	default:
		s.nextAction = s.timeNow().Add(time.Second)
		s.nextActionArmed = s.active.Load()
		s.log.Error("Error finding next L1 Origin", "err", err)
		s.emitter.Emit(s.ctx, rollup.L1TemporaryErrorEvent{Err: err})
		return
	}

	s.log.Info("Started sequencing new block", "parent", l2Head, "l1Origin", l1Origin)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	attrs, err := s.attrBuilder.PreparePayloadAttributes(fetchCtx, l2Head, l1Origin.ID())
	if err != nil {
		// Park before emitting: recovery is driven by the event echoing back
		// through the mailbox (temp-error backoff re-arms, reset stays parked
		// until confirmed). An armed past deadline would spin the loop until
		// the echo arrives.
		s.nextActionArmed = false
		if errors.Is(err, derive.ErrTemporary) {
			s.emitter.Emit(s.ctx, rollup.EngineTemporaryErrorEvent{Err: err})
			return
		} else if errors.Is(err, derive.ErrReset) {
			s.emitter.Emit(s.ctx, rollup.ResetEvent{Err: err})
			return
		} else if errors.Is(err, derive.ErrCritical) {
			s.emitter.Emit(s.ctx, rollup.CriticalErrorEvent{Err: err})
			return
		} else {
			s.emitter.Emit(s.ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected attributes-preparation error: %w", err),
			})
			return
		}
	}

	// If our next L2 block timestamp is beyond the Sequencer drift threshold, then we must produce
	// empty blocks (other than the L1 info deposit and any user deposits). We handle this by
	// setting NoTxPool to true, which will cause the Sequencer to not include any transactions
	// from the transaction pool.
	maxDrift := s.spec.MaxSequencerDrift(l1Origin.Time)
	attrs.NoTxPool = uint64(attrs.Timestamp) > l1Origin.Time+maxDrift
	if s.catchupSchedule != nil && s.catchupSchedule.ShouldParkForOrigin(uint64(attrs.Timestamp), l1Origin.Time, maxDrift) {
		s.nextAction = s.timeNow().Add(s.catchupSchedule.OriginRetryDelay())
		s.nextActionArmed = s.active.Load()
		s.log.Warn("Parking checkpoint catch-up until private L1 restores origin headroom",
			"payload_time", uint64(attrs.Timestamp), "origin_time", l1Origin.Time, "max_drift", maxDrift)
		return
	}

	// For the Ecotone activation block we shouldn't include any sequencer transactions.
	if s.rollupCfg.IsEcotoneActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		s.log.Info("Sequencing Ecotone upgrade block")
	}

	// For the Fjord activation block we shouldn't include any sequencer transactions.
	if s.rollupCfg.IsFjordActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		s.log.Info("Sequencing Fjord upgrade block")
	}

	// For the Granite activation block we can include sequencer transactions.
	if s.rollupCfg.IsGraniteActivationBlock(uint64(attrs.Timestamp)) {
		s.log.Info("Sequencing Granite upgrade block")
	}

	// For the Isthmus activation block we shouldn't include any sequencer transactions.
	if s.rollupCfg.IsIsthmusActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		s.log.Info("Sequencing Isthmus upgrade block")
	}

	// For the Jovian activation block we must not include any sequencer transactions.
	if s.rollupCfg.IsJovianActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		s.log.Info("Sequencing Jovian upgrade block")
	}

	// For the Karst activation block we must not include any sequencer transactions.
	if s.rollupCfg.IsKarstActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		s.log.Info("Sequencing Karst upgrade block")
	}

	// For the Lagoon activation block we must not include any sequencer transactions.
	if s.rollupCfg.IsInteropActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		s.log.Info("Sequencing Lagoon upgrade block")
	}

	if recoverMode {
		attrs.NoTxPool = true
		s.log.Warn("Sequencing temporarily without user transactions, in recover mode")
	}

	s.log.Debug("prepared attributes for new block",
		"num", l2Head.Number+1, "time", uint64(attrs.Timestamp),
		"origin", l1Origin, "origin_time", l1Origin.Time, "noTxPool", attrs.NoTxPool)

	// Start a payload building process.
	withParent := &derive.AttributesWithParent{
		Attributes:  attrs,
		Parent:      l2Head,
		Concluding:  false,
		DerivedFrom: eth.L1BlockRef{}, // zero, not going to be pending-safe / safe
	}

	// Don't try to start building a block again, until we have heard back from this attempt
	s.nextActionArmed = false

	// Reset building state, and remember what we are building on.
	// If we get a forkchoice update that conflicts, we will have to abort building.
	s.building = BuildingState{Onto: l2Head}

	result, err := s.eng.StartBuild(s.ctx, withParent)
	if err != nil {
		if errors.Is(err, engine.ErrStaleBuild) {
			// The engine head moved past our build target; the engine already requested
			// a forkchoice update, so wait for the next head update to re-plan.
			s.log.Warn("Engine rejected stale build attempt, waiting for next head update",
				"parent", l2Head, "err", err)
			s.building = BuildingState{}
			return
		}
		if errors.Is(err, engine.ErrBuildInvalid) {
			// No recovery event reaches us for invalid attributes of our own
			// builds; back off locally and retry with a new job.
			s.handleInvalid()
			return
		}
		// Temporary, reset, or critical error: the engine emitted the matching
		// event, which echoes back through the mailbox and re-arms or keeps us
		// parked as appropriate. Stay parked (set before StartBuild) meanwhile.
		return
	}
	if s.building.Onto != result.Parent {
		s.log.Warn("Canceling stale block-building job that was just started, as target to build onto has changed",
			"stale", result.Parent, "new", s.building.Onto, "job_id", result.Info.ID, "job_timestamp", result.Info.Timestamp)
		s.emitter.Emit(s.ctx, engine.BuildCancelEvent{
			Info:  result.Info,
			Force: true,
		})
		s.handleInvalid()
		return
	}
	s.log.Debug("Sequencer started building new block",
		"payloadID", result.Info.ID, "parent", result.Parent, "parent_time", result.Parent.Time)
	s.building.Info = result.Info
	s.building.Started = result.BuildStarted

	s.nextActionArmed = s.active.Load()

	// schedule sealing
	now := s.timeNow()
	if s.catchupSchedule != nil {
		deadline, err := s.catchupSchedule.SealDeadline(result.Parent, s.sealingDuration)
		if err != nil {
			s.log.Error("Invalid checkpoint-relative catch-up seal schedule", "err", err, "parent", result.Parent)
			s.metrics.RecordSequencingError()
			s.handleInvalid()
			return
		}
		if deadline.Before(now) {
			deadline = now
		}
		s.nextAction = deadline
		return
	}
	payloadTime := time.Unix(int64(result.Parent.Time+s.rollupCfg.BlockTime), 0)
	remainingTime := payloadTime.Sub(now)
	if remainingTime < s.sealingDuration {
		s.nextAction = now // if there's not enough time for sealing, don't wait.
	} else {
		// finish with margin of sealing duration before payloadTime
		s.nextAction = payloadTime.Add(-s.sealingDuration)
	}
}

func (s *Sequencer) NextAction() (t time.Time, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.nextAction, s.nextActionArmed
}

// Building returns the block-building job the sequencer currently has in
// flight. The zero value means no job is in flight.
func (s *Sequencer) Building() BuildingState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.building
}

func (s *Sequencer) Active() bool {
	return s.active.Load()
}

func (s *Sequencer) Start(ctx context.Context, head common.Hash) error {
	// must be leading to activate
	if isLeader, err := s.conductor.Leader(ctx); err != nil {
		return fmt.Errorf("sequencer leader check failed: %w", err)
	} else if !isLeader {
		return errors.New("sequencer is not the leader, aborting")
	}

	// Note: leader check happens before locking; this is how the Driver used to work,
	// and prevents the event-processing of the sequencer from being stalled due to a potentially slow conductor call.
	if err := s.mu.LockCtx(ctx); err != nil {
		return err
	}
	defer s.mu.Unlock()

	if s.active.Load() {
		return ErrSequencerAlreadyStarted
	}
	if s.unsafeHead == (eth.L2BlockRef{}) {
		return fmt.Errorf("no prestate, cannot determine if sequencer start at %s is safe", head)
	}
	if head != s.unsafeHead.Hash {
		return fmt.Errorf("block hash does not match: head %s, received %s", s.unsafeHead, head)
	}
	return s.forceStart()
}

func (s *Sequencer) Init(ctx context.Context, active bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.asyncGossip.Start()

	// The `unsafeHead` should be updated, so we can handle start-sequencer requests
	s.eng.RequestForkchoiceUpdate(s.ctx)

	if active {
		return s.forceStart()
	} else {
		s.metrics.SetSequencerState(false)
		if err := s.listener.SequencerStopped(); err != nil {
			return fmt.Errorf("failed to notify sequencer-state listener of initial stopped state: %w", err)
		}
		return nil
	}
}

// forceStart skips all the checks, and just starts the sequencer
func (s *Sequencer) forceStart() error {
	if s.unsafeHead == (eth.L2BlockRef{}) {
		// This happens if sequencing is activated on op-node startup.
		// The op-conductor check and choice of sequencing with this pre-state already happened before op-node startup.
		s.log.Info("Starting sequencing, without known pre-state")
		s.asyncGossip.Clear() // if we are starting from an unknown pre-state, just clear gossip out of caution.
	} else {
		// This happens when we start sequencing on an already-running node.
		s.log.Info("Starting sequencing on top of known pre-state", "unsafe", s.unsafeHead)
		if payload := s.asyncGossip.Get(); payload != nil &&
			payload.ExecutionPayload.BlockHash != s.unsafeHead.Hash {
			s.log.Warn("Cleared old block from async-gossip buffer, sequencing pre-state is different",
				"buffered", payload.ExecutionPayload.ID(), "prestate", s.unsafeHead)
			s.asyncGossip.Clear()
		}
	}

	if err := s.listener.SequencerStarted(); err != nil {
		return fmt.Errorf("failed to notify sequencer-state listener of start: %w", err)
	}
	// clear the building state; interrupting any existing sequencing job (there should never be one)
	s.building = BuildingState{}
	// Same reset rule as the error backoff: if a reset is unconfirmed, unsafeHead
	// is still the pre-reset head (which is exactly what Start checks against), so
	// arming now would sequence on a chain about to be rewound. The confirmation
	// arms us on the new head.
	s.nextActionArmed = !s.awaitingResetConfirm
	s.stalledByMaxSafeLag = false
	s.nextAction = s.timeNow()
	s.active.Store(true)
	s.metrics.SetSequencerState(true)
	s.wake() // wake the sequencer goroutine, so it picks up the new schedule
	s.log.Info("Sequencer has been started", "next action", s.nextAction)
	return nil
}

func (s *Sequencer) Stop(ctx context.Context) (common.Hash, error) {
	if err := s.mu.LockCtx(ctx); err != nil {
		return common.Hash{}, err
	}

	if !s.active.Load() {
		s.mu.Unlock()
		return common.Hash{}, ErrSequencerAlreadyStopped
	}

	// Wait for the head to catch up to a block we sealed and gossiped. A zero
	// marker means there is no such block, so there is nothing to wait for.
	for s.lastSealed != (eth.L2BlockRef{}) && s.unsafeHead.Hash != s.lastSealed.Hash {

		// if we are not the leader, lastSealed will never be updated and we will wait forever
		if isLeader, err := s.conductor.Leader(ctx); err != nil {
			s.log.Warn("Could not determine leadership while stopping. Skipping wait.", "err", err)
			break
		} else if !isLeader {
			s.log.Info("Not leader anymore, skipping head sync wait")
			break
		}

		headKnownCh := make(chan struct{})
		s.headKnownCh = headKnownCh
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return common.Hash{}, ctx.Err()
		case <-headKnownCh:
		}
		if err := s.mu.LockCtx(ctx); err != nil {
			return common.Hash{}, err
		}
	}
	defer s.mu.Unlock()

	// Stop() may have been called twice, so check if we are active after reacquiring the lock
	if !s.active.Load() {
		return common.Hash{}, ErrSequencerAlreadyStopped
	}

	if err := s.listener.SequencerStopped(); err != nil {
		return common.Hash{}, fmt.Errorf("failed to notify sequencer-state listener of stop: %w", err)
	}

	// Cancel any inflight block building. If we don't cancel this, we can resume sequencing an old block
	// even if we've received new unsafe heads in the interim, causing us to introduce a re-org.
	s.building = BuildingState{} // By wiping this state we cannot continue from it later.

	s.nextActionArmed = false
	s.stalledByMaxSafeLag = false
	s.active.Store(false)
	s.metrics.SetSequencerState(false)
	s.wake() // wake the sequencer goroutine, so it parks its action timer
	s.log.Info("Sequencer has been stopped")
	return s.unsafeHead.Hash, nil
}

func (s *Sequencer) SetMaxSafeLag(ctx context.Context, v uint64) error {
	if err := s.mu.LockCtx(ctx); err != nil {
		return err
	}
	defer s.mu.Unlock()
	s.maxSafeLag.Store(v)
	// Apply the new bound immediately instead of waiting for the next forkchoice
	// update: while stalled the sequencer goroutine is parked with no timer armed,
	// so relaxing or disabling the bound would otherwise not resume it at all.
	s.evalMaxSafeLag()
	s.wake()
	return nil
}

func (s *Sequencer) OverrideLeader(ctx context.Context) error {
	return s.conductor.OverrideLeader(ctx)
}

func (s *Sequencer) ConductorEnabled(ctx context.Context) bool {
	return s.conductor.Enabled(ctx)
}

func (s *Sequencer) SetRecoverMode(mode bool) {
	s.l1OriginSelector.SetRecoverMode(mode)
	s.recoverMode.Store(mode)
}

func (s *Sequencer) Close() {
	s.conductor.Close()
	s.asyncGossip.Stop()
}
