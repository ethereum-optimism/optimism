package managed

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
	"github.com/ethereum-optimism/optimism/op-service/binary"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	"github.com/ethereum-optimism/optimism/op-service/rpc"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// managedEventStream abstracts the event stream functionality for testing
type managedEventStream interface {
	Send(event *supervisortypes.ManagedEvent)
	Serve() (*supervisortypes.ManagedEvent, error)
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

// ManagedMode makes the op-node managed by an op-supervisor,
// by serving sync work and updating the canonical chain based on instructions.
type ManagedMode struct {
	log log.Logger

	emitter event.Emitter

	l1 L1Source
	l2 L2Source

	events managedEventStream

	// outgoing event timestamp trackers
	lastReset         eventTimestamp[struct{}]
	lastUnsafe        eventTimestamp[eth.BlockID]
	lastSafe          eventTimestamp[eth.BlockID]
	lastL1Traversal   eventTimestamp[eth.BlockID]
	lastExhaustedL1   eventTimestamp[eth.BlockID]
	lastReplacedBlock eventTimestamp[eth.BlockID]

	cfg *rollup.Config

	srv       *rpc.Server
	jwtSecret eth.Bytes32
}

func NewManagedMode(log log.Logger, cfg *rollup.Config, addr string, port int, jwtSecret eth.Bytes32, l1 L1Source, l2 L2Source, m opmetrics.RPCMetricer) *ManagedMode {
	log = log.With("mode", "managed", "chainId", cfg.L2ChainID)
	out := &ManagedMode{
		log:       log,
		cfg:       cfg,
		l1:        l1,
		l2:        l2,
		jwtSecret: jwtSecret,
		events:    rpc.NewStream[supervisortypes.ManagedEvent](log, 100),

		lastReset:         newEventTimestamp[struct{}](100 * time.Millisecond),
		lastUnsafe:        newEventTimestamp[eth.BlockID](100 * time.Millisecond),
		lastSafe:          newEventTimestamp[eth.BlockID](100 * time.Millisecond),
		lastL1Traversal:   newEventTimestamp[eth.BlockID](500 * time.Millisecond),
		lastExhaustedL1:   newEventTimestamp[eth.BlockID](500 * time.Millisecond),
		lastReplacedBlock: newEventTimestamp[eth.BlockID](100 * time.Millisecond),
	}

	out.srv = rpc.NewServer(addr, port, "v0.0.0",
		rpc.WithWebsocketEnabled(),
		rpc.WithLogger(log),
		rpc.WithJWTSecret(jwtSecret[:]),
		rpc.WithRPCRecorder(m.NewRecorder("interop_managed")),
	)
	out.srv.AddAPI(gethrpc.API{
		Namespace:     "interop",
		Service:       &InteropAPI{backend: out},
		Authenticated: true,
	})
	return out
}

// TestDisableEventDeduplication is a test-only function that disables event deduplication.
// It is necessary to make action tests work.
func (m *ManagedMode) TestDisableEventDeduplication() {
	m.lastReset.ttl = 0
	m.lastUnsafe.ttl = 0
	m.lastSafe.ttl = 0
	m.lastL1Traversal.ttl = 0
	m.lastExhaustedL1.ttl = 0
	m.lastReplacedBlock.ttl = 0
}

func (m *ManagedMode) Start(ctx context.Context) error {
	if m.emitter == nil {
		return errors.New("must have emitter before starting")
	}
	if err := m.srv.Start(); err != nil {
		return fmt.Errorf("failed to start interop RPC server: %w", err)
	}
	m.log.Info("Started interop RPC", "endpoint", m.WSEndpoint())
	return nil
}

func (m *ManagedMode) WSEndpoint() string {
	return fmt.Sprintf("ws://%s", m.srv.Endpoint())
}

func (m *ManagedMode) WSPort() (int, error) {
	return m.srv.Port()
}

func (m *ManagedMode) JWTSecret() eth.Bytes32 {
	return m.jwtSecret
}

func (m *ManagedMode) Stop(ctx context.Context) error {
	// stop RPC server
	if err := m.srv.Stop(); err != nil {
		return fmt.Errorf("failed to stop interop sub-system RPC server: %w", err)
	}

	m.log.Info("Interop sub-system stopped")
	return nil
}

func (m *ManagedMode) AttachEmitter(em event.Emitter) {
	m.emitter = em
}

// Outgoing events to supervisor
func (m *ManagedMode) OnEvent(ev event.Event) bool {
	switch x := ev.(type) {
	case rollup.ResetEvent:
		logger := m.log.New("err", x.Err)
		logger.Warn("Sending reset request to supervisor")
		if !m.lastReset.Update(struct{}{}) {
			logger.Warn("Skipped sending duplicate reset request")
			return true
		}
		msg := x.Err.Error()
		m.events.Send(&supervisortypes.ManagedEvent{Reset: &msg})

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
		m.events.Send(&supervisortypes.ManagedEvent{UnsafeBlock: &ref})

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
		m.events.Send(&supervisortypes.ManagedEvent{
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
		m.events.Send(&supervisortypes.ManagedEvent{
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
		m.events.Send(&supervisortypes.ManagedEvent{
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
		m.events.Send(&supervisortypes.ManagedEvent{ReplaceBlock: &supervisortypes.BlockReplacement{
			Replacement: x.Ref,
			Invalidated: out.BlockHash,
		}})

	default:
		return false
	}
	return true
}

func (m *ManagedMode) PullEvent() (*supervisortypes.ManagedEvent, error) {
	return m.events.Serve()
}

func (m *ManagedMode) Events(ctx context.Context) (*gethrpc.Subscription, error) {
	return m.events.Subscribe(ctx)
}

func (m *ManagedMode) UpdateCrossUnsafe(ctx context.Context, id eth.BlockID) error {
	l2Ref, err := m.l2.L2BlockRefByHash(ctx, id.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L2BlockRef: %w", err)
	}
	m.emitter.Emit(engine.PromoteCrossUnsafeEvent{
		Ref: l2Ref,
		Ctx: event.WrapCtx(ctx),
	})
	// We return early: there is no point waiting for the cross-unsafe engine-update synchronously.
	// All error-feedback comes to the supervisor by aborting derivation tasks with an error.
	return nil
}

func (m *ManagedMode) UpdateCrossSafe(ctx context.Context, derived eth.BlockID, derivedFrom eth.BlockID) error {
	l2Ref, err := m.l2.L2BlockRefByHash(ctx, derived.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L2BlockRef: %w", err)
	}
	l1Ref, err := m.l1.L1BlockRefByHash(ctx, derivedFrom.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L1BlockRef: %w", err)
	}
	m.emitter.Emit(engine.PromoteSafeEvent{
		Ref:    l2Ref,
		Source: l1Ref,
		Ctx:    event.WrapCtx(ctx),
	})
	// We return early: there is no point waiting for the cross-safe engine-update synchronously.
	// All error-feedback comes to the supervisor by aborting derivation tasks with an error.
	return nil
}

func (m *ManagedMode) UpdateFinalized(ctx context.Context, id eth.BlockID) error {
	l2Ref, err := m.l2.L2BlockRefByHash(ctx, id.Hash)
	if err != nil {
		return fmt.Errorf("failed to get L2BlockRef: %w", err)
	}
	m.emitter.Emit(engine.PromoteFinalizedEvent{Ref: l2Ref, Ctx: event.WrapCtx(ctx)})
	// We return early: there is no point waiting for the finalized engine-update synchronously.
	// All error-feedback comes to the supervisor by aborting derivation tasks with an error.
	return nil
}

func (m *ManagedMode) InvalidateBlock(ctx context.Context, seal supervisortypes.BlockSeal) error {
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

	m.emitter.Emit(engine.InteropInvalidateBlockEvent{
		Invalidated: ref, Attributes: annotated, Ctx: event.WrapCtx(ctx)})

	// The node will send an event once the replacement is ready
	return nil
}

func (m *ManagedMode) AnchorPoint(ctx context.Context) (supervisortypes.DerivedBlockRefPair, error) {
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
func (m *ManagedMode) ResetPreInterop(ctx context.Context) error {
	m.log.Info("Received pre-interop reset request")
	m.emitter.Emit(engine.ResetEngineRequestEvent{Ctx: event.WrapCtx(ctx)})
	return nil
}

func (m *ManagedMode) Reset(ctx context.Context, lUnsafe, xUnsafe, lSafe, xSafe, finalized eth.BlockID) error {
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
	_, err := verify(lUnsafe, "unsafe")
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

	latestLocalUnsafe, err := m.findLatestValidLocalUnsafe(ctx, lUnsafe)
	if err != nil {
		logger.Error("Cannot reset, no valid local-unsafe block found", "err", err)
		return err
	}

	m.emitter.Emit(rollup.ForceResetEvent{
		LocalUnsafe: latestLocalUnsafe,
		CrossUnsafe: xUnsafeRef,
		LocalSafe:   lSafeRef,
		CrossSafe:   xSafeRef,
		Finalized:   finalizedRef,
		Ctx:         event.WrapCtx(ctx),
	})
	return nil
}

// findLatestValidLocalUnsafe searches and returns the latest valid block of the L2 chain
// starting from `l2UnsafeTarget` and checking until the latest unsafe block.
func (m *ManagedMode) findLatestValidLocalUnsafe(ctx context.Context, l2UnsafeTarget eth.BlockID) (eth.L2BlockRef, error) {
	latestUnsafe, err := m.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return eth.L2BlockRef{}, err
	}

	logger := m.log.New("target", l2UnsafeTarget, "latestUnsafe", latestUnsafe)
	target := l2UnsafeTarget.Number

	logger.Info("Searching for latest valid local unsafe")

	targetDiff := int(latestUnsafe.Number - target)
	if targetDiff > 0 {
		// Binary search to find and return the last valid block for idx in [0, targetDiff)
		// We don't check validity of `target`, `target` is not in the search space, it is checked
		// in the walkback loop section below if necessary.

		// Search space:
		// ------------------------------------------------------------------------------------------
		// target.Number |  idx=0      idx=1      idx=2     ...  idx = targetDiff-1 = latestUnsafe   |
		// false         |  t/f        t/f        t/f       ...  t/f                                 |
		// ------------------------------------------------------------------------------------------
		idx, valid, err := binary.SearchL(targetDiff, func(i int) (bool, eth.L2BlockRef, error) {
			block, err := m.verifyBlock(ctx, logger, target+1+uint64(i))
			return block != (eth.L2BlockRef{}), block, err
		})
		if err != nil {
			return eth.L2BlockRef{}, err
		}

		if idx != -1 {
			logger.Info("Found last valid block with binary search", "valid", valid)
			return valid, nil
		} else {
			logger.Info("All blocks checked by binary search are invalid between target and latestUnsafe")
		}
	} else if targetDiff < 0 {
		logger.Warn("Latest unsafe block is older than target, using latest unsafe for search")
		target = latestUnsafe.Number
	}

	// In the following walkback loop, the following two cases are covered:
	// 1. targetDiff == 0 or targetDiff < 0 (i.e. target == latestUnsafe), or
	// 2. all blocks checked by binary search were invalid, so we have to go from `target` backwards indefinitely
	//    until we find a valid block
	for n := target; ; n-- {
		if n == target-1 {
			logger.Warn("No valid unsafe block found up to target, searching further")
		}

		valid, err := m.verifyBlock(ctx, logger, n)
		if err != nil {
			return eth.L2BlockRef{}, err
		}

		if valid != (eth.L2BlockRef{}) {
			logger.Info("Fould last valid block", "valid", valid)
			return valid, nil
		}
	}
}

// verifyBlock
func (m *ManagedMode) verifyBlock(ctx context.Context, logger log.Logger, blockNum uint64) (eth.L2BlockRef, error) {
	current, err := m.l2.L2BlockRefByNumber(ctx, blockNum)
	if err != nil {
		return eth.L2BlockRef{}, err
	}

	// Check if L1Origin has been reorged
	l1Blk, err := m.l1.L1BlockRefByNumber(ctx, current.L1Origin.Number)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	if l1Blk.Hash != current.L1Origin.Hash {
		logger.Debug("L1Origin field is invalid/outdated, so block is invalid and should be reorged", "currentNumber", current.Number, "currentL1Origin", current.L1Origin, "newL1Origin", l1Blk)
		return eth.L2BlockRef{}, nil
	}
	logger.Trace("L1Origin field points to canonical L1 block, so block is valid", "blocknum", blockNum, "l1Blk", l1Blk)
	return current, nil
}

func (m *ManagedMode) ProvideL1(ctx context.Context, nextL1 eth.BlockRef) error {
	m.log.Info("Received next L1 block", "nextL1", nextL1)
	m.emitter.Emit(derive.ProvideL1Traversal{
		NextL1: nextL1,
		Ctx:    event.WrapCtx(ctx),
	})
	return nil
}

func (m *ManagedMode) FetchReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	_, receipts, err := m.l2.FetchReceipts(ctx, blockHash)
	return receipts, err
}

func (m *ManagedMode) BlockRefByNumber(ctx context.Context, num uint64) (eth.BlockRef, error) {
	return m.l2.BlockRefByNumber(ctx, num)
}

func (m *ManagedMode) ChainID(ctx context.Context) (eth.ChainID, error) {
	return eth.ChainIDFromBig(m.cfg.L2ChainID), nil
}

func (m *ManagedMode) OutputV0AtTimestamp(ctx context.Context, timestamp uint64) (*eth.OutputV0, error) {
	ref, err := m.L2BlockRefByTimestamp(ctx, timestamp)
	if err != nil {
		return nil, err
	}
	return m.l2.OutputV0AtBlock(ctx, ref.Hash)
}

func (m *ManagedMode) PendingOutputV0AtTimestamp(ctx context.Context, timestamp uint64) (*eth.OutputV0, error) {
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

func (m *ManagedMode) L2BlockRefByTimestamp(ctx context.Context, timestamp uint64) (eth.L2BlockRef, error) {
	num, err := m.cfg.TargetBlockNumber(timestamp)
	if err != nil {
		return eth.L2BlockRef{}, err
	}
	return m.l2.L2BlockRefByNumber(ctx, num)
}
