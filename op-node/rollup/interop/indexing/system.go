package indexing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	gethrpc "github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// indexingEventStream abstracts the event stream functionality for testing
type indexingEventStream interface {
	Send(event *supervisortypes.IndexingEvent)
	Serve() (*supervisortypes.IndexingEvent, error)
	Subscribe(ctx context.Context) (*gethrpc.Subscription, error)
}

type L2Source interface {
	L2BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	BlockRefByHash(ctx context.Context, hash common.Hash) (eth.BlockRef, error)
	PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayloadEnvelope, error)
	BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error)
	FetchReceipts(ctx context.Context, blockHash common.Hash) (eth.BlockInfo, types.Receipts, error)
	OutputV0AtBlock(ctx context.Context, blockHash common.Hash) (*eth.OutputV0, error)
}

type L1Source interface {
	L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error)
	L1BlockRefByNumber(ctx context.Context, num uint64) (eth.L1BlockRef, error)
}
type EngineController interface {
	ForceReset(ctx context.Context, localUnsafe, crossUnsafe, localSafe, crossSafe, finalized eth.L2BlockRef)
	PromoteSafe(ctx context.Context, ref eth.L2BlockRef, source eth.L1BlockRef)
	PromoteFinalized(ctx context.Context, ref eth.L2BlockRef)
}

// IndexingMode makes the op-node managed by an op-supervisor,
// by serving sync work and updating the canonical chain based on instructions.
type IndexingMode struct {
	log log.Logger

	emitter event.Emitter

	l1 L1Source
	l2 L2Source

	events indexingEventStream

	// outgoing event timestamp trackers
	lastReset         eventTimestamp[struct{}]
	lastUnsafe        eventTimestamp[eth.BlockID]
	lastSafe          eventTimestamp[eth.BlockID]
	lastL1Traversal   eventTimestamp[eth.BlockID]
	lastExhaustedL1   eventTimestamp[eth.BlockID]
	lastReplacedBlock eventTimestamp[eth.BlockID]

	ctx    context.Context
	cancel context.CancelFunc

	cfg *rollup.Config

	srv       *rpc.Server
	jwtSecret eth.Bytes32

	engineController EngineController
}

func NewIndexingMode(log log.Logger, cfg *rollup.Config, addr string, port int, jwtSecret eth.Bytes32, l1 L1Source, l2 L2Source, m opmetrics.RPCMetricer) *IndexingMode {
	log = log.With("mode", "indexing", "chainId", cfg.L2ChainID)
	ctx, cancel := context.WithCancel(context.Background())
	out := &IndexingMode{
		log:       log,
		cfg:       cfg,
		l1:        l1,
		l2:        l2,
		jwtSecret: jwtSecret,
		events:    rpc.NewStream[supervisortypes.IndexingEvent](log, 100),

		lastReset:         newEventTimestamp[struct{}](100 * time.Millisecond),
		lastUnsafe:        newEventTimestamp[eth.BlockID](100 * time.Millisecond),
		lastSafe:          newEventTimestamp[eth.BlockID](100 * time.Millisecond),
		lastL1Traversal:   newEventTimestamp[eth.BlockID](500 * time.Millisecond),
		lastExhaustedL1:   newEventTimestamp[eth.BlockID](500 * time.Millisecond),
		lastReplacedBlock: newEventTimestamp[eth.BlockID](100 * time.Millisecond),

		ctx:    ctx,
		cancel: cancel,
	}

	out.srv = rpc.NewServer(addr, port, "v0.0.0",
		rpc.WithWebsocketEnabled(),
		rpc.WithLogger(log),
		rpc.WithJWTSecret(jwtSecret[:]),
		rpc.WithRPCRecorder(m.NewRecorder("interop_indexing")),
	)
	out.srv.AddAPI(gethrpc.API{
		Namespace:     "interop",
		Service:       &InteropAPI{backend: out},
		Authenticated: true,
	})
	return out
}

func (m *IndexingMode) SetEngineController(engineController EngineController) {
	m.engineController = engineController
}

