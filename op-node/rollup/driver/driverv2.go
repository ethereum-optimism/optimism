package driver

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"

	async2 "github.com/ethereum-optimism/optimism/op-node/rollup/async"
	"github.com/ethereum-optimism/optimism/op-node/rollup/conductor"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/async"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// TODO: it would be nice if we can enhance this to RLock with ctx timeout.
type LockedL2Ref struct {
	eth.L2BlockRef
	sync.RWMutex
}

type LockedL1Ref struct {
	eth.L1BlockRef
	sync.RWMutex
}

type DriverCondition struct {
	name string
	*async.RepeatCond
}

type EnginelessPipeline interface {
	Reset()
}

type DriverV2 struct {
	log log.Logger

	cfg *Config

	unsafeHead    LockedL2Ref
	safeHead      LockedL2Ref
	finalizedHead LockedL2Ref
	pendingHead   LockedL2Ref

	currentL1   LockedL1Ref
	finalizedL1 LockedL1Ref
	headL1      LockedL1Ref
	safeL1      LockedL1Ref

	// See specs draft in https://github.com/ethereum-optimism/specs/blob/664e92361ab7a1b1021660a1f67db8cdd50ba887/specs/interop/driver.md
	// This section enumerates all condition/effect pairs.
	attributesGeneration      *DriverCondition
	unsafeBlockSyncTrigger    *DriverCondition
	unsafeBlockProcessor      *DriverCondition
	sequencerAction           *DriverCondition
	attributesForceProcessing *DriverCondition
	safetyProgression         *DriverCondition
	safetyReversal            *DriverCondition
	finalityProgression       *DriverCondition
	engineConsistency         *DriverCondition

	conditions []*DriverCondition

	pipeline     EnginelessPipeline
	pipelineLock sync.Mutex // unfortunately we need a lock, since both attributes-generation and manual RPC may reset.

	sequencer SequencerIface

	// async gossiper for payloads to be gossiped without
	// blocking the event loop or waiting for insertion
	asyncGossiper      async2.AsyncGossiper
	sequencerConductor conductor.SequencerConductor

	engineController *derive.EngineController

	l2 L2Chain

	// TODO interop repeat-conditions: (also modify some of the above effects)
	// TODO interop safety progression
	// TODO interop safety reversal

	payloads     *derive.PayloadsQueue
	payloadsLock sync.RWMutex

	lifetimeCtx    context.Context
	lifetimeCancel context.CancelCauseFunc

	starter sync.Once

	closeWg sync.WaitGroup
}

var _ Driver = (*DriverV2)(nil)

func NewDriverV2(log log.Logger,
	sequencer SequencerIface,
	asyncGossiper async2.AsyncGossiper,
	sequencerConductor conductor.SequencerConductor,
	engineController *derive.EngineController,
	l2 L2Chain,
) *DriverV2 {
	lifetimeCtx, lifetimeCancel := context.WithCancelCause(context.Background())
	d := &DriverV2{
		log:                log,
		lifetimeCtx:        lifetimeCtx,
		lifetimeCancel:     lifetimeCancel,
		payloads:           derive.NewPayloadsQueue(log, derive.MaxUnsafePayloadsMemory, derive.PayloadMemSize),
		pipeline:           nil, // TODO
		sequencer:          sequencer,
		asyncGossiper:      asyncGossiper,
		sequencerConductor: sequencerConductor,
		engineController:   engineController,
		l2:                 l2,
	}

	todoCondition := func() bool {
		// TODO
		return false
	}
	todoEffect := func() {
		// TODO
	}
	d.attributesGeneration = d.registerCondition("attributes generation", &d.currentL1, d.checkGenerateAttributes, d.doGenerateAttributes)
	d.unsafeBlockSyncTrigger = d.registerCondition("unsafe block sync trigger", &d.unsafeHead, d.checkUnsafeBlockSyncTrigger, d.doUnsafeBlockSyncTrigger)
	d.unsafeBlockProcessor = d.registerCondition("unsafe block processor", &d.unsafeHead, d.checkUnsafeBlockProcessing, d.doUnsafeBlockProcessing)
	d.sequencerAction = d.registerCondition("sequencer action", &d.unsafeHead, d.checkSequencerAction, d.doSequencerAction)
	d.attributesForceProcessing = d.registerCondition("attributes force processing", &d.unsafeHead, todoCondition, todoEffect)
	d.safetyProgression = d.registerCondition("safety progression", &d.safeHead, todoCondition, todoEffect)
	d.safetyReversal = d.registerCondition("safety reversal", &d.safeHead, todoCondition, todoEffect)
	d.finalityProgression = d.registerCondition("finality progression", &d.finalizedHead, todoCondition, todoEffect)
	d.engineConsistency = d.registerCondition("engine consistency", &d.unsafeHead, todoCondition, todoEffect)
	// Design note: more conditions / effects can be registered.
	// And optionally based on hardforks, feature-flags, OP-Stack forks, etc.
	return d
}

