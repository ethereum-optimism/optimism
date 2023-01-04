package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	gosync "sync"
	"time"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/backoff"
)

// Deprecated: use eth.SyncStatus instead.
type SyncStatus = eth.SyncStatus

// sealingDuration defines the expected time it takes to seal the block
const sealingDuration = time.Millisecond * 50

var UninitializedL1StateErr = errors.New("the L1 Head in L1 State is not initialized yet")

type Driver struct {
	l1State L1StateIface

	// The derivation pipeline is reset whenever we reorg.
	// The derivation pipeline determines the new l2Safe.
	derivation DerivationPipeline

	// When the derivation pipeline is waiting for new data to do anything
	idleDerivation bool

	// Requests to block the event loop for synchronous execution to avoid reading an inconsistent state
	stateReq chan chan struct{}

	// Upon receiving a channel in this channel, the derivation pipeline is forced to be reset.
	// It tells the caller that the reset occurred by closing the passed in channel.
	forceReset chan chan struct{}

	// Rollup config: rollup chain configuration
	config *rollup.Config

	// Driver config: verifier and sequencer settings
	driverConfig *Config

	// L1 Signals:
	//
	// Not all L1 blocks, or all changes, have to be signalled:
	// the derivation process traverses the chain and handles reorgs as necessary,
	// the driver just needs to be aware of the *latest* signals enough so to not
	// lag behind actionable data.
	l1HeadSig      chan eth.L1BlockRef
	l1SafeSig      chan eth.L1BlockRef
	l1FinalizedSig chan eth.L1BlockRef

	// L2 Signals:
	unsafeL2Payloads chan *eth.ExecutionPayload

	l1               L1Chain
	l2               L2Chain
	l1OriginSelector L1OriginSelectorIface
	sequencer        SequencerIface
	network          Network // may be nil, network for is optional

	metrics     Metrics
	log         log.Logger
	snapshotLog log.Logger
	done        chan struct{}

	wg gosync.WaitGroup
}

// Start starts up the state loop.
// The loop will have been started iff err is not nil.
func (s *Driver) Start() error {
	s.derivation.Reset()

	s.wg.Add(1)
	go s.eventLoop()

	return nil
}

func (s *Driver) Close() error {
	s.done <- struct{}{}
	s.wg.Wait()
	return nil
}

// OnL1Head signals the driver that the L1 chain changed the "unsafe" block,
// also known as head of the chain, or "latest".
func (s *Driver) OnL1Head(ctx context.Context, unsafe eth.L1BlockRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.l1HeadSig <- unsafe:
		return nil
	}
}

// OnL1Safe signals the driver that the L1 chain changed the "safe",
// also known as the justified checkpoint (as seen on L1 beacon-chain).
func (s *Driver) OnL1Safe(ctx context.Context, safe eth.L1BlockRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.l1SafeSig <- safe:
		return nil
	}
}

func (s *Driver) OnL1Finalized(ctx context.Context, finalized eth.L1BlockRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.l1FinalizedSig <- finalized:
		return nil
	}
}

func (s *Driver) OnUnsafeL2Payload(ctx context.Context, payload *eth.ExecutionPayload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.unsafeL2Payloads <- payload:
		return nil
	}
}

