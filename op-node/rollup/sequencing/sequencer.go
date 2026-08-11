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
	l ctxlock.Lock

	// closed when driver system closes, to interrupt any ongoing API calls etc.
	ctx context.Context

	log             log.Logger
	rollupCfg       *rollup.Config
	spec            *rollup.ChainSpec
	sealingDuration time.Duration

	maxSafeLag atomic.Uint64

	// stalledByMaxSafeLag tracks whether the sequencer was stalled specifically
	// by the maxSafeLag check, to avoid incorrectly resuming when nextActionOK
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
	nextAction   time.Time
	nextActionOK bool
	// awaitingResetConfirm parks the sequencer between a reset signal and its
	// confirmation. The engine emits the rewound forkchoice update before
	// EngineResetConfirmedEvent, and that update must not re-arm the schedule:
	// the confirmation carries a deliberate one-block delay that keeps a reset
	// loop from running hot.
	awaitingResetConfirm bool

	latest       BuildingState
	latestSealed eth.L2BlockRef
	latestHead   eth.L2BlockRef
	// latestSafe is the safe head as of the last ingested forkchoice update,
	// used to enforce maxSafeLag on the direct-call scheduling path.
	latestSafe eth.L2BlockRef

	latestHeadSet chan struct{}

	// inbox is an ordered log of external events, appended by the event loop
	// via OnEvent and replayed by the sequencer goroutine via drainInbox.
	inboxMu sync.Mutex
	inbox   []event.Event

	// wakeCh coalesces wake-up signals (cap 1) for the sequencer goroutine,
	// so inbox appends and RPC start/stop interrupt its timer wait.
	wakeCh chan struct{}

	// toBlockRef converts a payload to a block-ref, and is only configurable for test-purposes
	toBlockRef func(rollupCfg *rollup.Config, payload *eth.ExecutionPayload) (eth.L2BlockRef, error)
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
		timeNow:          time.Now,
		wakeCh:           make(chan struct{}, 1),
		toBlockRef:       derive.PayloadToBlockRef,
	}
}

func (d *Sequencer) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

// OnEvent implements event.Deriver. It never blocks and never mutates
// sequencer state: accepted events are appended to the inbox log and
// replayed in arrival order on the sequencer goroutine.
func (d *Sequencer) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case engine.ForkchoiceUpdateInitEvent:
		d.ingest(engine.ForkchoiceUpdateEvent(x))
	case engine.PayloadSuccessEvent:
		// Only the block hash is used. Ingesting the event as-is would keep a
		// whole block, with all its transactions, alive in the inbox.
		d.ingest(payloadSuccess{blockHash: x.Envelope.ExecutionPayload.BlockHash})
	case engine.ForkchoiceUpdateEvent,
		rollup.ResetEvent,
		engine.EngineResetConfirmedEvent,
		rollup.EngineTemporaryErrorEvent,
		engine.BuildStartedEvent:
		d.ingest(x)
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
func (d *Sequencer) ingest(ev event.Event) {
	d.inboxMu.Lock()
	defer d.wake() // deferred LIFO: runs after the unlock below
	defer d.inboxMu.Unlock()
	if fcu, isHead := ev.(engine.ForkchoiceUpdateEvent); isHead && len(d.inbox) > 0 {
		if last, lastIsHead := d.inbox[len(d.inbox)-1].(engine.ForkchoiceUpdateEvent); lastIsHead &&
			fcu.UnsafeL2Head.Number >= last.UnsafeL2Head.Number {
			d.inbox[len(d.inbox)-1] = ev
			return
		}
	}
	d.inbox = append(d.inbox, ev)
}

// wake signals the sequencer goroutine to re-plan its schedule. Non-blocking.
func (d *Sequencer) wake() {
	select {
	case d.wakeCh <- struct{}{}:
	default:
	}
}

// drainInbox replays queued external events in arrival order.
// Must only be called from the sequencer goroutine.
func (d *Sequencer) drainInbox() {
	d.inboxMu.Lock()
	events := d.inbox
	d.inbox = nil
	d.inboxMu.Unlock()
	if len(events) == 0 {
		return
	}

	d.l.Lock()
	defer d.l.Unlock()
	preTime := d.nextAction
	preOk := d.nextActionOK
	for _, ev := range events {
		d.handleIngest(ev)
	}
	if d.nextActionOK != preOk || d.nextAction != preTime {
		d.log.Debug("Sequencer action schedule changed",
			"time", d.nextAction, "wait", d.nextAction.Sub(d.timeNow()), "ok", d.nextActionOK)
	}
}

