package derive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

type NextAttributesProvider interface {
	Origin() eth.L1BlockRef
	NextAttributes(context.Context, eth.L2BlockRef) (*eth.PayloadAttributes, error)
}

type Engine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayload, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

// Max memory used for buffering unsafe payloads
const maxUnsafePayloadsMemory = 500 * 1024 * 1024

// finalityLookback defines the amount of L1<>L2 relations to track for finalization purposes, one per L1 block.
//
// When L1 finalizes blocks, it finalizes finalityLookback blocks behind the L1 head.
// Non-finality may take longer, but when it does finalize again, it is within this range of the L1 head.
// Thus we only need to retain the L1<>L2 derivation relation data of this many L1 blocks.
//
// In the event of older finalization signals, misconfiguration, or insufficient L1<>L2 derivation relation data,
// then we may miss the opportunity to finalize more L2 blocks.
// This does not cause any divergence, it just causes lagging finalization status.
//
// The beacon chain on mainnet has 32 slots per epoch,
// and new finalization events happen at most 4 epochs behind the head.
// And then we add 1 to make pruning easier by leaving room for a new item without pruning the 32*4.
const finalityLookback = 4*32 + 1

type FinalityData struct {
	// The last L2 block that was fully derived and inserted into the L2 engine while processing this L1 block.
	L2Block eth.L2BlockRef
	// The L1 block this stage was at when inserting the L2 block.
	// When this L1 block is finalized, the L2 chain up to this block can be fully reproduced from finalized L1 data.
	L1Block eth.BlockID
}

// EngineQueue queues up payload attributes to consolidate or process with the provided Engine
type EngineQueue struct {
	log log.Logger
	cfg *rollup.Config

	finalized  eth.L2BlockRef
	safeHead   eth.L2BlockRef
	unsafeHead eth.L2BlockRef

	finalizedL1 eth.BlockID

	safeAttributes []*eth.PayloadAttributes
	unsafePayloads PayloadsQueue // queue of unsafe payloads, ordered by ascending block number, may have gaps

	// Tracks which L2 blocks where last derived from which L1 block. At most finalityLookback large.
	finalityData []FinalityData

	engine Engine
	prev   NextAttributesProvider

	progress Progress // only used for pipeline resets

	metrics Metrics
}

// NewEngineQueue creates a new EngineQueue, which should be Reset(origin) before use.
func NewEngineQueue(log log.Logger, cfg *rollup.Config, engine Engine, metrics Metrics, prev NextAttributesProvider) *EngineQueue {
	return &EngineQueue{
		log:          log,
		cfg:          cfg,
		engine:       engine,
		metrics:      metrics,
		finalityData: make([]FinalityData, 0, finalityLookback),
		unsafePayloads: PayloadsQueue{
			MaxSize: maxUnsafePayloadsMemory,
			SizeFn:  payloadMemSize,
		},
		prev: prev,
	}
}

func (eq *EngineQueue) Progress() Progress {
	return eq.progress
}

func (eq *EngineQueue) SetUnsafeHead(head eth.L2BlockRef) {
	eq.unsafeHead = head
	eq.metrics.RecordL2Ref("l2_unsafe", head)
}

func (eq *EngineQueue) AddUnsafePayload(payload *eth.ExecutionPayload) {
	if payload == nil {
		eq.log.Warn("cannot add nil unsafe payload")
		return
	}
	if err := eq.unsafePayloads.Push(payload); err != nil {
		eq.log.Warn("Could not add unsafe payload", "id", payload.ID(), "timestamp", uint64(payload.Timestamp), "err", err)
		return
	}
	p := eq.unsafePayloads.Peek()
	eq.metrics.RecordUnsafePayloadsBuffer(uint64(eq.unsafePayloads.Len()), eq.unsafePayloads.MemSize(), p.ID())
	eq.log.Trace("Next unsafe payload to process", "next", p.ID(), "timestamp", uint64(p.Timestamp))
}

func (eq *EngineQueue) AddSafeAttributes(attributes *eth.PayloadAttributes) {
	eq.log.Trace("Adding next safe attributes", "timestamp", attributes.Timestamp)
	eq.safeAttributes = append(eq.safeAttributes, attributes)
}

func (eq *EngineQueue) Finalize(l1Origin eth.BlockID) {
	eq.finalizedL1 = l1Origin
	eq.tryFinalizeL2()
}

func (eq *EngineQueue) Finalized() eth.L2BlockRef {
	return eq.finalized
}

