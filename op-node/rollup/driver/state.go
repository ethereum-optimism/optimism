package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	gosync "sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/backoff"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum/go-ethereum/log"
)

// Deprecated: use eth.SyncStatus instead.
type SyncStatus = eth.SyncStatus

type state struct {
	// Latest recorded head, safe block and finalized block of the L1 Chain, independent of derivation work
	l1Head      eth.L1BlockRef
	l1Safe      eth.L1BlockRef
	l1Finalized eth.L1BlockRef

	// The derivation pipeline is reset whenever we reorg.
	// The derivation pipeline determines the new l2Safe.
	derivation DerivationPipeline

	// When the derivation pipeline is waiting for new data to do anything
	idleDerivation bool

	// Requests for sync status. Synchronized with event loop to avoid reading an inconsistent sync status.
	syncStatusReq chan chan eth.SyncStatus

	// Upon receiving a channel in this channel, the derivation pipeline is forced to be reset.
	// It tells the caller that the reset occurred by closing the passed in channel.
	forceReset chan chan struct{}

	// Rollup config: rollup chain configuration
	Config *rollup.Config

	// Driver config: verifier and sequencer settings
	DriverConfig *Config

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

	l1      L1Chain
	l2      L2Chain
	output  outputInterface
	network Network // may be nil, network for is optional

	metrics     Metrics
	log         log.Logger
	snapshotLog log.Logger
	done        chan struct{}

	wg gosync.WaitGroup
}

// NewState creates a new driver state. State changes take effect though
// the given output, derivation pipeline and network interfaces.
func NewState(driverCfg *Config, log log.Logger, snapshotLog log.Logger, config *rollup.Config, l1Chain L1Chain, l2Chain L2Chain,
	output outputInterface, derivationPipeline DerivationPipeline, network Network, metrics Metrics) *state {
	return &state{
		derivation:       derivationPipeline,
		idleDerivation:   false,
		syncStatusReq:    make(chan chan eth.SyncStatus, 10),
		forceReset:       make(chan chan struct{}, 10),
		Config:           config,
		DriverConfig:     driverCfg,
		done:             make(chan struct{}),
		log:              log,
		snapshotLog:      snapshotLog,
		l1:               l1Chain,
		l2:               l2Chain,
		output:           output,
		network:          network,
		metrics:          metrics,
		l1HeadSig:        make(chan eth.L1BlockRef, 10),
		l1SafeSig:        make(chan eth.L1BlockRef, 10),
		l1FinalizedSig:   make(chan eth.L1BlockRef, 10),
		unsafeL2Payloads: make(chan *eth.ExecutionPayload, 10),
	}
}

// Start starts up the state loop.
// The loop will have been started iff err is not nil.
func (s *state) Start(_ context.Context) error {
	s.derivation.Reset()

	s.wg.Add(1)
	go s.eventLoop()

	return nil
}

func (s *state) Close() error {
	s.done <- struct{}{}
	s.wg.Wait()
	return nil
}

// OnL1Head signals the driver that the L1 chain changed the "unsafe" block,
// also known as head of the chain, or "latest".
func (s *state) OnL1Head(ctx context.Context, unsafe eth.L1BlockRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.l1HeadSig <- unsafe:
		return nil
	}
}

// OnL1Safe signals the driver that the L1 chain changed the "safe",
// also known as the justified checkpoint (as seen on L1 beacon-chain).
func (s *state) OnL1Safe(ctx context.Context, safe eth.L1BlockRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.l1SafeSig <- safe:
		return nil
	}
}

func (s *state) OnL1Finalized(ctx context.Context, finalized eth.L1BlockRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.l1FinalizedSig <- finalized:
		return nil
	}
}

func (s *state) OnUnsafeL2Payload(ctx context.Context, payload *eth.ExecutionPayload) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.unsafeL2Payloads <- payload:
		return nil
	}
}