// handleIngest dispatches a single inbox event to its handler.
// Must be called with the sequencer lock held.
func (d *Sequencer) handleIngest(ev event.Event) {
	switch x := ev.(type) {
	case engine.BuildStartedEvent:
		d.onBuildStarted(x)
	case payloadSuccess:
		d.onPayloadSuccess(x)
	case rollup.EngineTemporaryErrorEvent:
		d.onEngineTemporaryError(x)
	case rollup.ResetEvent:
		d.onReset(x)
	case engine.EngineResetConfirmedEvent:
		d.onEngineResetConfirmedEvent(x)
	case engine.ForkchoiceUpdateEvent:
		d.onForkchoiceUpdate(x)
	}
}

// RunLoop runs the sequencer scheduling loop: it replays ingested events and
// runs the next sequencer action when its scheduled time arrives, until ctx
// is canceled. Event ingestion and RPC start/stop interrupt the wait via the
// wake channel.
func (d *Sequencer) RunLoop(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		d.drainInbox()

		// Re-plan from post-drain state. A nil channel blocks forever,
		// disarming the timer case while no action is scheduled.
		var timerC <-chan time.Time
		if nextAction, ok := d.NextAction(); ok {
			// Same clock the due-check below uses, so an early fire re-arms on
			// the true remaining wait instead of busy-looping.
			timer.Reset(nextAction.Sub(d.timeNow()))
			timerC = timer.C
		} else {
			timer.Stop()
		}

		select {
		case <-ctx.Done():
			return
		case <-d.wakeCh:
			// loop around: drain the inbox and re-plan
		case <-timerC:
			d.drainInbox() // events may have arrived while we were sleeping
			// Only act on a deadline that is actually due. The drain may have
			// pushed it out (engine backoff, post-reset delay), and the timer
			// itself can fire early: deadlines are wall-clock values while the
			// sleep is monotonic, so a backward clock step arrives here as an
			// early fire. Either way, loop around and re-plan on the remainder.
			if next, ok := d.NextAction(); !ok || next.After(d.timeNow()) {
				d.log.Debug("Sequencer deadline not due yet, re-planning", "next", next)
				continue
			}
			d.RunAction()
		}
	}
}

func (d *Sequencer) onBuildStarted(x engine.BuildStartedEvent) {
	if x.DerivedFrom != (eth.L1BlockRef{}) {
		// If we are adding new blocks onto the tip of the chain, derived from L1,
		// then don't try to build on top of it immediately, as sequencer.
		d.log.Warn("Detected new block-building from L1 derivation, avoiding sequencing for now.",
			"build_job", x.Info.ID, "build_timestamp", x.Info.Timestamp,
			"parent", x.Parent, "derived_from", x.DerivedFrom)
		d.nextActionOK = false
		return
	}
	// Sequencer-originated builds are handled inline via direct engine calls.
	d.log.Debug("Ignoring build-started event of sequencer-originated block", "payloadID", x.Info.ID)
}

func (d *Sequencer) handleInvalid() {
	d.metrics.RecordSequencingError()
	d.latest = BuildingState{}
	// A block we sealed may be discarded here (invalid or denied insertion), and
	// will then never become the head. Reconcile the sealed marker so Stop, which
	// waits for the head to catch up to it, is not left waiting for a dead block.
	d.latestSealed = d.latestHead
	d.asyncGossip.Clear()
	// upon error, retry after one block worth of time
	blockTime := time.Duration(d.rollupCfg.BlockTime) * time.Second
	d.nextAction = d.timeNow().Add(blockTime)
	d.nextActionOK = d.active.Load()
}

func (d *Sequencer) onPayloadSuccess(x payloadSuccess) {
	// Sequencer-originated payloads are processed inline via direct engine calls.
	// For derivation-originated payloads (e.g. replacement blocks) we still clear
	// the async-gossip buffer, so the sequencer cannot reuse a stale payload after
	// a chain reset.
	if d.latest.Ref != (eth.L2BlockRef{}) && d.latest.Ref.Hash != x.blockHash {
		return // not relevant to us
	}
	d.latest = BuildingState{}
	d.asyncGossip.Clear()
}