func (d *DriverV2) registerCondition(name string, locker sync.Locker, conditional func() bool, effect func()) *DriverCondition {
	c := async.NewRepeatCond(d.lifetimeCtx, locker, func() bool {
		d.log.Debug("driver condition start", "name", name)
		startTime := time.Now()
		v := conditional()
		d.log.Debug("driver condition end", "name", name, "value", v, "elapsed", time.Since(startTime))
		return v
	}, func() {
		d.log.Debug("driver effect start", "name", name)
		startTime := time.Now()
		effect()
		d.log.Debug("driver effect end", "name", name, "elapsed", time.Since(startTime))
	})
	dc := &DriverCondition{name: name, RepeatCond: c}
	d.conditions = append(d.conditions, dc)
	return dc
}

// Start starts the driver operations: all condition/effects will be applicable, and signaled once.
func (d *DriverV2) Start() error {
	startProcess := func(r *DriverCondition) {
		d.closeWg.Add(1)

		go func() {
			<-r.Ctx().Done()
			if err := context.Cause(r.Ctx()); err != nil && !errors.Is(err, context.Canceled) {
				d.log.Error("driver process failed", "name", r.name, "err", err)
			}
			d.closeWg.Done()
		}()

		r.Start()
		r.Signal() // warm up, we may already have hit the condition
	}

	d.starter.Do(func() {
		for _, cond := range d.conditions {
			startProcess(cond)
		}
	})
	return nil
}

func (d *DriverV2) Close() error {
	// start closing all processes
	d.lifetimeCancel(context.Canceled)
	// wait for all processes to close
	d.closeWg.Wait()

	// collect errors from processes
	var result error
	for _, cond := range d.conditions {
		if err := cond.Ctx().Err(); err != nil && !errors.Is(err, context.Canceled) {
			result = errors.Join(result, fmt.Errorf("engine driver process %q close error: %w", cond.name, err))
		}
	}

	return result
}

func (d *DriverV2) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	// TODO: these read locks should be quick, but we may still want to respect the ctx,
	// that we introduced for v1 having prolonged global contention between everything.

	// We grab all locks, for a single global consistent view of the sync-status.
	d.unsafeHead.RLock()
	defer d.unsafeHead.RUnlock()
	d.safeHead.RLock()
	defer d.safeHead.RUnlock()
	d.finalizedHead.RLock()
	defer d.finalizedHead.RUnlock()
	d.currentL1.RLock()
	defer d.currentL1.RUnlock()
	d.finalizedL1.RLock()
	defer d.finalizedL1.RUnlock()

	return &eth.SyncStatus{
		CurrentL1:          d.currentL1.L1BlockRef,
		CurrentL1Finalized: d.finalizedL1.L1BlockRef,
		HeadL1:             d.headL1.L1BlockRef,
		SafeL1:             d.safeL1.L1BlockRef,
		FinalizedL1:        d.finalizedL1.L1BlockRef, // TODO what's the difference again?
		UnsafeL2:           d.unsafeHead.L2BlockRef,
		SafeL2:             d.safeHead.L2BlockRef,
		FinalizedL2:        d.finalizedHead.L2BlockRef,
		PendingSafeL2:      d.pendingHead.L2BlockRef,
	}, nil
}