func (s *state) handleNewL1HeadBlock(head eth.L1BlockRef) {
	// We don't need to do anything if the head hasn't changed.
	if s.l1Head == (eth.L1BlockRef{}) {
		s.log.Info("Received first L1 head signal", "l1_head", head)
	} else if s.l1Head.Hash == head.Hash {
		s.log.Trace("Received L1 head signal that is the same as the current head", "l1_head", head)
	} else if s.l1Head.Hash == head.ParentHash {
		// We got a new L1 block whose parent hash is the same as the current L1 head. Means we're
		// dealing with a linear extension (new block is the immediate child of the old one).
		s.log.Debug("L1 head moved forward", "l1_head", head)
	} else {
		if s.l1Head.Number >= head.Number {
			s.metrics.RecordL1ReorgDepth(s.l1Head.Number - head.Number)
		}
		// New L1 block is not the same as the current head or a single step linear extension.
		// This could either be a long L1 extension, or a reorg, or we simply missed a head update.
		s.log.Warn("L1 head signal indicates a possible L1 re-org", "old_l1_head", s.l1Head, "new_l1_head_parent", head.ParentHash, "new_l1_head", head)
	}
	s.snapshot("New L1 Head")
	s.metrics.RecordL1Ref("l1_head", head)
	s.l1Head = head
}

func (s *state) handleNewL1SafeBlock(safe eth.L1BlockRef) {
	s.log.Info("New L1 safe block", "l1_safe", safe)
	s.metrics.RecordL1Ref("l1_safe", safe)
	s.l1Safe = safe
}

func (s *state) handleNewL1FinalizedBlock(finalized eth.L1BlockRef) {
	s.log.Info("New L1 finalized block", "l1_finalized", finalized)
	s.metrics.RecordL1Ref("l1_finalized", finalized)
	s.l1Finalized = finalized
	s.derivation.Finalize(finalized.ID())
}

// findL1Origin determines what the next L1 Origin should be.
// The L1 Origin is either the L2 Head's Origin, or the following L1 block
// if the next L2 block's time is greater than or equal to the L2 Head's Origin.
func (s *state) findL1Origin(ctx context.Context) (eth.L1BlockRef, error) {
	l2Head := s.derivation.UnsafeL2Head()
	// If we are at the head block, don't do a lookup.
	if l2Head.L1Origin.Hash == s.l1Head.Hash {
		return s.l1Head, nil
	}

	// Grab a reference to the current L1 origin block.
	currentOrigin, err := s.l1.L1BlockRefByHash(ctx, l2Head.L1Origin.Hash)
	if err != nil {
		return eth.L1BlockRef{}, err
	}

	if currentOrigin.Number+1+s.DriverConfig.SequencerConfDepth > s.l1Head.Number {
		// TODO: we can decide to ignore confirmation depth if we would be forced
		//  to make an empty block (only deposits) by staying on the current origin.
		s.log.Info("sequencing with old origin to preserve conf depth",
			"current", currentOrigin, "current_time", currentOrigin.Time,
			"l1_head", s.l1Head, "l1_head_time", s.l1Head.Time,
			"l2_head", l2Head, "l2_head_time", l2Head.Time,
			"depth", s.DriverConfig.SequencerConfDepth)
		return currentOrigin, nil
	}

	// Attempt to find the next L1 origin block, where the next origin is the immediate child of
	// the current origin block.
	nextOrigin, err := s.l1.L1BlockRefByNumber(ctx, currentOrigin.Number+1)
	if err != nil {
		s.log.Error("Failed to get next origin. Falling back to current origin", "err", err)
		return currentOrigin, nil
	}

	// If the next L2 block time is greater than the next origin block's time, we can choose to
	// start building on top of the next origin. Sequencer implementation has some leeway here and
	// could decide to continue to build on top of the previous origin until the Sequencer runs out
	// of slack. For simplicity, we implement our Sequencer to always start building on the latest
	// L1 block when we can.
	if l2Head.Time+s.Config.BlockTime >= nextOrigin.Time {
		return nextOrigin, nil
	}

	return currentOrigin, nil
}