// TestDisableEventDeduplication is a test-only function that disables event deduplication.
// It is necessary to make action tests work.
func (m *IndexingMode) TestDisableEventDeduplication() {
	m.lastReset.ttl = 0
	m.lastUnsafe.ttl = 0
	m.lastSafe.ttl = 0
	m.lastL1Traversal.ttl = 0
	m.lastExhaustedL1.ttl = 0
	m.lastReplacedBlock.ttl = 0
}

func (m *IndexingMode) Start(ctx context.Context) error {
	if m.emitter == nil {
		return errors.New("must have emitter before starting")
	}
	if err := m.srv.Start(); err != nil {
		return fmt.Errorf("failed to start interop RPC server: %w", err)
	}
	m.log.Info("Started interop RPC", "endpoint", m.WSEndpoint())
	return nil
}

func (m *IndexingMode) WSEndpoint() string {
	return fmt.Sprintf("ws://%s", m.srv.Endpoint())
}

func (m *IndexingMode) WSPort() (int, error) {
	return m.srv.Port()
}

func (m *IndexingMode) JWTSecret() eth.Bytes32 {
	return m.jwtSecret
}

func (m *IndexingMode) Stop(ctx context.Context) error {
	// stop RPC server
	if err := m.srv.Stop(); err != nil {
		return fmt.Errorf("failed to stop interop sub-system RPC server: %w", err)
	}

	m.cancel()

	m.log.Info("Interop sub-system stopped")
	return nil
}

func (m *IndexingMode) AttachEmitter(em event.Emitter) {
	m.emitter = em
}

// Outgoing events to supervisor
func (m *IndexingMode) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case rollup.ResetEvent:
		logger := m.log.New("err", x.Err)
		logger.Warn("Sending reset request to supervisor")
		if !m.lastReset.Update(struct{}{}) {
			logger.Warn("Skipped sending duplicate reset request")
			return true
		}
		msg := x.Err.Error()
		m.events.Send(&supervisortypes.IndexingEvent{Reset: &msg})

	case engine.UnsafeUpdateEvent:
		logger := m.log.New("unsafe", x.Ref)
		if !m.cfg.IsInterop(x.Ref.Time) {
			logger.Debug("Ignoring non-Interop local unsafe update")
			return false
		} else if !m.lastUnsafe.Update(x.Ref.ID()) {
			logger.Warn("Skipped sending duplicate local unsafe update event")
			return true
		}
		ref := x.Ref.BlockRef()
		m.events.Send(&supervisortypes.IndexingEvent{UnsafeBlock: &ref})

	case engine.LocalSafeUpdateEvent:
		logger := m.log.New("derivedFrom", x.Source, "derived", x.Ref)
		if !m.cfg.IsInterop(x.Ref.Time) {
			logger.Debug("Ignoring non-Interop local safe update")
			return false
		} else if !m.lastSafe.Update(x.Ref.ID()) {
			logger.Warn("Skipped sending duplicate derivation update (new local safe)")
			return true
		}
		logger.Info("Sending derivation update to supervisor (new local safe)")
		m.events.Send(&supervisortypes.IndexingEvent{
			DerivationUpdate: &supervisortypes.DerivedBlockRefPair{
				Source:  x.Source,
				Derived: x.Ref.BlockRef(),
			},
		})

	case derive.DeriverL1StatusEvent:
		logger := m.log.New("derivedFrom", x.Origin, "derived", x.LastL2)
		if !m.cfg.IsInterop(x.LastL2.Time) {
			logger.Debug("Ignoring non-Interop L1 traversal")
			return false
		} else if !m.lastL1Traversal.Update(x.Origin.ID()) {
			logger.Warn("Skipped sending duplicate derivation update (L1 traversal)")
			return true
		}
		logger.Info("Sending derivation update to supervisor (L1 traversal)")
		m.events.Send(&supervisortypes.IndexingEvent{
			DerivationUpdate: &supervisortypes.DerivedBlockRefPair{
				Source:  x.Origin,
				Derived: x.LastL2.BlockRef(),
			},
			DerivationOriginUpdate: &x.Origin,
		})

	case derive.ExhaustedL1Event:
		logger := m.log.New("derivedFrom", x.L1Ref, "derived", x.LastL2)
		logger.Info("Exhausted L1 data")
		if !m.lastExhaustedL1.Update(x.L1Ref.ID()) {
			logger.Warn("Skipped sending duplicate exhausted L1 event", "derivedFrom", x.L1Ref, "derived", x.LastL2)
			return true
		}
		m.events.Send(&supervisortypes.IndexingEvent{
			ExhaustL1: &supervisortypes.DerivedBlockRefPair{
				Source:  x.L1Ref,
				Derived: x.LastL2.BlockRef(),
			},
		})

	case engine.InteropReplacedBlockEvent:
		logger := m.log.New("replacement", x.Ref)
		logger.Info("Replaced block")
		if !m.lastReplacedBlock.Update(x.Ref.ID()) {
			logger.Warn("Skipped sending duplicate replaced block event", "replacement", x.Ref)
			return true
		}
		out, err := DecodeInvalidatedBlockTxFromReplacement(x.Envelope.ExecutionPayload.Transactions)
		if err != nil {
			logger.Error("Failed to parse replacement block", "err", err)
			return true
		}
		m.events.Send(&supervisortypes.IndexingEvent{ReplaceBlock: &supervisortypes.BlockReplacement{
			Replacement: x.Ref,
			Invalidated: out.BlockHash,
		}})

	default:
		return false
	}
	return true
}