// RunAction performs one sequencer action: it processes a payload left in the
// async-gossip buffer, seals the pending block-building job, or starts a new
// build. Engine interactions happen as direct synchronous calls, not events.
func (d *Sequencer) RunAction() {
	d.drainInbox() // decide from up-to-date state

	d.l.Lock()
	defer d.l.Unlock()

	preTime := d.nextAction
	preOk := d.nextActionOK
	defer func() {
		if d.nextActionOK != preOk || d.nextAction != preTime {
			d.log.Debug("Sequencer action schedule changed",
				"time", d.nextAction, "wait", d.nextAction.Sub(d.timeNow()), "ok", d.nextActionOK)
		}
	}()

	d.log.Debug("Sequencer action")
	if !d.active.Load() {
		d.log.Debug("Ignoring stale sequencer action while inactive")
		// Every exit must leave the schedule changed or disarmed, so the loop
		// never re-fires an unchanged deadline. Start/Stop normally keep these
		// two in step; this makes it structural rather than incidental.
		d.nextActionOK = false
		return
	}
	// The inbox drain above may have parked the schedule (reset, derivation
	// build, engine backoff, maxSafeLag stall). The timer fire that triggered
	// this action predates that state: honor the post-drain decision.
	if !d.nextActionOK {
		d.log.Debug("Ignoring sequencer action, no action scheduled after inbox replay")
		return
	}
	payload := d.asyncGossip.Get()
	if payload != nil {
		if d.latest.Info.ID == (eth.PayloadID{}) {
			d.log.Warn("Found reusable payload from async gossiper, and no block was being built. Reusing payload.",
				"hash", payload.ExecutionPayload.BlockHash,
				"number", uint64(payload.ExecutionPayload.BlockNumber),
				"parent", payload.ExecutionPayload.ParentHash)
		}
		ref, err := d.toBlockRef(d.rollupCfg, payload.ExecutionPayload)
		if err != nil {
			d.log.Error("Payload from async-gossip buffer could not be turned into block-ref", "err", err)
			// Treat like an invalid payload: drop it and retry with a fresh
			// build after a backoff. No event echo re-arms this failure.
			d.handleInvalid()
			return
		}
		d.log.Info("Resuming sequencing with previously async-gossip confirmed payload",
			"payload", payload.ExecutionPayload.ID())
		// The payload is known and was already gossiped: it must have been sealed
		// before a temporary error. Retry processing to make it canonical.
		// Park the schedule first: on a temporary error the engine's
		// EngineTemporaryErrorEvent echoes back through the mailbox and re-arms
		// the backoff; re-firing before that echo would spin on the engine.
		d.nextActionOK = false
		if err := d.eng.ProcessPayload(d.ctx, payload, ref, time.Time{}); err != nil {
			if errors.Is(err, engine.ErrStaleBuild) {
				// The chain moved past the buffered payload; it is worthless now.
				// Nothing of ours is outstanding anymore: don't leave Stop
				// waiting for the dropped block to become the head.
				d.latest = BuildingState{}
				d.latestSealed = d.latestHead
				d.asyncGossip.Clear()
				if ref.ParentHash != d.latestHead.Hash {
					// The payload was already stale against the head we know, so
					// the forkchoice update the engine just requested carries that
					// same head and will not re-plan anything. Re-plan here, or the
					// sequencer parks forever waiting for an echo it has had.
					d.scheduleNextAction(d.latestHead)
				}
				// Otherwise the engine moved during the call: its forkchoice update
				// names a head we have not seen yet and re-arms us on arrival.
			} else if errors.Is(err, engine.ErrPayloadDenied) || errors.Is(err, engine.ErrPayloadInvalid) {
				d.handleInvalid()
			}
			return
		}
		d.latest = BuildingState{}
		d.asyncGossip.Clear()
		d.scheduleNextAction(ref)
		return
	}
	if d.latest.Info != (eth.PayloadInfo{}) {
		// We should not repeat the seal request.
		d.nextActionOK = false
		sealResult, err := d.eng.SealBuild(d.ctx, d.latest.Info, d.latest.Started)
		if err != nil {
			if errors.Is(err, engine.ErrStaleBuild) {
				// A competing block landed while we were building; drop the job
				// without committing or gossiping the stale sibling. Parked until
				// the engine-requested forkchoice update arrives.
				d.log.Warn("Dropping stale sealed block, chain moved past it", "payloadID", d.latest.Info.ID)
				d.latest = BuildingState{}
				return
			}
			// Restart building on seal errors (expired or invalid), this way we get
			// a block we should be able to seal (smaller, since we adapt build time).
			d.handleInvalid()
			return
		}
		envelope := sealResult.Envelope
		d.log.Info("Sequencer sealed block", "payloadID", d.latest.Info.ID,
			"block", envelope.ExecutionPayload.ID(),
			"parent", envelope.ExecutionPayload.ParentID(),
			"txs", len(envelope.ExecutionPayload.Transactions),
			"time", uint64(envelope.ExecutionPayload.Timestamp))

		// generous timeout, the conductor is important
		commitCtx, cancel := context.WithTimeout(d.ctx, time.Second*30)
		defer cancel()
		if err := d.conductor.CommitUnsafePayload(commitCtx, envelope); err != nil {
			d.emitter.Emit(d.ctx, rollup.EngineTemporaryErrorEvent{
				Err: fmt.Errorf("failed to commit unsafe payload to conductor: %w", err),
			})
			return
		}

		// Begin gossiping as soon as possible.
		// asyncGossip.Clear() is called once the payload is successfully inserted below,
		// or when a later action hits a non-temporary error.
		d.asyncGossip.Gossip(envelope)
		d.latest.Ref = sealResult.Ref
		d.latestSealed = sealResult.Ref
		// Now after having gossiped the block, try to put it in our own canonical chain.
		if err := d.eng.ProcessPayload(d.ctx, envelope, sealResult.Ref, d.latest.Started); err != nil {
			if errors.Is(err, engine.ErrStaleBuild) {
				// A competing block landed after we sealed and gossiped; the
				// gossiped sibling lost. Drop it rather than reorging back to it.
				// Parked until the engine-requested forkchoice update arrives.
				// Reset latestSealed so Stop does not wait for the dropped
				// block to ever become the head.
				d.log.Warn("Dropping stale processed block, chain moved past it", "block", sealResult.Ref)
				d.latest = BuildingState{}
				d.latestSealed = d.latestHead
				d.asyncGossip.Clear()
			} else if errors.Is(err, engine.ErrPayloadDenied) || errors.Is(err, engine.ErrPayloadInvalid) {
				d.handleInvalid()
			}
			return
		}
		d.log.Info("Sequencer inserted block",
			"block", sealResult.Ref, "parent", envelope.ExecutionPayload.ParentID())
		d.latest = BuildingState{}
		// The payload was already published upon sealing.
		// Now that we have processed it ourselves we don't need it anymore.
		d.asyncGossip.Clear()
		d.scheduleNextAction(sealResult.Ref)
	} else if d.latest == (BuildingState{}) {
		// If we have not started building anything, start building.
		d.startBuildingBlock()
	}
}