// startNewL2Block starts sequencing a new L2 block on top of the unsafe L2 Head.
func (s *Driver) startNewL2Block(ctx context.Context) error {
	l2Head := s.derivation.UnsafeL2Head()

	l1Head := s.l1State.L1Head()
	if l1Head == (eth.L1BlockRef{}) {
		return UninitializedL1StateErr
	}

	// Figure out which L1 origin block we're going to be building on top of.
	l1Origin, err := s.l1OriginSelector.FindL1Origin(ctx, l1Head, l2Head)
	if err != nil {
		s.log.Error("Error finding next L1 Origin", "err", err)
		return err
	}

	// Rollup is configured to not start producing blocks until a specific L1 block has been
	// reached. Don't produce any blocks until we're at that genesis block.
	if l1Origin.Number < s.config.Genesis.L1.Number {
		s.log.Info("Skipping block production because the next L1 Origin is behind the L1 genesis", "next", l1Origin.ID(), "genesis", s.config.Genesis.L1)
		return fmt.Errorf("the L1 origin %s cannot be before genesis at %s", l1Origin, s.config.Genesis.L1)
	}

	// Should never happen. Sequencer will halt if we get into this situation somehow.
	nextL2Time := l2Head.Time + s.config.BlockTime
	if nextL2Time < l1Origin.Time {
		s.log.Error("Cannot build L2 block for time before L1 origin",
			"l2Unsafe", l2Head, "nextL2Time", nextL2Time, "l1Origin", l1Origin, "l1OriginTime", l1Origin.Time)
		return fmt.Errorf("cannot build L2 block on top %s for time %d before L1 origin %s at time %d",
			l2Head, nextL2Time, l1Origin, l1Origin.Time)
	}

	// Start creating the new block.
	return s.sequencer.StartBuildingBlock(ctx, l1Origin)
}

// completeNewBlock completes a previously started L2 block sequencing job.
func (s *Driver) completeNewBlock(ctx context.Context) error {
	payload, err := s.sequencer.CompleteBuildingBlock(ctx)
	if err != nil {
		s.metrics.RecordSequencingError()
		s.log.Error("Failed to seal block as sequencer", "err", err)
		return err
	}

	// Generate an L2 block ref from the payload.
	newUnsafeL2Head, err := derive.PayloadToBlockRef(payload, &s.config.Genesis)
	if err != nil {
		s.metrics.RecordSequencingError()
		s.log.Error("Sequenced payload cannot be transformed into valid L2 block reference", "err", err)
		return fmt.Errorf("sequenced payload cannot be transformed into valid L2 block reference: %w", err)
	}

	// Update our L2 head block based on the new unsafe block we just generated.
	s.derivation.SetUnsafeHead(newUnsafeL2Head)

	s.log.Info("Sequenced new l2 block", "l2_unsafe", newUnsafeL2Head, "l1_origin", newUnsafeL2Head.L1Origin, "txs", len(payload.Transactions), "time", newUnsafeL2Head.Time)
	s.metrics.CountSequencedTxs(len(payload.Transactions))

	if s.network != nil {
		if err := s.network.PublishL2Payload(ctx, payload); err != nil {
			s.log.Warn("failed to publish newly created block", "id", payload.ID(), "err", err)
			s.metrics.RecordPublishingError()
			// publishing of unsafe data via p2p is optional. Errors are not severe enough to change/halt sequencing but should be logged and metered.
		}
	}
	return nil
}