func (m *IndexingMode) PullEvent() (*supervisortypes.IndexingEvent, error) {
	return m.events.Serve()
}

func (m *IndexingMode) Events(ctx context.Context) (*gethrpc.Subscription, error) {
	return m.events.Subscribe(ctx)
}

func (m *IndexingMode) UpdateCrossUnsafe(ctx context.Context, id eth.BlockID) error {
	l2Ref, err := m.l2.L2BlockRefByHash(ctx, id.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L2BlockRef: %w", err)
	}
	m.emitter.Emit(m.ctx, engine.PromoteCrossUnsafeEvent{
		Ref: l2Ref,
	})
	// We return early: there is no point waiting for the cross-unsafe engine-update synchronously.
	// All error-feedback comes to the supervisor by aborting derivation tasks with an error.
	return nil
}

func (m *IndexingMode) UpdateCrossSafe(ctx context.Context, derived eth.BlockID, derivedFrom eth.BlockID) error {
	l2Ref, err := m.l2.L2BlockRefByHash(ctx, derived.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L2BlockRef: %w", err)
	}
	l1Ref, err := m.l1.L1BlockRefByHash(ctx, derivedFrom.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L1BlockRef: %w", err)
	}
	m.engineController.PromoteSafe(ctx, l2Ref, l1Ref)
	return nil
}

func (m *IndexingMode) UpdateFinalized(ctx context.Context, id eth.BlockID) error {
	l2Ref, err := m.l2.L2BlockRefByHash(ctx, id.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L2BlockRef: %w", err)
	}
	m.engineController.PromoteFinalized(ctx, l2Ref)
	return nil
}

func (m *IndexingMode) InvalidateBlock(ctx context.Context, seal supervisortypes.BlockSeal) error {
	m.log.Info("Invalidating block", "block", seal)

	// Fetch the block we invalidate, so we can re-use the attributes that stay.
	block, err := m.l2.PayloadByHash(ctx, seal.Hash)
	if err != nil { // cannot invalidate if it wasn't there.
		return fmt.Errorf("failed to get block: %w", err)
	}
	parentRef, err := m.l2.L2BlockRefByHash(ctx, block.ExecutionPayload.ParentHash)
	if err != nil {
		return fmt.Errorf("failed to get parent of invalidated block: %w", err)
	}

	ref := block.ExecutionPayload.BlockRef()

	// Create the attributes that we build the replacement block with.
	attributes := AttributesToReplaceInvalidBlock(block)
	annotated := &derive.AttributesWithParent{
		Attributes:  attributes,
		Parent:      parentRef,
		Concluding:  true,
		DerivedFrom: engine.ReplaceBlockSource,
	}

	m.emitter.Emit(m.ctx, engine.InteropInvalidateBlockEvent{
		Invalidated: ref, Attributes: annotated})

	// The node will send an event once the replacement is ready
	return nil
}