func (d *Sequencer) onEngineTemporaryError(x rollup.EngineTemporaryErrorEvent) {
	if d.latest == (BuildingState{}) {
		d.log.Debug("Engine reported temporary error while building state is empty", "err", x.Err)
	}
	d.log.Error("Engine failed temporarily, backing off sequencer", "err", x.Err)
	if errors.Is(x.Err, engine.ErrEngineSyncing) { // if it is syncing we can back off by more
		d.nextAction = d.timeNow().Add(30 * time.Second)
	} else {
		d.nextAction = d.timeNow().Add(time.Second)
	}
	// A reset in flight keeps the sequencer parked: the engine has not rewound
	// yet, so re-arming here would resume building on the pre-reset head. The
	// confirmation carries the cool-down, and a backoff must not pre-empt it.
	d.nextActionOK = d.active.Load() && !d.awaitingResetConfirm
	// Re-check the lag bound: this is the only ingest-path re-arm, so it is the
	// only way a maxSafeLag-stalled sequencer can be released early.
	d.evalMaxSafeLag()
	// We don't explicitly cancel block building jobs upon temporary errors: we may still finish the block (if any).
	// Any unfinished block building work eventually times out, and will be cleaned up that way.
	// Note that this only applies to temporary errors upon starting a block-building job.
	// If the engine errors upon sealing, the seal-error handling restarts the build with fresh attributes.

	// If we don't have an ID of a job to resume, then start over.
	// (d.latest.Onto would be set if we emitted BuildStart already)
	if d.latest.Info == (eth.PayloadInfo{}) {
		d.latest = BuildingState{}
	}
}

func (d *Sequencer) onReset(x rollup.ResetEvent) {
	d.log.Error("Sequencer encountered reset signal, aborting work", "err", x.Err)
	d.metrics.RecordSequencerReset()
	// try to cancel any ongoing payload building job
	if d.latest.Info != (eth.PayloadInfo{}) {
		d.emitter.Emit(d.ctx, engine.BuildCancelEvent{Info: d.latest.Info})
	}
	d.latest = BuildingState{}
	d.stalledByMaxSafeLag = false
	// no action to perform until we get a reset-confirmation
	d.nextActionOK = false
	d.awaitingResetConfirm = true
}

func (d *Sequencer) onEngineResetConfirmedEvent(engine.EngineResetConfirmedEvent) {
	d.awaitingResetConfirm = false
	d.nextActionOK = d.active.Load()
	// Before sequencing we can wait a block,
	// assuming the execution-engine just churned through some work for the reset.
	// This will also prevent any potential reset-loop from running too hot.
	d.nextAction = d.timeNow().Add(time.Second * time.Duration(d.rollupCfg.BlockTime))
	// The reset delivered fresh forkchoice state just before this confirmation;
	// re-check the lag bound rather than unconditionally re-arming, so a still
	// badly lagging sequencer does not build one extra block before the next
	// forkchoice update re-stalls it. (onReset cleared stalledByMaxSafeLag.)
	d.evalMaxSafeLag()
	d.log.Info("Engine reset confirmed, sequencer may continue", "next", d.nextActionOK)
}