func (eq *EngineQueue) UnsafeL2Head() eth.L2BlockRef {
	return eq.unsafeHead
}

func (eq *EngineQueue) SafeL2Head() eth.L2BlockRef {
	return eq.safeHead
}

func (eq *EngineQueue) LastL2Time() uint64 {
	if len(eq.safeAttributes) == 0 {
		return eq.safeHead.Time
	}
	return uint64(eq.safeAttributes[len(eq.safeAttributes)-1].Timestamp)
}

func (eq *EngineQueue) Step(ctx context.Context, _ Progress) error {
	if len(eq.safeAttributes) > 0 {
		return eq.tryNextSafeAttributes(ctx)
	}
	outOfData := false
	if len(eq.safeAttributes) == 0 {
		if next, err := eq.prev.NextAttributes(ctx, eq.safeHead); err == io.EOF {
			outOfData = true
		} else if err != nil {
			return err
		} else {
			eq.safeAttributes = append(eq.safeAttributes, next)
			return NotEnoughData
		}
	}
	if eq.unsafePayloads.Len() > 0 {
		return eq.tryNextUnsafePayload(ctx)
	}

	if outOfData {
		return io.EOF
	} else {
		return nil
	}
}

// tryFinalizeL2 traverses the past L1 blocks, checks if any has been finalized,
// and then marks the latest fully derived L2 block from this as finalized,
// or defaults to the current finalized L2 block.
func (eq *EngineQueue) tryFinalizeL2() {
	if eq.finalizedL1 == (eth.BlockID{}) {
		return // if no L1 information is finalized yet, then skip this
	}
	// default to keep the same finalized block
	finalizedL2 := eq.finalized
	// go through the latest inclusion data, and find the last L2 block that was derived from a finalized L1 block
	for _, fd := range eq.finalityData {
		if fd.L2Block.Number > finalizedL2.Number && fd.L1Block.Number <= eq.finalizedL1.Number {
			finalizedL2 = fd.L2Block
		}
	}
	eq.finalized = finalizedL2
	eq.metrics.RecordL2Ref("l2_finalized", finalizedL2)
}

// postProcessSafeL2 buffers the L1 block the safe head was fully derived from,
// to finalize it once the L1 block, or later, finalizes.
func (eq *EngineQueue) postProcessSafeL2() {
	// prune finality data if necessary
	if len(eq.finalityData) >= finalityLookback {
		eq.finalityData = append(eq.finalityData[:0], eq.finalityData[1:finalityLookback]...)
	}
	// remember the last L2 block that we fully derived from the given finality data
	if len(eq.finalityData) == 0 || eq.finalityData[len(eq.finalityData)-1].L1Block.Number < eq.prev.Origin().Number {
		// append entry for new L1 block
		eq.finalityData = append(eq.finalityData, FinalityData{
			L2Block: eq.safeHead,
			L1Block: eq.prev.Origin().ID(),
		})
	} else {
		// if it's a now L2 block that was derived from the same latest L1 block, then just update the entry
		eq.finalityData[len(eq.finalityData)-1].L2Block = eq.safeHead
	}
}

func (eq *EngineQueue) logSyncProgress(reason string) {
	eq.log.Info("Sync progress",
		"reason", reason,
		"l2_finalized", eq.finalized,
		"l2_safe", eq.safeHead,
		"l2_unsafe", eq.unsafeHead,
		"l2_time", eq.unsafeHead.Time,
		"l1_derived", eq.prev.Origin(),
	)
}