// createNewL2Block builds a L2 block on top of the L2 Head (unsafe). Used by Sequencer nodes to
// construct new L2 blocks. Verifier nodes will use handleEpoch instead.
func (s *state) createNewL2Block(ctx context.Context) error {
	// Figure out which L1 origin block we're going to be building on top of.
	l1Origin, err := s.findL1Origin(ctx)
	if err != nil {
		s.log.Error("Error finding next L1 Origin", "err", err)
		return err
	}

	// Rollup is configured to not start producing blocks until a specific L1 block has been
	// reached. Don't produce any blocks until we're at that genesis block.
	if l1Origin.Number < s.Config.Genesis.L1.Number {
		s.log.Info("Skipping block production because the next L1 Origin is behind the L1 genesis", "next", l1Origin.ID(), "genesis", s.Config.Genesis.L1)
		return nil
	}

	l2Head := s.derivation.UnsafeL2Head()
	l2Safe := s.derivation.SafeL2Head()
	l2Finalized := s.derivation.Finalized()

	// Should never happen. Sequencer will halt if we get into this situation somehow.
	nextL2Time := l2Head.Time + s.Config.BlockTime
	if nextL2Time < l1Origin.Time {
		s.log.Error("Cannot build L2 block for time before L1 origin",
			"l2Unsafe", l2Head, "nextL2Time", nextL2Time, "l1Origin", l1Origin, "l1OriginTime", l1Origin.Time)
		return fmt.Errorf("cannot build L2 block on top %s for time %d before L1 origin %s at time %d",
			l2Head, nextL2Time, l1Origin, l1Origin.Time)
	}

	// Actually create the new block.
	newUnsafeL2Head, payload, err := s.output.createNewBlock(ctx, l2Head, l2Safe.ID(), l2Finalized.ID(), l1Origin)
	if err != nil {
		s.log.Error("Could not extend chain as sequencer", "err", err, "l2_parent", l2Head, "l1_origin", l1Origin)
		return err
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
func (s *state) eventLoop() {
	defer s.wg.Done()
	s.log.Info("State loop started")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start a ticker to produce L2 blocks at a constant rate. Ticker will only run if we're
	// running in Sequencer mode.
	var l2BlockCreationTickerCh <-chan time.Time
	if s.DriverConfig.SequencerEnabled {
		l2BlockCreationTicker := time.NewTicker(time.Duration(s.Config.BlockTime) * time.Second)
		defer l2BlockCreationTicker.Stop()
		l2BlockCreationTickerCh = l2BlockCreationTicker.C
	}

	// stepReqCh is used to request that the driver attempts to step forward by one L1 block.
	stepReqCh := make(chan struct{}, 1)

	// l2BlockCreationReqCh is used to request that the driver create a new L2 block. Only used if
	// we're running in Sequencer mode, because otherwise we'll be deriving our blocks via the
	// stepping process.
	l2BlockCreationReqCh := make(chan struct{}, 1)

	// reqL2BlockCreation requests that a block be created. Won't deadlock if the channel is full.
	reqL2BlockCreation := func() {
		select {
		case l2BlockCreationReqCh <- struct{}{}:
		// Don't deadlock if the channel is already full
		default:
		}
	}

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

	for {
		select {
		case <-l2BlockCreationTickerCh:
			s.log.Trace("L2 Creation Ticker")
			s.snapshot("L2 Creation Ticker")
			reqL2BlockCreation()

		case <-l2BlockCreationReqCh:
			s.snapshot("L2 Block Creation Request")
			if !s.idleDerivation {
				s.log.Warn("not creating block, node is deriving new l2 data", "head_l1", s.l1Head)
				break
			}
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := s.createNewL2Block(ctx)
			cancel()
			if err != nil {
				s.log.Error("Error creating new L2 block", "err", err)
				s.metrics.RecordSequencingError()
				break // if we fail, we wait for the next block creation trigger.
			}

			// We need to catch up to the next origin as quickly as possible. We can do this by
			// requesting a new block ASAP instead of waiting for the next tick.
			// We don't request a block if the confirmation depth is not met.
			l2Head := s.derivation.UnsafeL2Head()
			if s.l1Head.Number > l2Head.L1Origin.Number+s.DriverConfig.SequencerConfDepth {
				s.log.Trace("Building another L2 block asap to catch up with L1 head", "l2_unsafe", l2Head, "l2_unsafe_l1_origin", l2Head.L1Origin, "l1_head", s.l1Head)
				// But not too quickly to minimize busy-waiting for new blocks
				time.AfterFunc(time.Millisecond*10, reqL2BlockCreation)
			}

		case payload := <-s.unsafeL2Payloads:
			s.snapshot("New unsafe payload")
			s.log.Info("Optimistically queueing unsafe L2 execution payload", "id", payload.ID())
			s.derivation.AddUnsafePayload(payload)
			s.metrics.RecordReceivedUnsafePayload(payload)
			reqStep()

		case newL1Head := <-s.l1HeadSig:
			s.handleNewL1HeadBlock(newL1Head)
			reqStep() // a new L1 head may mean we have the data to not get an EOF again.
		case newL1Safe := <-s.l1SafeSig:
			s.handleNewL1SafeBlock(newL1Safe)
			// no step, justified L1 information does not do anything for L2 derivation or status
		case newL1Finalized := <-s.l1FinalizedSig:
			s.handleNewL1FinalizedBlock(newL1Finalized)
			reqStep() // we may be able to mark more L2 data as finalized now
		case <-delayedStepReq:
			delayedStepReq = nil
			step()
		case <-stepReqCh:
			s.metrics.SetDerivationIdle(false)
			s.idleDerivation = false
			s.log.Debug("Derivation process step", "onto_origin", s.derivation.Progress().Origin, "onto_closed", s.derivation.Progress().Closed, "attempts", stepAttempts)
			stepCtx, cancel := context.WithTimeout(ctx, time.Second*10) // TODO pick a timeout for executing a single step
			err := s.derivation.Step(stepCtx)
			cancel()
			stepAttempts += 1 // count as attempt by default. We reset to 0 if we are making healthy progress.
			if err == io.EOF {
				s.log.Debug("Derivation process went idle", "progress", s.derivation.Progress().Origin)
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
		case respCh := <-s.syncStatusReq:
			respCh <- eth.SyncStatus{
				CurrentL1:   s.derivation.Progress().Origin,
				HeadL1:      s.l1Head,
				SafeL1:      s.l1Safe,
				FinalizedL1: s.l1Finalized,
				UnsafeL2:    s.derivation.UnsafeL2Head(),
				SafeL2:      s.derivation.SafeL2Head(),
				FinalizedL2: s.derivation.Finalized(),
			}
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
func (s *state) ResetDerivationPipeline(ctx context.Context) error {
	respCh := make(chan struct{})
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

func (s *state) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	respCh := make(chan eth.SyncStatus)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case s.syncStatusReq <- respCh:
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case resp := <-respCh:
			return &resp, nil
		}
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

func (s *state) snapshot(event string) {
	s.snapshotLog.Info("Rollup State Snapshot",
		"event", event,
		"l1Head", deferJSONString{s.l1Head},
		"l1Current", deferJSONString{s.derivation.Progress().Origin},
		"l2Head", deferJSONString{s.derivation.UnsafeL2Head()},
		"l2Safe", deferJSONString{s.derivation.SafeL2Head()},
		"l2FinalizedHead", deferJSONString{s.derivation.Finalized()})
}