func (d *Sequencer) onForkchoiceUpdate(x engine.ForkchoiceUpdateEvent) {
	d.log.Debug("Sequencer is processing forkchoice update", "unsafe", x.UnsafeL2Head, "latest", d.latestHead)

	d.latestSafe = x.SafeL2Head
	if !d.active.Load() {
		d.setLatestHead(x.UnsafeL2Head)
		return
	}
	// Drop a stale block-building job if the chain no longer sits on its parent.
	// The head can move backwards (block replacement, backup-unsafe restore, an
	// engine reset), so compare identity, not height.
	if d.latest != (BuildingState{}) && d.latest.Onto.Hash != x.UnsafeL2Head.Hash {
		d.log.Debug("Dropping stale/completed block-building job",
			"state", d.latest.Onto, "unsafe_head", x.UnsafeL2Head)
		// The cleared state stops the next sequencer action from sealing the stale build job.
		d.latest = BuildingState{}
	}
	if x.UnsafeL2Head.Hash != d.latestHead.Hash && !d.awaitingResetConfirm {
		// Any change of head — including a rewind — needs a fresh plan on top of
		// it. Re-planning only on a higher block number would leave the sequencer
		// parked forever after a rewind, because the engine rejects builds and
		// seals that no longer extend its head and the recovering forkchoice
		// update is the rewound one.
		d.scheduleNextAction(x.UnsafeL2Head)
	} else {
		d.setLatestHead(x.UnsafeL2Head)
		// Re-evaluate the stall even when the head is unchanged: the resume
		// trigger is the safe head catching up, which arrives on exactly such
		// forkchoice updates. (scheduleNextAction evaluates it itself.)
		d.evalMaxSafeLag()
	}
}

// scheduleNextAction arms the sequencer to build the next block on top of the
// given new head, leaving spare time if the head is fresh enough.
// Called after a block insertion, either from RunAction (direct-call path)
// or from onForkchoiceUpdate (event path).
func (d *Sequencer) scheduleNextAction(newHead eth.L2BlockRef) {
	d.nextActionOK = true
	now := d.timeNow()
	blockTime := time.Duration(d.rollupCfg.BlockTime) * time.Second
	payloadTime := time.Unix(int64(newHead.Time+d.rollupCfg.BlockTime), 0)
	remainingTime := payloadTime.Sub(now)
	if remainingTime > blockTime {
		// if we have too much time, then wait before starting the build
		d.nextAction = payloadTime.Add(-blockTime)
	} else {
		// otherwise start instantly
		d.nextAction = now
	}
	d.setLatestHead(newHead)
	// The stall check must run after arming (nextActionOK=true above), so it
	// can override it. Evaluating here, not just on forkchoice updates, keeps
	// the lag bound enforced on the direct-call path too: the sequencer must
	// not outrun it while the stalling forkchoice echo is still queued.
	d.evalMaxSafeLag()
}

// evalMaxSafeLag stalls block production while the safe head (as of the last
// ingested forkchoice update) lags the unsafe head by more than the configured
// bound, and resumes it when the safe head catches up.
// Must run after any schedule arming so the stall can override it.
func (d *Sequencer) evalMaxSafeLag() {
	if maxSafeLag := d.maxSafeLag.Load(); maxSafeLag > 0 {
		if d.latestSafe.Number+maxSafeLag <= d.latestHead.Number {
			if !d.stalledByMaxSafeLag {
				d.log.Warn("sequencer has fallen behind safe head by more than lag, stalling",
					"head", d.latestHead, "safe", d.latestSafe, "max_lag", maxSafeLag)
			}
			d.nextActionOK = false
			d.stalledByMaxSafeLag = true
		} else if d.stalledByMaxSafeLag {
			// Safe head has caught up after a maxSafeLag stall, resume sequencing.
			// Only resume if we were stalled by maxSafeLag specifically, to avoid
			// interfering with other nextActionOK=false states (reset, L1-derivation backoff, etc).
			d.log.Info("safe head caught up, resuming sequencing",
				"head", d.latestHead, "safe", d.latestSafe, "max_lag", maxSafeLag)
			d.stalledByMaxSafeLag = false
			d.nextActionOK = d.active.Load()
			d.nextAction = d.timeNow()
		}
	} else if d.stalledByMaxSafeLag {
		// maxSafeLag was disabled at runtime (set to 0) while stalled; resume immediately.
		d.log.Info("maxSafeLag disabled, resuming sequencing")
		d.stalledByMaxSafeLag = false
		d.nextActionOK = d.active.Load()
		d.nextAction = d.timeNow()
	}
}