func (eq *EngineQueue) tryNextUnsafePayload(ctx context.Context) error {
	first := eq.unsafePayloads.Peek()

	if uint64(first.BlockNumber) <= eq.safeHead.Number {
		eq.log.Info("skipping unsafe payload, since it is older than safe head", "safe", eq.safeHead.ID(), "unsafe", first.ID(), "payload", first.ID())
		eq.unsafePayloads.Pop()
		return nil
	}

	// Ensure that the unsafe payload builds upon the current unsafe head
	// TODO: once we support snap-sync we can remove this condition, and handle the "SYNCING" status of the execution engine.
	if first.ParentHash != eq.unsafeHead.Hash {
		if uint64(first.BlockNumber) == eq.unsafeHead.Number+1 {
			eq.log.Info("skipping unsafe payload, since it does not build onto the existing unsafe chain", "safe", eq.safeHead.ID(), "unsafe", first.ID(), "payload", first.ID())
			eq.unsafePayloads.Pop()
		}
		return io.EOF // time to go to next stage if we cannot process the first unsafe payload
	}

	ref, err := PayloadToBlockRef(first, &eq.cfg.Genesis)
	if err != nil {
		eq.log.Error("failed to decode L2 block ref from payload", "err", err)
		eq.unsafePayloads.Pop()
		return nil
	}

	status, err := eq.engine.NewPayload(ctx, first)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to update insert payload: %w", err))
	}
	if status.Status != eth.ExecutionValid {
		eq.unsafePayloads.Pop()
		return NewTemporaryError(fmt.Errorf("cannot process unsafe payload: new - %v; parent: %v; err: %w",
			first.ID(), first.ParentID(), eth.NewPayloadErr(first, status)))
	}

	// Mark the new payload as valid
	fc := eth.ForkchoiceState{
		HeadBlockHash:      first.BlockHash,
		SafeBlockHash:      eq.safeHead.Hash, // this should guarantee we do not reorg past the safe head
		FinalizedBlockHash: eq.finalized.Hash,
	}
	fcRes, err := eq.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var inputErr eth.InputError
		if errors.As(err, &inputErr) {
			switch inputErr.Code {
			case eth.InvalidForkchoiceState:
				return NewResetError(fmt.Errorf("pre-unsafe-block forkchoice update was inconsistent with engine, need reset to resolve: %w", inputErr.Unwrap()))
			default:
				return NewTemporaryError(fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err))
			}
		} else {
			return NewTemporaryError(fmt.Errorf("failed to update forkchoice to prepare for new unsafe payload: %w", err))
		}
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		eq.unsafePayloads.Pop()
		return NewTemporaryError(fmt.Errorf("cannot prepare unsafe chain for new payload: new - %v; parent: %v; err: %w",
			first.ID(), first.ParentID(), eth.ForkchoiceUpdateErr(fcRes.PayloadStatus)))
	}

	eq.unsafeHead = ref
	eq.unsafePayloads.Pop()
	eq.metrics.RecordL2Ref("l2_unsafe", ref)
	eq.log.Trace("Executed unsafe payload", "hash", ref.Hash, "number", ref.Number, "timestamp", ref.Time, "l1Origin", ref.L1Origin)
	eq.logSyncProgress("unsafe payload from sequencer")

	return nil
}

func (eq *EngineQueue) tryNextSafeAttributes(ctx context.Context) error {
	if eq.safeHead.Number < eq.unsafeHead.Number {
		return eq.consolidateNextSafeAttributes(ctx)
	} else if eq.safeHead.Number == eq.unsafeHead.Number {
		return eq.forceNextSafeAttributes(ctx)
	} else {
		// For some reason the unsafe head is behind the safe head. Log it, and correct it.
		eq.log.Error("invalid sync state, unsafe head is behind safe head", "unsafe", eq.unsafeHead, "safe", eq.safeHead)
		eq.unsafeHead = eq.safeHead
		eq.metrics.RecordL2Ref("l2_unsafe", eq.unsafeHead)
		return nil
	}
}

// consolidateNextSafeAttributes tries to match the next safe attributes against the existing unsafe chain,
// to avoid extra processing or unnecessary unwinding of the chain.
// However, if the attributes do not match, they will be forced with forceNextSafeAttributes.
func (eq *EngineQueue) consolidateNextSafeAttributes(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	payload, err := eq.engine.PayloadByNumber(ctx, eq.safeHead.Number+1)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// engine may have restarted, or inconsistent safe head. We need to reset
			return NewResetError(fmt.Errorf("expected engine was synced and had unsafe block to reconcile, but cannot find the block: %w", err))
		}
		return NewTemporaryError(fmt.Errorf("failed to get existing unsafe payload to compare against derived attributes from L1: %w", err))
	}
	if err := AttributesMatchBlock(eq.safeAttributes[0], eq.safeHead.Hash, payload); err != nil {
		eq.log.Warn("L2 reorg: existing unsafe block does not match derived attributes from L1", "err", err)
		// geth cannot wind back a chain without reorging to a new, previously non-canonical, block
		return eq.forceNextSafeAttributes(ctx)
	}
	ref, err := PayloadToBlockRef(payload, &eq.cfg.Genesis)
	if err != nil {
		return NewResetError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}
	eq.safeHead = ref
	eq.metrics.RecordL2Ref("l2_safe", ref)
	// unsafe head stays the same, we did not reorg the chain.
	eq.safeAttributes = eq.safeAttributes[1:]
	eq.postProcessSafeL2()
	eq.logSyncProgress("reconciled with L1")

	return nil
}