// the eventLoop responds to L1 changes and internal timers to produce L2 blocks.
func (s *Driver) eventLoop() {
	defer s.wg.Done()
	s.log.Info("State loop started")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// stepReqCh is used to request that the driver attempts to step forward by one L1 block.
	stepReqCh := make(chan struct{}, 1)

	// channel, nil by default (not firing), but used to schedule re-attempts with delay
	var delayedStepReq <-chan time.Time

	// keep track of consecutive failed attempts, to adjust the backoff time accordingly
	bOffStrategy := backoff.Exponential()
	stepAttempts := 0

	// step requests a derivation step to be taken. Won't deadlock if the channel is full.
	step := func() {
		select {
		case stepReqCh <- struct{}{}:
		// Don't deadlock if the channel is already full
		default:
		}
	}

	// reqStep requests a derivation step nicely, with a delay if this is a reattempt, or not at all if we already scheduled a reattempt.
	reqStep := func() {
		if stepAttempts > 0 {
			// if this is not the first attempt, we re-schedule with a backoff, *without blocking other events*
			if delayedStepReq == nil {
				delay := bOffStrategy.Duration(stepAttempts)
				s.log.Debug("scheduling re-attempt with delay", "attempts", stepAttempts, "delay", delay)
				delayedStepReq = time.After(delay)
			} else {
				s.log.Debug("ignoring step request, already scheduled re-attempt after previous failure", "attempts", stepAttempts)
			}
		} else {
			step()
		}
	}

	// We call reqStep right away to finish syncing to the tip of the chain if we're behind.
	// reqStep will also be triggered when the L1 head moves forward or if there was a reorg on the
	// L1 chain that we need to handle.
	reqStep()

	blockTime := time.Duration(s.config.BlockTime) * time.Second

	var sequenceErr error
	var sequenceErrTime time.Time
	sequencerTimer := time.NewTimer(0)
	var sequencerCh <-chan time.Time
	var sequencingPlannedOnto eth.BlockID
	var sequencerSealNext bool
	planSequencerAction := func() {
		delay, seal, onto := s.sequencer.PlanNextSequencerAction(sequenceErr)
		if sequenceErr != nil && time.Since(sequenceErrTime) > delay {
			sequenceErr = nil
		}
		sequencerCh = sequencerTimer.C
		if len(sequencerCh) > 0 { // empty if not already drained before resetting
			<-sequencerCh
		}
		sequencerTimer.Reset(delay)
		sequencingPlannedOnto = onto
		sequencerSealNext = seal
	}

	for {
		// If we are sequencing, update the trigger for the next sequencer action.
		// This may adjust at any time based on fork-choice changes or previous errors.
		if s.driverConfig.SequencerEnabled {
			// update sequencer time if the head changed
			if sequencingPlannedOnto != s.derivation.UnsafeL2Head().ID() {
				planSequencerAction()
			}
		} else {
			sequencerCh = nil
		}

		select {
		case <-sequencerCh:
			s.log.Info("sequencing now!", "seal", sequencerSealNext, "idle_derivation", s.idleDerivation)
			if sequencerSealNext {
				// try to seal the current block task, and allow it to take up to 3 block times.
				// If this fails we will simply start a new block building job.
				ctx, cancel := context.WithTimeout(ctx, 3*blockTime)
				sequenceErr = s.completeNewBlock(ctx)
				cancel()
			} else {
				// Start the block building, don't allow the starting of sequencing to get stuck for more the time of 1 block.
				ctx, cancel := context.WithTimeout(ctx, blockTime)
				sequenceErr = s.startNewL2Block(ctx)
				cancel()
			}
			if sequenceErr != nil {
				s.log.Error("sequencing error", "err", sequenceErr)
				sequenceErrTime = time.Now()
			}
			planSequencerAction() // schedule the next sequencer action to keep the sequencing looping
		case payload := <-s.unsafeL2Payloads:
			s.snapshot("New unsafe payload")
			s.log.Info("Optimistically queueing unsafe L2 execution payload", "id", payload.ID())
			s.derivation.AddUnsafePayload(payload)
			s.metrics.RecordReceivedUnsafePayload(payload)
			reqStep()

		case newL1Head := <-s.l1HeadSig:
			s.l1State.HandleNewL1HeadBlock(newL1Head)
			reqStep() // a new L1 head may mean we have the data to not get an EOF again.
		case newL1Safe := <-s.l1SafeSig:
			s.l1State.HandleNewL1SafeBlock(newL1Safe)
			// no step, justified L1 information does not do anything for L2 derivation or status
		case newL1Finalized := <-s.l1FinalizedSig:
			s.l1State.HandleNewL1FinalizedBlock(newL1Finalized)
			s.derivation.Finalize(newL1Finalized)
			reqStep() // we may be able to mark more L2 data as finalized now
		case <-delayedStepReq:
			delayedStepReq = nil
			step()
		case <-stepReqCh:
			s.metrics.SetDerivationIdle(false)
			s.idleDerivation = false
			s.log.Debug("Derivation process step", "onto_origin", s.derivation.Origin(), "attempts", stepAttempts)
			err := s.derivation.Step(context.Background())
			stepAttempts += 1 // count as attempt by default. We reset to 0 if we are making healthy progress.
			if err == io.EOF {
				s.log.Debug("Derivation process went idle", "progress", s.derivation.Origin())
				s.idleDerivation = true
				stepAttempts = 0
				s.metrics.SetDerivationIdle(true)
				continue
			} else if err != nil && errors.Is(err, derive.ErrReset) {
				// If the pipeline corrupts, e.g. due to a reorg, simply reset it
				s.log.Warn("Derivation pipeline is reset", "err", err)
				s.derivation.Reset()
				s.metrics.RecordPipelineReset()
				continue
			} else if err != nil && errors.Is(err, derive.ErrTemporary) {
				s.log.Warn("Derivation process temporary error", "attempts", stepAttempts, "err", err)
				reqStep()
				continue
			} else if err != nil && errors.Is(err, derive.ErrCritical) {
				s.log.Error("Derivation process critical error", "err", err)
				return
			} else if err != nil && errors.Is(err, derive.NotEnoughData) {
				stepAttempts = 0 // don't do a backoff for this error
				reqStep()
				continue
			} else if err != nil {
				s.log.Error("Derivation process error", "attempts", stepAttempts, "err", err)
				reqStep()
				continue
			} else {
				stepAttempts = 0
				reqStep() // continue with the next step if we can
			}
		case respCh := <-s.stateReq:
			respCh <- struct{}{}
		case respCh := <-s.forceReset:
			s.log.Warn("Derivation pipeline is manually reset")
			s.derivation.Reset()
			s.metrics.RecordPipelineReset()
			close(respCh)
		case <-s.done:
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

// syncStatus returns the current sync status, and should only be called synchronously with
// the driver event loop to avoid retrieval of an inconsistent status.
func (s *Driver) syncStatus() *eth.SyncStatus {
	return &eth.SyncStatus{
		CurrentL1:          s.derivation.Origin(),
		CurrentL1Finalized: s.derivation.FinalizedL1(),
		HeadL1:             s.l1State.L1Head(),
		SafeL1:             s.l1State.L1Safe(),
		FinalizedL1:        s.l1State.L1Finalized(),
		UnsafeL2:           s.derivation.UnsafeL2Head(),
		SafeL2:             s.derivation.SafeL2Head(),
		FinalizedL2:        s.derivation.Finalized(),
	}
}

// SyncStatus blocks the driver event loop and captures the syncing status.
// If the event loop is too busy and the context expires, a context error is returned.
func (s *Driver) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	wait := make(chan struct{})
	select {
	case s.stateReq <- wait:
		resp := s.syncStatus()
		<-wait
		return resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// BlockRefWithStatus blocks the driver event loop and captures the syncing status,
// along with an L2 block reference by number consistent with that same status.
// If the event loop is too busy and the context expires, a context error is returned.
func (s *Driver) BlockRefWithStatus(ctx context.Context, num uint64) (eth.L2BlockRef, *eth.SyncStatus, error) {
	wait := make(chan struct{})
	select {
	case s.stateReq <- wait:
		resp := s.syncStatus()
		ref, err := s.l2.L2BlockRefByNumber(ctx, num)
		<-wait
		return ref, resp, err
	case <-ctx.Done():
		return eth.L2BlockRef{}, nil, ctx.Err()
	}
}

// deferJSONString helps avoid a JSON-encoding performance hit if the snapshot logger does not run
type deferJSONString struct {
	x any
}

func (v deferJSONString) String() string {
	out, _ := json.Marshal(v.x)
	return string(out)
}

func (s *Driver) snapshot(event string) {
	s.snapshotLog.Info("Rollup State Snapshot",
		"event", event,
		"l1Head", deferJSONString{s.l1State.L1Head()},
		"l1Current", deferJSONString{s.derivation.Origin()},
		"l2Head", deferJSONString{s.derivation.UnsafeL2Head()},
		"l2Safe", deferJSONString{s.derivation.SafeL2Head()},
		"l2FinalizedHead", deferJSONString{s.derivation.Finalized()})
}