func (d *Sequencer) setLatestHead(head eth.L2BlockRef) {
	d.latestHead = head
	if d.latestHeadSet != nil {
		close(d.latestHeadSet)
		d.latestHeadSet = nil
	}
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (d *Sequencer) startBuildingBlock() {
	ctx := d.ctx
	l2Head := d.latestHead

	// If we do not have data to know what to build on, then request a forkchoice update.
	// Park until the head update arrives through the mailbox; re-firing on the
	// unchanged schedule would hot-loop forkchoice requests while the engine
	// state is still unknown (e.g. before the initial engine reset completes).
	if l2Head == (eth.L2BlockRef{}) {
		d.nextActionOK = false
		d.eng.RequestForkchoiceUpdate(d.ctx)
		return
	}

	recoverMode := d.recoverMode.Load()

	// Figure out which L1 origin block we're going to be building on top of.
	l1Origin, err := d.l1OriginSelector.FindL1Origin(ctx, l2Head)
	switch {
	case err == nil:
	case errors.Is(err, ErrInvalidL1Origin), errors.Is(err, ErrNextL1OriginOrphaned):
		d.metrics.RecordSequencerInconsistentL1Origin(l2Head.L1Origin, l1Origin.ID())
		// Park: the ResetEvent echoes back through the mailbox (onReset keeps us
		// parked until the reset is confirmed).
		d.nextActionOK = false
		d.emitter.Emit(d.ctx, rollup.ResetEvent{
			Err: fmt.Errorf("cannot build new L2 block with L1 origin %s (parent L1 %s) on current L2 head %s with L1 origin %s",
				l1Origin, l1Origin.ParentHash, l2Head, l2Head.L1Origin),
		})
		return
	case errors.Is(err, ErrNextL1OriginRequired):
		fallthrough
	default:
		d.nextAction = d.timeNow().Add(time.Second)
		d.nextActionOK = d.active.Load()
		d.log.Error("Error finding next L1 Origin", "err", err)
		d.emitter.Emit(d.ctx, rollup.L1TemporaryErrorEvent{Err: err})
		return
	}

	d.log.Info("Started sequencing new block", "parent", l2Head, "l1Origin", l1Origin)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	attrs, err := d.attrBuilder.PreparePayloadAttributes(fetchCtx, l2Head, l1Origin.ID())
	if err != nil {
		// Park before emitting: recovery is driven by the event echoing back
		// through the mailbox (temp-error backoff re-arms, reset stays parked
		// until confirmed). An armed past deadline would spin the loop until
		// the echo arrives.
		d.nextActionOK = false
		if errors.Is(err, derive.ErrTemporary) {
			d.emitter.Emit(d.ctx, rollup.EngineTemporaryErrorEvent{Err: err})
			return
		} else if errors.Is(err, derive.ErrReset) {
			d.emitter.Emit(d.ctx, rollup.ResetEvent{Err: err})
			return
		} else if errors.Is(err, derive.ErrCritical) {
			d.emitter.Emit(d.ctx, rollup.CriticalErrorEvent{Err: err})
			return
		} else {
			d.emitter.Emit(d.ctx, rollup.CriticalErrorEvent{
				Err: fmt.Errorf("unexpected attributes-preparation error: %w", err),
			})
			return
		}
	}

	// If our next L2 block timestamp is beyond the Sequencer drift threshold, then we must produce
	// empty blocks (other than the L1 info deposit and any user deposits). We handle this by
	// setting NoTxPool to true, which will cause the Sequencer to not include any transactions
	// from the transaction pool.
	attrs.NoTxPool = uint64(attrs.Timestamp) > l1Origin.Time+d.spec.MaxSequencerDrift(l1Origin.Time)

	// For the Ecotone activation block we shouldn't include any sequencer transactions.
	if d.rollupCfg.IsEcotoneActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		d.log.Info("Sequencing Ecotone upgrade block")
	}

	// For the Fjord activation block we shouldn't include any sequencer transactions.
	if d.rollupCfg.IsFjordActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		d.log.Info("Sequencing Fjord upgrade block")
	}

	// For the Granite activation block we can include sequencer transactions.
	if d.rollupCfg.IsGraniteActivationBlock(uint64(attrs.Timestamp)) {
		d.log.Info("Sequencing Granite upgrade block")
	}

	// For the Isthmus activation block we shouldn't include any sequencer transactions.
	if d.rollupCfg.IsIsthmusActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		d.log.Info("Sequencing Isthmus upgrade block")
	}

	// For the Jovian activation block we must not include any sequencer transactions.
	if d.rollupCfg.IsJovianActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		d.log.Info("Sequencing Jovian upgrade block")
	}

	// For the Karst activation block we must not include any sequencer transactions.
	if d.rollupCfg.IsKarstActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		d.log.Info("Sequencing Karst upgrade block")
	}

	// For the Lagoon activation block we must not include any sequencer transactions.
	if d.rollupCfg.IsInteropActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		d.log.Info("Sequencing Lagoon upgrade block")
	}

	if recoverMode {
		attrs.NoTxPool = true
		d.log.Warn("Sequencing temporarily without user transactions, in recover mode")
	}

	d.log.Debug("prepared attributes for new block",
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
	d.nextActionOK = false

	// Reset building state, and remember what we are building on.
	// If we get a forkchoice update that conflicts, we will have to abort building.
	d.latest = BuildingState{Onto: l2Head}

	result, err := d.eng.StartBuild(d.ctx, withParent)
	if err != nil {
		if errors.Is(err, engine.ErrStaleBuild) {
			// The engine head moved past our build target; the engine already requested
			// a forkchoice update, so wait for the next head update to re-plan.
			d.log.Warn("Engine rejected stale build attempt, waiting for next head update",
				"parent", l2Head, "err", err)
			d.latest = BuildingState{}
			return
		}
		if errors.Is(err, engine.ErrBuildInvalid) {
			// No recovery event reaches us for invalid attributes of our own
			// builds; back off locally and retry with a new job.
			d.handleInvalid()
			return
		}
		// Temporary, reset, or critical error: the engine emitted the matching
		// event, which echoes back through the mailbox and re-arms or keeps us
		// parked as appropriate. Stay parked (set before StartBuild) meanwhile.
		return
	}
	if d.latest.Onto != result.Parent {
		d.log.Warn("Canceling stale block-building job that was just started, as target to build onto has changed",
			"stale", result.Parent, "new", d.latest.Onto, "job_id", result.Info.ID, "job_timestamp", result.Info.Timestamp)
		d.emitter.Emit(d.ctx, engine.BuildCancelEvent{
			Info:  result.Info,
			Force: true,
		})
		d.handleInvalid()
		return
	}
	d.log.Debug("Sequencer started building new block",
		"payloadID", result.Info.ID, "parent", result.Parent, "parent_time", result.Parent.Time)
	d.latest.Info = result.Info
	d.latest.Started = result.BuildStarted

	d.nextActionOK = d.active.Load()

	// schedule sealing
	now := d.timeNow()
	payloadTime := time.Unix(int64(result.Parent.Time+d.rollupCfg.BlockTime), 0)
	remainingTime := payloadTime.Sub(now)
	if remainingTime < d.sealingDuration {
		d.nextAction = now // if there's not enough time for sealing, don't wait.
	} else {
		// finish with margin of sealing duration before payloadTime
		d.nextAction = payloadTime.Add(-d.sealingDuration)
	}
}