func (m *IndexingMode) AnchorPoint(ctx context.Context) (supervisortypes.DerivedBlockRefPair, error) {
	// TODO: maybe cache non-genesis anchor point when seeing safe Interop activation block?
	//  Only needed if we don't test for activation block in the supervisor.
	if !m.cfg.IsInterop(m.cfg.Genesis.L2Time) {
		return supervisortypes.DerivedBlockRefPair{}, &gethrpc.JsonError{
			Code:    InteropInactiveRPCErrCode,
			Message: "Interop inactive at genesis",
		}
	}

	l1Ref, err := m.l1.L1BlockRefByHash(ctx, m.cfg.Genesis.L1.Hash)
	if err != nil {
		return supervisortypes.DerivedBlockRefPair{}, fmt.Errorf("failed to fetch L1 block ref: %w", err)
	}
	l2Ref, err := m.l2.L2BlockRefByHash(ctx, m.cfg.Genesis.L2.Hash)
	if err != nil {
		return supervisortypes.DerivedBlockRefPair{}, fmt.Errorf("failed to fetch L2 block ref: %w", err)
	}
	return supervisortypes.DerivedBlockRefPair{
		Source:  l1Ref,
		Derived: l2Ref.BlockRef(),
	}, nil
}

const (
	InternalErrorRPCErrcode    = -32603
	BlockNotFoundRPCErrCode    = -39001
	ConflictingBlockRPCErrCode = -39002
	InteropInactiveRPCErrCode  = -39003
)

// TODO: add ResetPreInterop, called by supervisor if bisection went pre-Interop. Emit ResetEngineRequestEvent.
func (m *IndexingMode) ResetPreInterop(ctx context.Context) error {
	m.log.Info("Received pre-interop reset request")
	m.emitter.Emit(ctx, engine.ResetEngineRequestEvent{})
	return nil
}

func (m *IndexingMode) Reset(ctx context.Context, lUnsafe, xUnsafe, lSafe, xSafe, finalized eth.BlockID) error {
	logger := m.log.New(
		"localUnsafe", lUnsafe,
		"crossUnsafe", xUnsafe,
		"localSafe", lSafe,
		"crossSafe", xSafe,
		"finalized", finalized)
	logger.Info("Received reset request",
		"localUnsafe", lUnsafe,
		"crossUnsafe", xUnsafe,
		"localSafe", lSafe,
		"crossSafe", xSafe,
		"finalized", finalized)
	verify := func(ref eth.BlockID, name string) (eth.L2BlockRef, error) {
		result, err := m.l2.L2BlockRefByNumber(ctx, ref.Number)
		if err != nil {
			if errors.Is(err, ethereum.NotFound) {
				logger.Warn("Cannot reset, target block not found", "refName", name)
				return eth.L2BlockRef{}, &gethrpc.JsonError{
					Code:    BlockNotFoundRPCErrCode,
					Message: name + " reset target not found",
					Data:    nil, // TODO communicate the latest block that we do have.
				}
			}
			logger.Warn("unable to find reference", "refName", name)
			return eth.L2BlockRef{}, &gethrpc.JsonError{
				Code:    InternalErrorRPCErrcode,
				Message: "failed to find block reference",
				Data:    name,
			}
		}
		if result.Hash != ref.Hash {
			return eth.L2BlockRef{}, &gethrpc.JsonError{
				Code:    ConflictingBlockRPCErrCode,
				Message: "conflicting block",
				Data:    result,
			}
		}
		return result, nil
	}

	// verify all provided references
	lUnsafeRef, err := verify(lUnsafe, "unsafe")
	if err != nil {
		logger.Error("Cannot reset, local-unsafe target invalid")
		return err
	}
	xUnsafeRef, err := verify(xUnsafe, "cross-unsafe")
	if err != nil {
		logger.Error("Cannot reset, cross-safe target invalid")
		return err
	}
	lSafeRef, err := verify(lSafe, "safe")
	if err != nil {
		logger.Error("Cannot reset, local-safe target invalid")
		return err
	}
	xSafeRef, err := verify(xSafe, "cross-safe")
	if err != nil {
		logger.Error("Cannot reset, cross-safe target invalid")
		return err
	}
	finalizedRef, err := verify(finalized, "finalized")
	if err != nil {
		logger.Error("Cannot reset, finalized block not known")
		return err
	}

	// sanity check for max-reorg depth and pre-interop check
	if err := m.sanityCheck(ctx, logger, lUnsafeRef); err != nil {
		return err
	}

	m.engineController.ForceReset(ctx, lUnsafeRef, xUnsafeRef, lSafeRef, xSafeRef, finalizedRef)
	return nil
}

