package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	gosync "sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

type Driver struct {
	l1State L1StateIface

	// The derivation pipeline is reset whenever we reorg.
	// The derivation pipeline determines the new l2Safe.
	derivation DerivationPipeline

	// Requests to block the event loop for synchronous execution to avoid reading an inconsistent state
	stateReq chan chan struct{}

	// Upon receiving a channel in this channel, the derivation pipeline is forced to be reset.
	// It tells the caller that the reset occurred by closing the passed in channel.
	forceReset chan chan struct{}

	// Upon receiving a hash in this channel, the sequencer is started at the given hash.
	// It tells the caller that the sequencer started by closing the passed in channel (or returning an error).
	startSequencer chan hashAndErrorChannel

	// Upon receiving a channel in this channel, the sequencer is stopped.
	// It tells the caller that the sequencer stopped by returning the latest sequenced L2 block hash.
	stopSequencer chan chan hashAndError

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

	// Interface to signal the L2 block range to sync.
	altSync AltSync

	// L2 Signals:

	unsafeL2Payloads chan *eth.ExecutionPayload

	l1        L1Chain
	l2        L2Chain
	sequencer SequencerIface
	network   Network // may be nil, network for is optional

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

	sequencerTimer := time.NewTimer(0)
	var sequencerCh <-chan time.Time
	planSequencerAction := func() {
		delay := s.sequencer.PlanNextSequencerAction()
		sequencerCh = sequencerTimer.C
		if len(sequencerCh) > 0 { // empty if not already drained before resetting
			<-sequencerCh
		}
		sequencerTimer.Reset(delay)
	}

	// Create a ticker to check if there is a gap in the engine queue. Whenever
	// there is, we send requests to sync source to retrieve the missing payloads.
	syncCheckInterval := time.Duration(s.config.BlockTime) * time.Second * 2
	altSyncTicker := time.NewTicker(syncCheckInterval)
	defer altSyncTicker.Stop()
	lastUnsafeL2 := s.derivation.UnsafeL2Head()

	for {
		// If we are sequencing, and the L1 state is ready, update the trigger for the next sequencer action.
		// This may adjust at any time based on fork-choice changes or previous errors.
		// And avoid sequencing if the derivation pipeline indicates the engine is not ready.
		if s.driverConfig.SequencerEnabled && !s.driverConfig.SequencerStopped &&
			s.l1State.L1Head() != (eth.L1BlockRef{}) && s.derivation.EngineReady() {
			// update sequencer time if the head changed
			if s.sequencer.BuildingOnto().ID() != s.derivation.UnsafeL2Head().ID() {
				planSequencerAction()
			}
		} else {
			sequencerCh = nil
		}

		// If the engine is not ready, or if the L2 head is actively changing, then reset the alt-sync:
		// there is no need to request L2 blocks when we are syncing already.
		if head := s.derivation.UnsafeL2Head(); head != lastUnsafeL2 || !s.derivation.EngineReady() {
			lastUnsafeL2 = head
			altSyncTicker.Reset(syncCheckInterval)
		}

		select {
		case <-sequencerCh:
			payload, err := s.sequencer.RunNextSequencerAction(ctx)
			if err != nil {
				s.log.Error("Sequencer critical error", "err", err)
				return
			}
			if s.network != nil && payload != nil {
				// Publishing of unsafe data via p2p is optional.
				// Errors are not severe enough to change/halt sequencing but should be logged and metered.
				if err := s.network.PublishL2Payload(ctx, payload); err != nil {
					s.log.Warn("failed to publish newly created block", "id", payload.ID(), "err", err)
					s.metrics.RecordPublishingError()
				}
			}
			planSequencerAction() // schedule the next sequencer action to keep the sequencing looping
		case <-altSyncTicker.C:
			// Check if there is a gap in the current unsafe payload queue.
			ctx, cancel := context.WithTimeout(ctx, time.Second*2)
			err := s.checkForGapInUnsafeQueue(ctx)
			cancel()
			if err != nil {
				s.log.Warn("failed to check for unsafe L2 blocks to sync", "err", err)
			}
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
			s.log.Debug("Derivation process step", "onto_origin", s.derivation.Origin(), "attempts", stepAttempts)
			err := s.derivation.Step(context.Background())
			stepAttempts += 1 // count as attempt by default. We reset to 0 if we are making healthy progress.
			if err == io.EOF {
				s.log.Debug("Derivation process went idle", "progress", s.derivation.Origin())
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
		case resp := <-s.startSequencer:
			unsafeHead := s.derivation.UnsafeL2Head().Hash
			if !s.driverConfig.SequencerStopped {
				resp.err <- errors.New("sequencer already running")
			} else if !bytes.Equal(unsafeHead[:], resp.hash[:]) {
				resp.err <- fmt.Errorf("block hash does not match: head %s, received %s", unsafeHead.String(), resp.hash.String())
			} else {
				s.log.Info("Sequencer has been started")
				s.driverConfig.SequencerStopped = false
				close(resp.err)
				planSequencerAction() // resume sequencing
			}
		case respCh := <-s.stopSequencer:
			if s.driverConfig.SequencerStopped {
				respCh <- hashAndError{err: errors.New("sequencer not running")}
			} else {
				s.log.Warn("Sequencer has been stopped")
				s.driverConfig.SequencerStopped = true
				respCh <- hashAndError{hash: s.derivation.UnsafeL2Head().Hash}
			}
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

func (s *Driver) StartSequencer(ctx context.Context, blockHash common.Hash) error {
	if !s.driverConfig.SequencerEnabled {
		return errors.New("sequencer is not enabled")
	}
	h := hashAndErrorChannel{
		hash: blockHash,
		err:  make(chan error, 1),
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.startSequencer <- h:
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-h.err:
			return e
		}
	}
}

func (s *Driver) StopSequencer(ctx context.Context) (common.Hash, error) {
	if !s.driverConfig.SequencerEnabled {
		return common.Hash{}, errors.New("sequencer is not enabled")
	}
	respCh := make(chan hashAndError, 1)
	select {
	case <-ctx.Done():
		return common.Hash{}, ctx.Err()
	case s.stopSequencer <- respCh:
		select {
		case <-ctx.Done():
			return common.Hash{}, ctx.Err()
		case he := <-respCh:
			return he.hash, he.err
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

type hashAndError struct {
	hash common.Hash
	err  error
}

type hashAndErrorChannel struct {
	hash common.Hash
	err  chan error
}

// checkForGapInUnsafeQueue checks if there is a gap in the unsafe queue and attempts to retrieve the missing payloads from an alt-sync method.
// WARNING: This is only an outgoing signal, the blocks are not guaranteed to be retrieved.
// Results are received through OnUnsafeL2Payload.
func (s *Driver) checkForGapInUnsafeQueue(ctx context.Context) error {
	// subtract genesis time from wall clock to get the time elapsed since genesis, and then divide that
	// difference by the block time to get the expected L2 block number at the current time. If the
	// unsafe head does not have this block number, then there is a gap in the queue.
	wallClock := uint64(time.Now().Unix())
	genesisTimestamp := s.config.Genesis.L2Time
	if wallClock < genesisTimestamp {
		s.log.Debug("nothing to sync, did not reach genesis L2 time yet", "genesis", genesisTimestamp)
		return nil
	}
	wallClockGenesisDiff := wallClock - genesisTimestamp
	// Note: round down, we should not request blocks into the future.
	blocksSinceGenesis := wallClockGenesisDiff / s.config.BlockTime
	expectedL2Block := s.config.Genesis.L2.Number + blocksSinceGenesis

	start, end := s.derivation.GetUnsafeQueueGap(expectedL2Block)
	// Check if there is a gap between the unsafe head and the expected L2 block number at the current time.
	if end > start {
		s.log.Debug("requesting missing unsafe L2 block range", "start", start, "end", end, "size", end-start)
		return s.altSync.RequestL2Range(ctx, start, end)
	}
	return nil
}