func (d *Sequencer) NextAction() (t time.Time, ok bool) {
	d.l.Lock()
	defer d.l.Unlock()
	return d.nextAction, d.nextActionOK
}

// Building returns the block-building job the sequencer currently has in
// flight. The zero value means no job is in flight.
func (d *Sequencer) Building() BuildingState {
	d.l.Lock()
	defer d.l.Unlock()
	return d.latest
}

func (d *Sequencer) Active() bool {
	return d.active.Load()
}

func (d *Sequencer) Start(ctx context.Context, head common.Hash) error {
	// must be leading to activate
	if isLeader, err := d.conductor.Leader(ctx); err != nil {
		return fmt.Errorf("sequencer leader check failed: %w", err)
	} else if !isLeader {
		return errors.New("sequencer is not the leader, aborting")
	}

	// Note: leader check happens before locking; this is how the Driver used to work,
	// and prevents the event-processing of the sequencer from being stalled due to a potentially slow conductor call.
	if err := d.l.LockCtx(ctx); err != nil {
		return err
	}
	defer d.l.Unlock()

	if d.active.Load() {
		return ErrSequencerAlreadyStarted
	}
	if d.latestHead == (eth.L2BlockRef{}) {
		return fmt.Errorf("no prestate, cannot determine if sequencer start at %s is safe", head)
	}
	if head != d.latestHead.Hash {
		return fmt.Errorf("block hash does not match: head %s, received %s", d.latestHead, head)
	}
	return d.forceStart()
}

func (d *Sequencer) Init(ctx context.Context, active bool) error {
	d.l.Lock()
	defer d.l.Unlock()

	d.asyncGossip.Start()

	// The `latestHead` should be updated, so we can handle start-sequencer requests
	d.eng.RequestForkchoiceUpdate(d.ctx)

	if active {
		return d.forceStart()
	} else {
		d.metrics.SetSequencerState(false)
		if err := d.listener.SequencerStopped(); err != nil {
			return fmt.Errorf("failed to notify sequencer-state listener of initial stopped state: %w", err)
		}
		return nil
	}
}