// forceNextSafeAttributes inserts the provided attributes, reorging away any conflicting unsafe chain.
func (eq *EngineQueue) forceNextSafeAttributes(ctx context.Context) error {
	if len(eq.safeAttributes) == 0 {
		return nil
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      eq.safeHead.Hash,
		SafeBlockHash:      eq.safeHead.Hash,
		FinalizedBlockHash: eq.finalized.Hash,
	}
	attrs := eq.safeAttributes[0]
	payload, errType, err := InsertHeadBlock(ctx, eq.log, eq.engine, fc, attrs, true)
	if err != nil {
		switch errType {
		case BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			return NewTemporaryError(fmt.Errorf("temporarily cannot insert new safe block: %w", err))
		case BlockInsertPrestateErr:
			return NewResetError(fmt.Errorf("need reset to resolve pre-state problem: %w", err))
		case BlockInsertPayloadErr:
			eq.log.Warn("could not process payload derived from L1 data", "err", err)
			// filter everything but the deposits
			var deposits []hexutil.Bytes
			for _, tx := range attrs.Transactions {
				if len(tx) > 0 && tx[0] == types.DepositTxType {
					deposits = append(deposits, tx)
				}
			}
			if len(attrs.Transactions) > len(deposits) {
				eq.log.Warn("dropping sequencer transactions from payload for re-attempt, batcher may have included invalid transactions",
					"txs", len(attrs.Transactions), "deposits", len(deposits), "parent", eq.safeHead)
				eq.safeAttributes[0].Transactions = deposits
				return nil
			}
			return NewCriticalError(fmt.Errorf("failed to process block with only deposit transactions: %w", err))
		default:
			return NewCriticalError(fmt.Errorf("unknown InsertHeadBlock error type %d: %w", errType, err))
		}
	}
	ref, err := PayloadToBlockRef(payload, &eq.cfg.Genesis)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}
	eq.safeHead = ref
	eq.unsafeHead = ref
	eq.metrics.RecordL2Ref("l2_safe", ref)
	eq.metrics.RecordL2Ref("l2_unsafe", ref)
	eq.safeAttributes = eq.safeAttributes[1:]
	eq.postProcessSafeL2()
	eq.logSyncProgress("processed safe block derived from L1")

	return nil
}

// ResetStep Walks the L2 chain backwards until it finds an L2 block whose L1 origin is canonical.
// The unsafe head is set to the head of the L2 chain, unless the existing safe head is not canonical.
func (eq *EngineQueue) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	result, err := sync.FindL2Heads(ctx, eq.cfg, l1Fetcher, eq.engine)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find the L2 Heads to start from: %w", err))
	}
	finalized, safe, unsafe := result.Finalized, result.Safe, result.Unsafe
	l1Origin, err := l1Fetcher.L1BlockRefByHash(ctx, safe.L1Origin.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch the new L1 progress: origin: %v; err: %w", safe.L1Origin, err))
	}
	if safe.Time < l1Origin.Time {
		return NewResetError(fmt.Errorf("cannot reset block derivation to start at L2 block %s with time %d older than its L1 origin %s with time %d, time invariant is broken",
			safe, safe.Time, l1Origin, l1Origin.Time))
	}

	pipelineNumber := l1Origin.Number - eq.cfg.ChannelTimeout
	if l1Origin.Number < eq.cfg.ChannelTimeout {
		pipelineNumber = 0
	}
	pipelineOrigin, err := l1Fetcher.L1BlockRefByNumber(ctx, pipelineNumber)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch the new L1 progress: origin: %v; err: %w", pipelineNumber, err))
	}
	eq.log.Debug("Reset engine queue", "safeHead", safe, "unsafe", unsafe, "safe_timestamp", safe.Time, "unsafe_timestamp", unsafe.Time, "l1Origin", l1Origin)
	eq.unsafeHead = unsafe
	eq.safeHead = safe
	eq.finalized = finalized
	eq.finalityData = eq.finalityData[:0]
	// note: we do not clear the unsafe payloadds queue; if the payloads are not applicable anymore the parent hash checks will clear out the old payloads.
	eq.progress = Progress{
		Origin: pipelineOrigin,
	}
	eq.metrics.RecordL2Ref("l2_finalized", finalized)
	eq.metrics.RecordL2Ref("l2_safe", safe)
	eq.metrics.RecordL2Ref("l2_unsafe", unsafe)
	eq.logSyncProgress("reset derivation work")
	return io.EOF
}
