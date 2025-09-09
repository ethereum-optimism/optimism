package sequencing

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/protolambda/ctxlock"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

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

	log       log.Logger
	rollupCfg *rollup.Config
	spec      *rollup.ChainSpec

	maxSafeLag atomic.Uint64

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

	eng    attributes.EngineController
	engine engine.ExecEngine

	attrBuilder      derive.AttributesBuilder
	l1OriginSelector L1OriginSelectorIface

	metrics Metrics

	// timeNow enables sequencer testing to mock the time
	timeNow func() time.Time

	// nextAction is when the next sequencing action should be performed
	nextAction   time.Time
	nextActionOK bool

	latest       BuildingState
	latestSealed eth.L2BlockRef
	latestHead   eth.L2BlockRef

	latestHeadSet chan struct{}

	// toBlockRef converts a payload to a block-ref, and is only configurable for test-purposes
	toBlockRef func(rollupCfg *rollup.Config, payload *eth.ExecutionPayload) (eth.L2BlockRef, error)
}

var _ SequencerIface = (*Sequencer)(nil)

func NewSequencer(driverCtx context.Context, log log.Logger, rollupCfg *rollup.Config,
	attributesBuilder derive.AttributesBuilder,
	l1OriginSelector L1OriginSelectorIface,
	listener SequencerStateListener,
	conductor conductor.SequencerConductor,
	asyncGossip AsyncGossiper,
	metrics Metrics,
	eng attributes.EngineController,
	engine engine.ExecEngine,
) *Sequencer {
	return &Sequencer{
		ctx:              driverCtx,
		log:              log,
		rollupCfg:        rollupCfg,
		spec:             rollup.NewChainSpec(rollupCfg),
		listener:         listener,
		conductor:        conductor,
		asyncGossip:      asyncGossip,
		attrBuilder:      attributesBuilder,
		l1OriginSelector: l1OriginSelector,
		metrics:          metrics,
		eng:              eng,
		timeNow:          time.Now,
		toBlockRef:       derive.PayloadToBlockRef,
		engine:           engine,
	}
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
	d.nextActionOK = true
	d.nextAction = d.timeNow()
	d.active.Store(true)
	d.metrics.SetSequencerState(true)
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
	d.active.Store(false)
	d.metrics.SetSequencerState(false)
	d.log.Info("Sequencer has been stopped")
	return d.latestHead.Hash, nil
}

func (d *Sequencer) AttachEmitter(em event.Emitter) {
	d.emitter = em
}

// StartBuildingBlock initiates a block building job on top of the given L2 head, safe and finalized blocks, and using the provided l1Origin.
func (d *Sequencer) startBuildingBlock() {
	ctx := d.ctx
	l2Head := d.latestHead

	// If we do not have data to know what to build on, then request a forkchoice update
	if l2Head == (eth.L2BlockRef{}) {
		d.eng.RequestForkchoiceUpdate(d.ctx)
		return
	}
	// If we have already started trying to build on top of this block, we can avoid starting over again.
	if d.latest.Onto == l2Head {
		return
	}

	recoverMode := d.recoverMode.Load()

	// Figure out which L1 origin block we're going to be building on top of.
	l1Origin, err := d.l1OriginSelector.FindL1Origin(ctx, l2Head)
	if err != nil {
		d.nextAction = d.timeNow().Add(time.Second)
		d.nextActionOK = d.active.Load()
		d.log.Error("Error finding next L1 Origin", "err", err)
		d.emitter.Emit(d.ctx, rollup.L1TemporaryErrorEvent{Err: err})
		return
	}

	if !(l2Head.L1Origin.Hash == l1Origin.ParentHash || l2Head.L1Origin.Hash == l1Origin.Hash) {
		d.metrics.RecordSequencerInconsistentL1Origin(l2Head.L1Origin, l1Origin.ID())
		d.emitter.Emit(d.ctx, rollup.ResetEvent{
			Err: fmt.Errorf("cannot build new L2 block with L1 origin %s (parent L1 %s) on current L2 head %s with L1 origin %s",
				l1Origin, l1Origin.ParentHash, l2Head, l2Head.L1Origin),
		})
		return
	}

	d.log.Info("Started sequencing new block", "parent", l2Head, "l1Origin", l1Origin)

	fetchCtx, cancel := context.WithTimeout(ctx, time.Second*20)
	defer cancel()

	attrs, err := d.attrBuilder.PreparePayloadAttributes(fetchCtx, l2Head, l1Origin.ID())
	if err != nil {
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

	// For the Interop activation block we must not include any sequencer transactions.
	if d.rollupCfg.IsInteropActivationBlock(uint64(attrs.Timestamp)) {
		attrs.NoTxPool = true
		d.log.Info("Sequencing Interop upgrade block")
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

	d.emitter.Emit(d.ctx, engine.BuildStartEvent{
		Attributes: withParent,
	})
}

func (d *Sequencer) NextAction() (t time.Time, ok bool) {
	d.l.Lock()
	defer d.l.Unlock()
	return d.nextAction, d.nextActionOK
}

func (d *Sequencer) Active() bool {
	return d.active.Load()
}

func (d *Sequencer) SetMaxSafeLag(ctx context.Context, v uint64) error {
	d.maxSafeLag.Store(v)
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

func (d *Sequencer) handleInvalid() {
	d.metrics.RecordSequencingError()
	d.latest = BuildingState{}
	d.asyncGossip.Clear()
	// upon error, retry after one block worth of time
	blockTime := time.Duration(d.rollupCfg.BlockTime) * time.Second
	d.nextAction = d.timeNow().Add(blockTime)
	d.nextActionOK = d.active.Load()
}

func (d *Sequencer) setLatestHead(head eth.L2BlockRef) {
	d.latestHead = head
	if d.latestHeadSet != nil {
		close(d.latestHeadSet)
		d.latestHeadSet = nil
	}
}