// forceStart skips all the checks, and just starts the sequencer
func (d *Sequencer) forceStart() error {
	if d.latestHead == (eth.L2BlockRef{}) {
		// This happens if sequencing is activated on op-node startup.
		// The op-conductor check and choice of sequencing with this pre-state already happened before op-node startup.
		d.log.Info("Starting sequencing, without known pre-state")
		d.asyncGossip.Clear() // if we are starting from an unknown pre-state, just clear gossip out of caution.
	} else {
		// This happens when we start sequencing on an already-running node.
		d.log.Info("Starting sequencing on top of known pre-state", "head", d.latestHead)
		if payload := d.asyncGossip.Get(); payload != nil &&
			payload.ExecutionPayload.BlockHash != d.latestHead.Hash {
			d.log.Warn("Cleared old block from async-gossip buffer, sequencing pre-state is different",
				"buffered", payload.ExecutionPayload.ID(), "prestate", d.latestHead)
			d.asyncGossip.Clear()
		}
	}

	if err := d.listener.SequencerStarted(); err != nil {
		return fmt.Errorf("failed to notify sequencer-state listener of start: %w", err)
	}
	// clear the building state; interrupting any existing sequencing job (there should never be one)
	d.latest = BuildingState{}
	// Same reset rule as the error backoff: if a reset is unconfirmed, latestHead
	// is still the pre-reset head (which is exactly what Start checks against), so
	// arming now would sequence on a chain about to be rewound. The confirmation
	// arms us on the new head.
	d.nextActionOK = !d.awaitingResetConfirm
	d.stalledByMaxSafeLag = false
	d.nextAction = d.timeNow()
	d.active.Store(true)
	d.metrics.SetSequencerState(true)
	d.wake() // wake the sequencer goroutine, so it picks up the new schedule
	d.log.Info("Sequencer has been started", "next action", d.nextAction)
	return nil
}

func (d *Sequencer) Stop(ctx context.Context) (common.Hash, error) {
	if err := d.l.LockCtx(ctx); err != nil {
		return common.Hash{}, err
	}

	if !d.active.Load() {
		d.l.Unlock()
		return common.Hash{}, ErrSequencerAlreadyStopped
	}

	// ensure latestHead has been updated to the latest sealed/gossiped block before stopping the sequencer
	for d.latestHead.Hash != d.latestSealed.Hash {

		// if we are not the leader, latestSealed will never be updated and we will wait forever
		if isLeader, err := d.conductor.Leader(ctx); err != nil {
			d.log.Warn("Could not determine leadership while stopping. Skipping wait.", "err", err)
			break
		} else if !isLeader {
			d.log.Info("Not leader anymore, skipping head sync wait")
			break
		}

		latestHeadSet := make(chan struct{})
		d.latestHeadSet = latestHeadSet
		d.l.Unlock()
		select {
		case <-ctx.Done():
			return common.Hash{}, ctx.Err()
		case <-latestHeadSet:
		}
		if err := d.l.LockCtx(ctx); err != nil {
			return common.Hash{}, err
		}
	}
	defer d.l.Unlock()

	// Stop() may have been called twice, so check if we are active after reacquiring the lock
	if !d.active.Load() {
		return common.Hash{}, ErrSequencerAlreadyStopped
	}

	if err := d.listener.SequencerStopped(); err != nil {
		return common.Hash{}, fmt.Errorf("failed to notify sequencer-state listener of stop: %w", err)
	}

	// Cancel any inflight block building. If we don't cancel this, we can resume sequencing an old block
	// even if we've received new unsafe heads in the interim, causing us to introduce a re-org.
	d.latest = BuildingState{} // By wiping this state we cannot continue from it later.

	d.nextActionOK = false
	d.stalledByMaxSafeLag = false
	d.active.Store(false)
	d.metrics.SetSequencerState(false)
	d.wake() // wake the sequencer goroutine, so it parks its action timer
	d.log.Info("Sequencer has been stopped")
	return d.latestHead.Hash, nil
}

func (d *Sequencer) SetMaxSafeLag(ctx context.Context, v uint64) error {
	if err := d.l.LockCtx(ctx); err != nil {
		return err
	}
	defer d.l.Unlock()
	d.maxSafeLag.Store(v)
	// Apply the new bound immediately instead of waiting for the next forkchoice
	// update: while stalled the sequencer goroutine is parked with no timer armed,
	// so relaxing or disabling the bound would otherwise not resume it at all.
	d.evalMaxSafeLag()
	d.wake()
	return nil
}

func (d *Sequencer) OverrideLeader(ctx context.Context) error {
	return d.conductor.OverrideLeader(ctx)
}

func (d *Sequencer) ConductorEnabled(ctx context.Context) bool {
	return d.conductor.Enabled(ctx)
}

func (d *Sequencer) SetRecoverMode(mode bool) {
	d.l1OriginSelector.SetRecoverMode(mode)
	d.recoverMode.Store(mode)
}

func (d *Sequencer) Close() {
	d.conductor.Close()
	d.asyncGossip.Stop()
}
