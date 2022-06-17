package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	gosync "sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum/go-ethereum/log"
)

type state struct {
	// Chain State
	l1Head      eth.L1BlockRef // Latest recorded head of the L1 Chain, independent of derivation work
	l2Head      eth.L2BlockRef // L2 Unsafe Head
	l2SafeHead  eth.L2BlockRef // L2 Safe Head - this is the head of the L2 chain as derived from L1
	l2Finalized eth.L2BlockRef // L2 Block that will never be reversed

	// The derivation pipeline is reset whenever we reorg.
	// The derivation pipeline determines the new l2SafeHead.
	derivation DerivationPipeline

	emitter *derive.ChannelEmitter

	// Rollup config
	Config    *rollup.Config
	sequencer bool

	// Connections (in/out)
	l1Heads          chan eth.L1BlockRef
	unsafeL2Payloads chan *eth.ExecutionPayload
	l1               L1Chain
	l2               L2Chain
	output           outputInterface
	network          Network // may be nil, network for is optional

	log         log.Logger
	snapshotLog log.Logger
	done        chan struct{}

	wg gosync.WaitGroup
}

// NewState creates a new driver state. State changes take effect though
// the given output, derivation pipeline and network interfaces.
func NewState(log log.Logger, snapshotLog log.Logger, config *rollup.Config, l1Chain L1Chain, l2Chain L2Chain,
	output outputInterface, derivationPipeline DerivationPipeline, network Network, sequencer bool) *state {
	return &state{
		derivation:       derivationPipeline,
		emitter:          derive.NewChannelEmitter(log, config, l2Chain),
		Config:           config,
		done:             make(chan struct{}),
		log:              log,
		snapshotLog:      snapshotLog,
		l1:               l1Chain,
		l2:               l2Chain,
		output:           output,
		network:          network,
		sequencer:        sequencer,
		l1Heads:          make(chan eth.L1BlockRef, 10),
		unsafeL2Payloads: make(chan *eth.ExecutionPayload, 10),
	}
}

// Start starts up the state loop. The context is only for initialization.
// The loop will have been started iff err is not nil.
func (s *state) Start(ctx context.Context) error {
	l1Head, err := s.l1.L1HeadBlockRef(ctx)
	if err != nil {
		return err
	}
	s.l1Head = l1Head

	if err := s.resetDerivation(ctx); err != nil {
		return fmt.Errorf("failed to reset derivation pipeline to starting point")
	}

	s.wg.Add(1)
	go s.eventLoop()

	return nil
}

func (s *state) Emitter() *derive.ChannelEmitter {
	return s.emitter
}

func (s *state) Close() error {
	s.done <- struct{}{}
	s.wg.Wait()
	return nil
}

func (s *state) OnL1Head(ctx context.Context, head eth.L1BlockRef) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.l1Heads <- head:
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

func (s *state) handleNewL1Block(newL1Head eth.L1BlockRef) {
	// We don't need to do anything if the head hasn't changed.
	if s.l1Head.Hash == newL1Head.Hash {
		s.log.Trace("Received L1 head signal that is the same as the current head", "l1Head", newL1Head)
	} else if s.l1Head.Hash == newL1Head.ParentHash {
		// We got a new L1 block whose parent hash is the same as the current L1 head. Means we're
		// dealing with a linear extension (new block is the immediate child of the old one).
		s.log.Debug("L1 head moved forward", "l1Head", newL1Head)
	} else {
		// New L1 block is not the same as the current head or a single step linear extension.
		// This could either be a long L1 extension, or a reorg. Both can be handled the same way.
		s.log.Warn("L1 Head signal indicates an L1 re-org", "old_l1_head", s.l1Head, "new_l1_head_parent", newL1Head.ParentHash, "new_l1_head", newL1Head)
	}
	s.l1Head = newL1Head
	s.emitter.SetL1Time(newL1Head.Time)
}

func (s *state) resetDerivation(ctx context.Context) error {
	var unsafeL2Head, safeL2Head eth.L2BlockRef
	// Check that we are past the genesis
	if s.l1Head.Number > s.Config.Genesis.L1.Number {
		// Upon resetting, make sure we are on the correct chain.
		// The engine might be way behind/ahead of what we previously thought it was at.
		var err error
		unsafeL2Head, safeL2Head, err = sync.FindL2Heads(ctx, s.Config.SeqWindowSize, s.l1, s.l2, &s.Config.Genesis)
		if err != nil {
			s.log.Error("Could not get new unsafe L2 head when trying to handle a re-org", "err", err)
			return err
		}
	} else {
		// pre-genesis (i.e. our L1 view is behind, even though we know the L1 block we anchor the rollup at)
		// we just reset the derivation pipeline to the known L2 starting point
		unsafeL2Head = eth.L2BlockRef{
			Hash:           s.Config.Genesis.L2.Hash,
			Number:         s.Config.Genesis.L2.Number,
			Time:           s.Config.Genesis.L2Time,
			L1Origin:       s.Config.Genesis.L1,
			SequenceNumber: 0,
		}
		safeL2Head = unsafeL2Head
	}

	if err := s.derivation.Reset(ctx, safeL2Head, unsafeL2Head); err != nil {
		s.log.Error("Failed to reset derivation pipeline after reorg was detected", "err", err)
		return err
	}
	return nil
}

