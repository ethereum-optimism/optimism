package driver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	gosync "sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/log"
)

// SyncStatus is a snapshot of the driver
type SyncStatus struct {
	// CurrentL1 is the block that the derivation process is currently at,
	// this may not be fully derived into L2 data yet.
	// If the node is synced, this matches the HeadL1, minus the verifier confirmation distance.
	CurrentL1 eth.L1BlockRef `json:"current_l1"`
	// HeadL1 is the perceived head of the L1 chain, no confirmation distance.
	// The head is not guaranteed to build on the other L1 sync status fields,
	// as the node may be in progress of resetting to adapt to a L1 reorg.
	HeadL1 eth.L1BlockRef `json:"head_l1"`
	// UnsafeL2 is the absolute tip of the L2 chain,
	// pointing to block data that has not been submitted to L1 yet.
	// The sequencer is building this, and verifiers may also be ahead of the
	// SafeL2 block if they sync blocks via p2p or other offchain sources.
	UnsafeL2 eth.L2BlockRef `json:"unsafe_l2"`
	// SafeL2 points to the L2 block that was derived from the L1 chain.
	// This point may still reorg if the L1 chain reorgs.
	SafeL2 eth.L2BlockRef `json:"safe_l2"`
	// FinalizedL2 points to the L2 block that was derived fully from
	// finalized L1 information, thus irreversible.
	FinalizedL2 eth.L2BlockRef `json:"finalized_l2"`
}

type state struct {
	// Chain State
	l1Head      eth.L1BlockRef // Latest recorded head of the L1 Chain, independent of derivation work
	l2Head      eth.L2BlockRef // L2 Unsafe Head
	l2SafeHead  eth.L2BlockRef // L2 Safe Head - this is the head of the L2 chain as derived from L1
	l2Finalized eth.L2BlockRef // L2 Block that will never be reversed

	// The derivation pipeline is reset whenever we reorg.
	// The derivation pipeline determines the new l2SafeHead.
	derivation DerivationPipeline

	// When the derivation pipeline is waiting for new data to do anything
	idleDerivation bool

	// Requests for sync status. Synchronized with event loop to avoid reading an inconsistent sync status.
	syncStatusReq chan chan SyncStatus

	// Rollup config: rollup chain configuration
	Config *rollup.Config

	// Driver config: verifier and sequencer settings
	DriverConfig *Config

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
func NewState(driverCfg *Config, log log.Logger, snapshotLog log.Logger, config *rollup.Config, l1Chain L1Chain, l2Chain L2Chain,
	output outputInterface, derivationPipeline DerivationPipeline, network Network) *state {
	return &state{
		derivation:       derivationPipeline,
		idleDerivation:   false,
		syncStatusReq:    make(chan chan SyncStatus, 10),
		Config:           config,
		DriverConfig:     driverCfg,
		done:             make(chan struct{}),
		log:              log,
		snapshotLog:      snapshotLog,
		l1:               l1Chain,
		l2:               l2Chain,
		output:           output,
		network:          network,
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
	s.l2Head, _ = s.l2.L2BlockRefByNumber(ctx, nil)

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

	if currentOrigin.Number+1+s.DriverConfig.SequencerConfDepth > s.l1Head.Number {
		// TODO: we can decide to ignore confirmation depth if we would be forced
		//  to make an empty block (only deposits) by staying on the current origin.
		s.log.Info("sequencing with old origin to preserve conf depth",
			"current", currentOrigin, "current_time", currentOrigin.Time,
			"l1_head", s.l1Head, "l1_head_time", s.l1Head.Time,
			"l2_head", s.l2Head, "l2_head_time", s.l2Head.Time,
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
	s.derivation.SetUnsafeHead(newUnsafeL2Head)
	s.l2Head = newUnsafeL2Head

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

	// reqStep requests a derivation step to be taken. Won't deadlock if the channel is full.
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
			if !s.idleDerivation {
				s.log.Warn("not creating block, node is deriving new l2 data", "head_l1", s.l1Head)
				break
			}
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := s.createNewL2Block(ctx)
			cancel()
			if err != nil {
				s.log.Error("Error creating new L2 block", "err", err)
			}

			// We need to catch up to the next origin as quickly as possible. We can do this by
			// requesting a new block ASAP instead of waiting for the next tick.
			// We don't request a block if the confirmation depth is not met.
			if s.l1Head.Number > s.l2Head.L1Origin.Number+s.DriverConfig.SequencerConfDepth {
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
			s.log.Info("new l1 Head")
			s.snapshot("New L1 Head")
			s.handleNewL1Block(newL1Head)
			reqStep() // a new L1 head may mean we have the data to not get an EOF again.
		case <-stepReqCh:
			s.idleDerivation = false
			s.log.Debug("Derivation process step", "onto_origin", s.derivation.Progress().Origin, "onto_closed", s.derivation.Progress().Closed)
			stepCtx, cancel := context.WithTimeout(ctx, time.Second*10) // TODO pick a timeout for executing a single step
			err := s.derivation.Step(stepCtx)
			cancel()
			if err == io.EOF {
				s.log.Debug("Derivation process went idle", "progress", s.derivation.Progress().Origin)
				s.idleDerivation = true
				continue
			} else if err != nil {
				// If the pipeline corrupts, e.g. due to a reorg, simply reset it
				s.log.Warn("Derivation pipeline is reset", "err", err)
				s.derivation.Reset()
			} else {
				finalized, safe, unsafe := s.derivation.Finalized(), s.derivation.SafeL2Head(), s.derivation.UnsafeL2Head()
				// log sync progress when it changes
				if s.l2Finalized != finalized || s.l2SafeHead != safe || s.l2Head != unsafe {
					s.log.Info("Sync progress", "finalized", finalized, "safe", safe, "unsafe", unsafe)
				}
				// update the heads
				s.l2Finalized = finalized
				s.l2SafeHead = safe
				s.l2Head = unsafe
				reqStep() // continue with the next step if we can
			}
		case respCh := <-s.syncStatusReq:
			respCh <- SyncStatus{
				CurrentL1:   s.derivation.Progress().Origin,
				HeadL1:      s.l1Head,
				UnsafeL2:    s.l2Head,
				SafeL2:      s.l2SafeHead,
				FinalizedL2: s.l2Finalized,
			}
		case <-s.done:
			return
		}
	}
}

func (s *state) SyncStatus(ctx context.Context) (*SyncStatus, error) {
	respCh := make(chan SyncStatus)
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
		"l2Head", deferJSONString{s.l2Head},
		"l2SafeHead", deferJSONString{s.l2SafeHead},
		"l2FinalizedHead", deferJSONString{s.l2Finalized})
}