func (d *DriverV2) BlockRefWithStatus(ctx context.Context, num uint64) (eth.L2BlockRef, *eth.SyncStatus, error) {
	// special case for finalized blocks: don't lock the unsafe L2 chain, if we are fetching finalized data.
	d.finalizedHead.RLock()
	defer d.finalizedHead.RUnlock()

	// don't lock the unsafe part if the block we are fetching is finalized and not going to change.
	if num > d.finalizedHead.Number {
		d.unsafeHead.RLock()
		tip := d.unsafeHead.L2BlockRef
		defer d.unsafeHead.RUnlock() // unlock after having fetched the block and sync-status, for consistency.
		if tip.Number < num {
			return eth.L2BlockRef{}, nil, ethereum.NotFound
		}
	}

	ref, err := d.l2.L2BlockRefByNumber(ctx, num)
	if err != nil {
		return eth.L2BlockRef{}, nil, fmt.Errorf("failed to fetch block %d: %w", num, err)
	}
	status, err := d.SyncStatus(ctx)
	if err != nil {
		return eth.L2BlockRef{}, nil, err
	}
	return ref, status, nil
}

func (d *DriverV2) ResetDerivationPipeline(ctx context.Context) error {
	d.pipelineLock.Lock()
	defer d.pipelineLock.Unlock()
	d.pipeline.Reset()
	return nil
}

func (d *DriverV2) StartSequencer(ctx context.Context, blockHash common.Hash) error {
	// Grab the lock over the tip of the chain. This ensures we aren't currently doing an open/seal sequencing step.
	d.unsafeHead.Lock()
	defer d.unsafeHead.Unlock()
	if d.cfg.SequencerStopped {
		return ErrSequencerAlreadyStopped
	}
	// Ensure that the request is consistent;
	// we don't want to continue sequencing on an older part of the chain.
	if d.unsafeHead.Hash != blockHash {
		return fmt.Errorf("block hash does not match: head %s, received %s", d.unsafeHead.Hash, blockHash)
	}
	d.cfg.SequencerEnabled = true
	d.cfg.SequencerStopped = false
	// signal that we may start a new block sequencing job
	d.sequencerAction.Signal()
	return nil
}

func (d *DriverV2) StopSequencer(ctx context.Context) (common.Hash, error) {
	// Grab the lock over the tip of the chain. This ensures we aren't currently doing an open/seal sequencing step.
	d.unsafeHead.Lock()
	defer d.unsafeHead.Unlock()

	if d.cfg.SequencerStopped {
		return common.Hash{}, ErrSequencerAlreadyStopped
	}

	// Cancel any inflight block building. If we don't cancel this, we can resume sequencing an old block
	// even if we've received new unsafe heads in the interim, causing us to introduce a re-org.
	d.sequencer.CancelBuildingBlock(ctx)

	d.cfg.SequencerStopped = true

	return d.unsafeHead.Hash, nil
}

func (d *DriverV2) SequencerActive(ctx context.Context) (bool, error) {
	d.unsafeHead.RLock()
	defer d.unsafeHead.RUnlock()
	return d.cfg.SequencerEnabled && !d.cfg.SequencerStopped, nil
}

func (d *DriverV2) OnUnsafeL2Payload(ctx context.Context, payload *eth.ExecutionPayloadEnvelope) error {
	d.payloadsLock.Lock()
	defer d.payloadsLock.Unlock()
	err := d.payloads.Push(payload)
	if err != nil {
		return err
	}
	d.onNewUnsafeBlock()
	return nil
}

func (d *DriverV2) OnL1Head(ctx context.Context, unsafe eth.L1BlockRef) error {
	d.headL1.Lock()
	defer d.headL1.Unlock()
	d.headL1.L1BlockRef = unsafe
	// signal we may be able to generate new L2 data from L1
	d.attributesGeneration.Signal()
	return nil
}

func (d *DriverV2) OnL1Safe(ctx context.Context, safe eth.L1BlockRef) error {
	d.safeL1.Lock()
	defer d.safeL1.Unlock()
	d.safeL1.L1BlockRef = safe
	// we only maintain the L1 safe head for debugging info, it doesn't affect L2 state or safety.
	return nil
}

func (d *DriverV2) OnL1Finalized(ctx context.Context, finalized eth.L1BlockRef) error {
	d.finalizedL1.Lock()
	defer d.finalizedL1.Unlock()
	d.finalizedL1.L1BlockRef = finalized
	// signal we may finalize L2 now
	d.finalityProgression.Signal()
	return nil
}