// findL1Origin determines what the next L1 Origin should be.
// The L1 Origin is either the L2 Head's Origin, or the following L1 block
// if the next L2 block's time is greater than or equal to the L2 Head's Origin.
func (s *state) findL1Origin(ctx context.Context) (eth.L1BlockRef, error) {
	// If we are at the head block, don't do a lookup.
	if s.l2Head.L1Origin.Hash == s.l1Head.Hash {
		return s.l1Head, nil
	}

	// Grab a reference to the current L1 origin block.
	currentOrigin, err := s.l1.L1BlockRefByHash(ctx, s.l2Head.L1Origin.Hash)
	if err != nil {
		return eth.L1BlockRef{}, err
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
	// TODO: Can add confirmation depth here if we want.
	if s.l2Head.Time+s.Config.BlockTime >= nextOrigin.Time {
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

	// Should never happen. Sequencer will halt if we get into this situation somehow.
	nextL2Time := s.l2Head.Time + s.Config.BlockTime
	if nextL2Time < l1Origin.Time {
		s.log.Error("Cannot build L2 block for time before L1 origin",
			"l2Head", s.l2Head, "nextL2Time", nextL2Time, "l1Origin", l1Origin, "l1OriginTime", l1Origin.Time)
		return fmt.Errorf("cannot build L2 block on top %s for time %d before L1 origin %s at time %d",
			s.l2Head, nextL2Time, l1Origin, l1Origin.Time)
	}

	// Actually create the new block.
	newUnsafeL2Head, payload, err := s.output.createNewBlock(ctx, s.l2Head, s.l2SafeHead.ID(), s.l2Finalized.ID(), l1Origin)
	if err != nil {
		s.log.Error("Could not extend chain as sequencer", "err", err, "l2UnsafeHead", s.l2Head, "l1Origin", l1Origin)
		return err
	}

	// Update our L2 head block based on the new unsafe block we just generated.
	s.l2Head = newUnsafeL2Head
	s.emitter.SetL2UnsafeHead(s.l2Head)

	s.log.Info("Sequenced new l2 block", "l2Head", s.l2Head, "l1Origin", s.l2Head.L1Origin, "txs", len(payload.Transactions), "time", s.l2Head.Time)

	if s.network != nil {
		if err := s.network.PublishL2Payload(ctx, payload); err != nil {
			s.log.Warn("failed to publish newly created block", "id", payload.ID(), "err", err)
			return err
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
	if s.sequencer {
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

	// reqStep requests that a driver stpe be taken. Won't deadlock if the channel is full.
	// TODO: Rename step request
	reqStep := func() {
		select {
		case stepReqCh <- struct{}{}:
		// Don't deadlock if the channel is already full
		default:
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
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := s.createNewL2Block(ctx)
			cancel()
			if err != nil {
				s.log.Error("Error creating new L2 block", "err", err)
			}

			// We need to catch up to the next origin as quickly as possible. We can do this by
			// requesting a new block ASAP instead of waiting for the next tick.
			// TODO: If we want to consider confirmations, need to consider here too.
			if s.l1Head.Number > s.l2Head.L1Origin.Number {
				s.log.Trace("Asking for a second L2 block asap", "l2Head", s.l2Head)
				// But not too quickly to minimize busy-waiting for new blocks
				time.AfterFunc(time.Millisecond*10, reqL2BlockCreation)
			}

		case payload := <-s.unsafeL2Payloads:
			s.snapshot("New unsafe payload")
			s.log.Info("Optimistically queueing unsafe L2 execution payload", "id", payload.ID())
			s.derivation.AddUnsafePayload(payload)
			reqStep()

		case newL1Head := <-s.l1Heads:
			s.snapshot("New L1 Head")
			s.handleNewL1Block(newL1Head)
			reqStep() // a new L1 head may mean we have the data to not get an EOF again.
		case <-stepReqCh:
			s.log.Trace("Derivation process step", "onto", s.derivation.CurrentL1())
			stepCtx, cancel := context.WithTimeout(ctx, time.Second*10) // TODO pick a timeout for executing a single step
			err := s.derivation.Step(stepCtx)
			cancel()
			if err == io.EOF {
				continue
			} else if err != nil {
				s.log.Warn("derivation pipeline critically failed, resetting it", "err", err)
				// If the pipeline corrupts, simply reset it
				if err := s.resetDerivation(ctx); err != nil {
					s.log.Error("failed to reset derivation pipeline after failing step", "err", err)
				}
			} else {
				finalized, safe, unsafe := s.derivation.Finalized(), s.derivation.SafeL2Head(), s.derivation.UnsafeL2Head()
				// log sync progress when it changes
				if s.l2Finalized != finalized || s.l2SafeHead != safe || s.l2Head != unsafe {
					s.log.Info("sync progress", "finalized", finalized, "safe", safe, "unsafe", unsafe)
				}
				// update the heads
				s.l2Finalized = finalized
				s.l2SafeHead = safe
				s.l2Head = unsafe
				s.emitter.SetL2SafeHead(safe)
				s.emitter.SetL2UnsafeHead(unsafe)
				reqStep() // continue with the next step if we can
			}
		case <-s.done:
			return
		}
	}
}

func (s *state) snapshot(event string) {
	l1HeadJSON, _ := json.Marshal(s.l1Head)
	l1CurrentJSON, _ := json.Marshal(s.l2Head)
	l2HeadJSON, _ := json.Marshal(s.l2Head)
	l2SafeHeadJSON, _ := json.Marshal(s.l2SafeHead)
	l2FinalizedHeadJSON, _ := json.Marshal(s.l2Finalized)

	s.snapshotLog.Info("Rollup State Snapshot",
		"event", event,
		"l1Head", string(l1HeadJSON),
		"l1Current", string(l1CurrentJSON),
		"l2Head", string(l2HeadJSON),
		"l2SafeHead", string(l2SafeHeadJSON),
		"l2FinalizedHead", string(l2FinalizedHeadJSON))
}