func (m *IndexingMode) ProvideL1(ctx context.Context, nextL1 eth.BlockRef) error {
	m.log.Info("Received next L1 block", "nextL1", nextL1)
	m.emitter.Emit(m.ctx, derive.ProvideL1Traversal{
		NextL1: nextL1,
	})
	return nil
}

func (m *IndexingMode) FetchReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	_, receipts, err := m.l2.FetchReceipts(ctx, blockHash)
	return receipts, err
}

func (m *IndexingMode) ChainID(ctx context.Context) (eth.ChainID, error) {
	return eth.ChainIDFromBig(m.cfg.L2ChainID), nil
}

func (m *IndexingMode) OutputV0AtTimestamp(ctx context.Context, timestamp uint64) (*eth.OutputV0, error) {
	ref, err := m.L2BlockRefByTimestamp(ctx, timestamp)
	if err != nil {
		return nil, err
	}
	return m.l2.OutputV0AtBlock(ctx, ref.Hash)
}

func (m *IndexingMode) PendingOutputV0AtTimestamp(ctx context.Context, timestamp uint64) (*eth.OutputV0, error) {
	ref, err := m.L2BlockRefByTimestamp(ctx, timestamp)
	if err != nil {
		return nil, err
	}
	if ref.Number == 0 {
		// The genesis block cannot have been invalid
		return m.l2.OutputV0AtBlock(ctx, ref.Hash)
	}

	payload, err := m.l2.PayloadByHash(ctx, ref.Hash)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block (%v): %w", ref, err)
	}
	optimisticOutput, err := DecodeInvalidatedBlockTxFromReplacement(payload.ExecutionPayload.Transactions)
	if errors.Is(err, ErrNotReplacementBlock) {
		// This block was not replaced so use the canonical output root as pending
		return m.l2.OutputV0AtBlock(ctx, ref.Hash)
	} else if err != nil {
		return nil, fmt.Errorf("failed parse replacement block (%v): %w", ref, err)
	}
	return optimisticOutput, nil
}

func (m *IndexingMode) L2BlockRefByTimestamp(ctx context.Context, timestamp uint64) (eth.L2BlockRef, error) {
	num, err := m.cfg.TargetBlockNumber(timestamp)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	return m.l2.L2BlockRefByNumber(ctx, num)
}

func (m *IndexingMode) L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error) {
	return m.l2.L2BlockRefByNumber(ctx, num)
}

func (m *IndexingMode) sanityCheck(ctx context.Context, logger log.Logger, proposedUnsafe eth.L2BlockRef) error {
	currentUnsafe, err := m.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return fmt.Errorf("failed to get previous unsafe block: %w", err)
	}

	// check we are not reorging L2 incredibly deep
	if proposedUnsafe.L1Origin.Number+(sync.MaxReorgSeqWindows*m.cfg.SyncLookback()) < currentUnsafe.L1Origin.Number {
		// If the reorg depth is too large, something is fishy.
		// This can legitimately happen if L1 goes down for a while. But in that case,
		// restarting the L2 node with a bigger configured MaxReorgDepth is an acceptable
		// stopgap solution.
		logger.Error("reorg is too deep", "proposed_l1origin", proposedUnsafe.L1Origin.Number, "currentUnsafe_l1origin", currentUnsafe.L1Origin.Number, "sync_lookback", m.cfg.SyncLookback())
		return fmt.Errorf("%w: traversed back to L2 block %s, but too deep compared to previous unsafe block %s", sync.TooDeepReorgErr, proposedUnsafe, currentUnsafe)
	}

	// check we are not reorging to a non-interop block
	if !m.cfg.IsInterop(proposedUnsafe.Time) {
		err := fmt.Errorf("proposed local-unsafe block %s found to be reorged to is not interop-enabled", proposedUnsafe)
		logger.Error(err.Error(), "block", proposedUnsafe)
		return err
	}

	return nil
}
