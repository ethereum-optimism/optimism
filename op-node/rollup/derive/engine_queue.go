package derive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
)

type attributesWithParent struct {
	attributes *eth.PayloadAttributes
	parent     eth.L2BlockRef
}

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
	SystemConfigL2Fetcher
}

// EngineState provides a read-only interface of the forkchoice state properties of the L2 Engine.
type EngineState interface {
	Finalized() eth.L2BlockRef
	UnsafeL2Head() eth.L2BlockRef
	SafeL2Head() eth.L2BlockRef
}

// EngineControl enables other components to build blocks with the Engine,
// while keeping the forkchoice state and payload-id management internal to
// avoid state inconsistencies between different users of the EngineControl.
type EngineControl interface {
	EngineState

	// StartPayload requests the engine to start building a block with the given attributes.
	// If updateSafe, the resulting block will be marked as a safe block.
	StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *eth.PayloadAttributes, updateSafe bool) (errType BlockInsertionErrType, err error)
	// ConfirmPayload requests the engine to complete the current block. If no block is being built, or if it fails, an error is returned.
	ConfirmPayload(ctx context.Context) (out *eth.ExecutionPayload, errTyp BlockInsertionErrType, err error)
	// CancelPayload requests the engine to stop building the current block without making it canonical.
	// This is optional, as the engine expires building jobs that are left uncompleted, but can still save resources.
	CancelPayload(ctx context.Context, force bool) error
	// BuildingPayload indicates if a payload is being built, and onto which block it is being built, and whether or not it is a safe payload.
	BuildingPayload() (onto eth.L2BlockRef, id eth.PayloadID, safe bool)
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

// finalityDelay is the number of L1 blocks to traverse before trying to finalize L2 blocks again.
// We do not want to do this too often, since it requires fetching a L1 block by number, so no cache data.
const finalityDelay = 64

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

	buildingOnto eth.L2BlockRef
	buildingID   eth.PayloadID
	buildingSafe bool

	// Track when the rollup node changes the forkchoice without engine action,
	// e.g. on a reset after a reorg, or after consolidating a block.
	// This update may repeat if the engine returns a temporary error.
	needForkchoiceUpdate bool

	// finalizedL1 is the currently perceived finalized L1 block.
	// This may be ahead of the current traversed origin when syncing.
	finalizedL1 eth.L1BlockRef

	// triedFinalizeAt tracks at which origin we last tried to finalize during sync.
	triedFinalizeAt eth.L1BlockRef

	// The queued-up attributes
	safeAttributes *attributesWithParent
	unsafePayloads *PayloadsQueue // queue of unsafe payloads, ordered by ascending block number, may have gaps and duplicates

	// Tracks which L2 blocks where last derived from which L1 block. At most finalityLookback large.
	finalityData []FinalityData

	engine Engine
	prev   NextAttributesProvider

	origin eth.L1BlockRef   // updated on resets, and whenever we read from the previous stage.
	sysCfg eth.SystemConfig // only used for pipeline resets

	metrics   Metrics
	l1Fetcher L1Fetcher
}

var _ EngineControl = (*EngineQueue)(nil)

// NewEngineQueue creates a new EngineQueue, which should be Reset(origin) before use.
func NewEngineQueue(log log.Logger, cfg *rollup.Config, engine Engine, metrics Metrics, prev NextAttributesProvider, l1Fetcher L1Fetcher) *EngineQueue {
	return &EngineQueue{
		log:            log,
		cfg:            cfg,
		engine:         engine,
		metrics:        metrics,
		finalityData:   make([]FinalityData, 0, finalityLookback),
		unsafePayloads: NewPayloadsQueue(maxUnsafePayloadsMemory, payloadMemSize),
		prev:           prev,
		l1Fetcher:      l1Fetcher,
	}
}

// Origin identifies the L1 chain (incl.) that included and/or produced all the safe L2 blocks.
func (eq *EngineQueue) Origin() eth.L1BlockRef {
	return eq.origin
}

func (eq *EngineQueue) SystemConfig() eth.SystemConfig {
	return eq.sysCfg
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

func (eq *EngineQueue) Finalize(l1Origin eth.L1BlockRef) {
	if l1Origin.Number < eq.finalizedL1.Number {
		eq.log.Error("ignoring old L1 finalized block signal! Is the L1 provider corrupted?", "prev_finalized_l1", eq.finalizedL1, "signaled_finalized_l1", l1Origin)
		return
	}

	// remember the L1 finalization signal
	eq.finalizedL1 = l1Origin

	// Sanity check: we only try to finalize L2 immediately, without fetching additional data,
	// if we are on the same chain as the signal.
	// If we are on a different chain, the signal will be ignored,
	// and tryFinalizeL1Origin() will eventually detect that we are on the wrong chain,
	// if not resetting due to reorg elsewhere already.
	for _, fd := range eq.finalityData {
		if fd.L1Block == l1Origin.ID() {
			eq.tryFinalizeL2()
			return
		}
	}

	eq.log.Info("received L1 finality signal, but missing data for immediate L2 finalization", "prev_finalized_l1", eq.finalizedL1, "signaled_finalized_l1", l1Origin)
}

// FinalizedL1 identifies the L1 chain (incl.) that included and/or produced all the finalized L2 blocks.
// This may return a zeroed ID if no finalization signals have been seen yet.
func (eq *EngineQueue) FinalizedL1() eth.L1BlockRef {
	return eq.finalizedL1
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

func (eq *EngineQueue) Step(ctx context.Context) error {
	if eq.needForkchoiceUpdate {
		return eq.tryUpdateEngine(ctx)
	}
	if eq.safeAttributes != nil {
		return eq.tryNextSafeAttributes(ctx)
	}
	outOfData := false
	newOrigin := eq.prev.Origin()
	// Check if the L2 unsafe head origin is consistent with the new origin
	if err := eq.verifyNewL1Origin(ctx, newOrigin); err != nil {
		return err
	}
	eq.origin = newOrigin
	eq.postProcessSafeL2() // make sure we track the last L2 safe head for every new L1 block
	// try to finalize the L2 blocks we have synced so far (no-op if L1 finality is behind)
	if err := eq.tryFinalizePastL2Blocks(ctx); err != nil {
		return err
	}
	if next, err := eq.prev.NextAttributes(ctx, eq.safeHead); err == io.EOF {
		outOfData = true
	} else if err != nil {
		return err
	} else {
		eq.safeAttributes = &attributesWithParent{
			attributes: next,
			parent:     eq.safeHead,
		}
		eq.log.Debug("Adding next safe attributes", "safe_head", eq.safeHead, "next", next)
		return NotEnoughData
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

// verifyNewL1Origin checks that the L2 unsafe head still has a L1 origin that is on the canonical chain.
// If the unsafe head origin is after the new L1 origin it is assumed to still be canonical.
// The check is only required when moving to a new L1 origin.
func (eq *EngineQueue) verifyNewL1Origin(ctx context.Context, newOrigin eth.L1BlockRef) error {
	if newOrigin == eq.origin {
		return nil
	}
	unsafeOrigin := eq.unsafeHead.L1Origin
	if newOrigin.Number == unsafeOrigin.Number && newOrigin.ID() != unsafeOrigin {
		return NewResetError(fmt.Errorf("l1 origin was inconsistent with l2 unsafe head origin, need reset to resolve: l1 origin: %v; unsafe origin: %v",
			newOrigin.ID(), unsafeOrigin))
	}
	// Avoid requesting an older block by checking against the parent hash
	if newOrigin.Number == unsafeOrigin.Number+1 && newOrigin.ParentHash != unsafeOrigin.Hash {
		return NewResetError(fmt.Errorf("l2 unsafe head origin is no longer canonical, need reset to resolve: canonical hash: %v; unsafe origin hash: %v",
			newOrigin.ParentHash, unsafeOrigin.Hash))
	}
	if newOrigin.Number > unsafeOrigin.Number+1 {
		// If unsafe origin is further behind new origin, check it's still on the canonical chain.
		canonical, err := eq.l1Fetcher.L1BlockRefByNumber(ctx, unsafeOrigin.Number)
		if err != nil {
			return NewTemporaryError(fmt.Errorf("failed to fetch canonical L1 block at slot: %v; err: %w", unsafeOrigin.Number, err))
		}
		if canonical.ID() != unsafeOrigin {
			eq.log.Error("Resetting due to origin mismatch")
			return NewResetError(fmt.Errorf("l2 unsafe head origin is no longer canonical, need reset to resolve: canonical: %v; unsafe origin: %v",
				canonical, unsafeOrigin))
		}
	}
	return nil
}

func (eq *EngineQueue) tryFinalizePastL2Blocks(ctx context.Context) error {
	if eq.finalizedL1 == (eth.L1BlockRef{}) {
		return nil
	}

	// If the L1 is finalized beyond the point we are traversing (e.g. during sync),
	// then we should check if we can finalize this L1 block we are traversing.
	// Otherwise, nothing to act on here, we will finalize later on a new finality signal matching the recent history.
	if eq.finalizedL1.Number < eq.origin.Number {
		return nil
	}

	// If we recently tried finalizing, then don't try again just yet, but traverse more of L1 first.
	if eq.triedFinalizeAt != (eth.L1BlockRef{}) && eq.origin.Number <= eq.triedFinalizeAt.Number+finalityDelay {
		return nil
	}

	eq.log.Info("processing L1 finality information", "l1_finalized", eq.finalizedL1, "l1_origin", eq.origin, "previous", eq.triedFinalizeAt)

	// Sanity check we are indeed on the finalizing chain, and not stuck on something else.
	// We assume that the block-by-number query is consistent with the previously received finalized chain signal
	ref, err := eq.l1Fetcher.L1BlockRefByNumber(ctx, eq.origin.Number)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to check if on finalizing L1 chain: %w", err))
	}
	if ref.Hash != eq.origin.Hash {
		return NewResetError(fmt.Errorf("need to reset, we are on %s, not on the finalizing L1 chain %s (towards %s)", eq.origin, ref, eq.finalizedL1))
	}
	eq.tryFinalizeL2()
	return nil
}

// tryFinalizeL2 traverses the past L1 blocks, checks if any has been finalized,
// and then marks the latest fully derived L2 block from this as finalized,
// or defaults to the current finalized L2 block.
func (eq *EngineQueue) tryFinalizeL2() {
	if eq.finalizedL1 == (eth.L1BlockRef{}) {
		return // if no L1 information is finalized yet, then skip this
	}
	eq.triedFinalizeAt = eq.origin
	// default to keep the same finalized block
	finalizedL2 := eq.finalized
	// go through the latest inclusion data, and find the last L2 block that was derived from a finalized L1 block
	for _, fd := range eq.finalityData {
		if fd.L2Block.Number > finalizedL2.Number && fd.L1Block.Number <= eq.finalizedL1.Number {
			finalizedL2 = fd.L2Block
			eq.needForkchoiceUpdate = true
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
	if len(eq.finalityData) == 0 || eq.finalityData[len(eq.finalityData)-1].L1Block.Number < eq.origin.Number {
		// append entry for new L1 block
		eq.finalityData = append(eq.finalityData, FinalityData{
			L2Block: eq.safeHead,
			L1Block: eq.origin.ID(),
		})
		last := &eq.finalityData[len(eq.finalityData)-1]
		eq.log.Debug("extended finality-data", "last_l1", last.L1Block, "last_l2", last.L2Block)
	} else {
		// if it's a new L2 block that was derived from the same latest L1 block, then just update the entry
		last := &eq.finalityData[len(eq.finalityData)-1]
		if last.L2Block != eq.safeHead { // avoid logging if there are no changes
			last.L2Block = eq.safeHead
			eq.log.Debug("updated finality-data", "last_l1", last.L1Block, "last_l2", last.L2Block)
		}
	}
}

func (eq *EngineQueue) logSyncProgress(reason string) {
	eq.log.Info("Sync progress",
		"reason", reason,
		"l2_finalized", eq.finalized,
		"l2_safe", eq.safeHead,
		"l2_unsafe", eq.unsafeHead,
		"l2_time", eq.unsafeHead.Time,
		"l1_derived", eq.origin,
	)
}

// tryUpdateEngine attempts to update the engine with the current forkchoice state of the rollup node,
// this is a no-op if the nodes already agree on the forkchoice state.
func (eq *EngineQueue) tryUpdateEngine(ctx context.Context) error {
	fc := eth.ForkchoiceState{
		HeadBlockHash:      eq.unsafeHead.Hash,
		SafeBlockHash:      eq.safeHead.Hash,
		FinalizedBlockHash: eq.finalized.Hash,
	}
	_, err := eq.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		var inputErr eth.InputError
		if errors.As(err, &inputErr) {
			switch inputErr.Code {
			case eth.InvalidForkchoiceState:
				return NewResetError(fmt.Errorf("forkchoice update was inconsistent with engine, need reset to resolve: %w", inputErr.Unwrap()))
			default:
				return NewTemporaryError(fmt.Errorf("unexpected error code in forkchoice-updated response: %w", err))
			}
		} else {
			return NewTemporaryError(fmt.Errorf("failed to sync forkchoice with engine: %w", err))
		}
	}
	eq.needForkchoiceUpdate = false
	return nil
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
	if eq.safeAttributes == nil { // sanity check the attributes are there
		return nil
	}
	// validate the safe attributes before processing them. The engine may have completed processing them through other means.
	if eq.safeHead != eq.safeAttributes.parent {
		// Previously the attribute's parent was the safe head. If the safe head advances so safe head's parent is the same as the
		// attribute's parent then we need to cancel the attributes.
		if eq.safeHead.ParentHash == eq.safeAttributes.parent.Hash {
			eq.log.Warn("queued safe attributes are stale, safehead progressed",
				"safe_head", eq.safeHead, "safe_head_parent", eq.safeHead.ParentID(), "attributes_parent", eq.safeAttributes.parent)
			eq.safeAttributes = nil
			return nil
		}
		// If something other than a simple advance occurred, perform a full reset
		return NewResetError(fmt.Errorf("safe head changed to %s with parent %s, conflicting with queued safe attributes on top of %s",
			eq.safeHead, eq.safeHead.ParentID(), eq.safeAttributes.parent))

	}
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
	if err := AttributesMatchBlock(eq.safeAttributes.attributes, eq.safeHead.Hash, payload, eq.log); err != nil {
		eq.log.Warn("L2 reorg: existing unsafe block does not match derived attributes from L1", "err", err, "unsafe", eq.unsafeHead, "safe", eq.safeHead)
		// geth cannot wind back a chain without reorging to a new, previously non-canonical, block
		return eq.forceNextSafeAttributes(ctx)
	}
	ref, err := PayloadToBlockRef(payload, &eq.cfg.Genesis)
	if err != nil {
		return NewResetError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}
	eq.safeHead = ref
	eq.needForkchoiceUpdate = true
	eq.metrics.RecordL2Ref("l2_safe", ref)
	// unsafe head stays the same, we did not reorg the chain.
	eq.safeAttributes = nil
	eq.postProcessSafeL2()
	eq.logSyncProgress("reconciled with L1")

	return nil
}

// forceNextSafeAttributes inserts the provided attributes, reorging away any conflicting unsafe chain.
func (eq *EngineQueue) forceNextSafeAttributes(ctx context.Context) error {
	if eq.safeAttributes == nil {
		return nil
	}
	attrs := eq.safeAttributes.attributes
	errType, err := eq.StartPayload(ctx, eq.safeHead, attrs, true)
	if err == nil {
		_, errType, err = eq.ConfirmPayload(ctx)
	}
	if err != nil {
		switch errType {
		case BlockInsertTemporaryErr:
			// RPC errors are recoverable, we can retry the buffered payload attributes later.
			return NewTemporaryError(fmt.Errorf("temporarily cannot insert new safe block: %w", err))
		case BlockInsertPrestateErr:
			_ = eq.CancelPayload(ctx, true)
			return NewResetError(fmt.Errorf("need reset to resolve pre-state problem: %w", err))
		case BlockInsertPayloadErr:
			_ = eq.CancelPayload(ctx, true)
			eq.log.Warn("could not process payload derived from L1 data, dropping batch", "err", err)
			// Count the number of deposits to see if the tx list is deposit only.
			depositCount := 0
			for _, tx := range attrs.Transactions {
				if len(tx) > 0 && tx[0] == types.DepositTxType {
					depositCount += 1
				}
			}
			// Deposit transaction execution errors are suppressed in the execution engine, but if the
			// block is somehow invalid, there is nothing we can do to recover & we should exit.
			// TODO: Can this be triggered by an empty batch with invalid data (like parent hash or gas limit?)
			if len(attrs.Transactions) == depositCount {
				eq.log.Error("deposit only block was invalid", "parent", eq.safeHead, "err", err)
				return NewCriticalError(fmt.Errorf("failed to process block with only deposit transactions: %w", err))
			}
			// drop the payload without inserting it
			eq.safeAttributes = nil
			// suppress the error b/c we want to retry with the next batch from the batch queue
			// If there is no valid batch the node will eventually force a deposit only block. If
			// the deposit only block fails, this will return the critical error above.
			return nil

		default:
			return NewCriticalError(fmt.Errorf("unknown InsertHeadBlock error type %d: %w", errType, err))
		}
	}
	eq.safeAttributes = nil
	eq.logSyncProgress("processed safe block derived from L1")

	return nil
}

func (eq *EngineQueue) StartPayload(ctx context.Context, parent eth.L2BlockRef, attrs *eth.PayloadAttributes, updateSafe bool) (errType BlockInsertionErrType, err error) {
	if eq.buildingID != (eth.PayloadID{}) {
		eq.log.Warn("did not finish previous block building, starting new building now", "prev_onto", eq.buildingOnto, "prev_payload_id", eq.buildingID, "new_onto", parent)
		// TODO: maybe worth it to force-cancel the old payload ID here.
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      parent.Hash,
		SafeBlockHash:      eq.safeHead.Hash,
		FinalizedBlockHash: eq.finalized.Hash,
	}
	id, errTyp, err := StartPayload(ctx, eq.engine, fc, attrs)
	if err != nil {
		return errTyp, err
	}
	eq.buildingID = id
	eq.buildingSafe = updateSafe
	eq.buildingOnto = parent
	return BlockInsertOK, nil
}

func (eq *EngineQueue) ConfirmPayload(ctx context.Context) (out *eth.ExecutionPayload, errTyp BlockInsertionErrType, err error) {
	if eq.buildingID == (eth.PayloadID{}) {
		return nil, BlockInsertPrestateErr, fmt.Errorf("cannot complete payload building: not currently building a payload")
	}
	if eq.buildingOnto.Hash != eq.unsafeHead.Hash { // E.g. when safe-attributes consolidation fails, it will drop the existing work.
		eq.log.Warn("engine is building block that reorgs previous unsafe head", "onto", eq.buildingOnto, "unsafe", eq.unsafeHead)
	}
	fc := eth.ForkchoiceState{
		HeadBlockHash:      common.Hash{}, // gets overridden
		SafeBlockHash:      eq.safeHead.Hash,
		FinalizedBlockHash: eq.finalized.Hash,
	}
	payload, errTyp, err := ConfirmPayload(ctx, eq.log, eq.engine, fc, eq.buildingID, eq.buildingSafe)
	if err != nil {
		return nil, errTyp, fmt.Errorf("failed to complete building on top of L2 chain %s, id: %s, error (%d): %w", eq.buildingOnto, eq.buildingID, errTyp, err)
	}
	ref, err := PayloadToBlockRef(payload, &eq.cfg.Genesis)
	if err != nil {
		return nil, BlockInsertPayloadErr, NewResetError(fmt.Errorf("failed to decode L2 block ref from payload: %w", err))
	}

	eq.unsafeHead = ref
	eq.metrics.RecordL2Ref("l2_unsafe", ref)

	if eq.buildingSafe {
		eq.safeHead = ref
		eq.postProcessSafeL2()
		eq.metrics.RecordL2Ref("l2_safe", ref)
	}
	eq.resetBuildingState()
	return payload, BlockInsertOK, nil
}

func (eq *EngineQueue) CancelPayload(ctx context.Context, force bool) error {
	if eq.buildingID == (eth.PayloadID{}) { // only cancel if there is something to cancel.
		return nil
	}
	// the building job gets wrapped up as soon as the payload is retrieved, there's no explicit cancel in the Engine API
	eq.log.Error("cancelling old block sealing job", "payload", eq.buildingID)
	_, err := eq.engine.GetPayload(ctx, eq.buildingID)
	if err != nil {
		eq.log.Error("failed to cancel block building job", "payload", eq.buildingID, "err", err)
		if !force {
			return err
		}
	}
	eq.resetBuildingState()
	return nil
}

func (eq *EngineQueue) BuildingPayload() (onto eth.L2BlockRef, id eth.PayloadID, safe bool) {
	return eq.buildingOnto, eq.buildingID, eq.buildingSafe
}

func (eq *EngineQueue) resetBuildingState() {
	eq.buildingID = eth.PayloadID{}
	eq.buildingOnto = eth.L2BlockRef{}
	eq.buildingSafe = false
}

// ResetStep Walks the L2 chain backwards until it finds an L2 block whose L1 origin is canonical.
// The unsafe head is set to the head of the L2 chain, unless the existing safe head is not canonical.
func (eq *EngineQueue) Reset(ctx context.Context, _ eth.L1BlockRef, _ eth.SystemConfig) error {
	result, err := sync.FindL2Heads(ctx, eq.cfg, eq.l1Fetcher, eq.engine, eq.log)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find the L2 Heads to start from: %w", err))
	}
	finalized, safe, unsafe := result.Finalized, result.Safe, result.Unsafe
	l1Origin, err := eq.l1Fetcher.L1BlockRefByHash(ctx, safe.L1Origin.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch the new L1 progress: origin: %v; err: %w", safe.L1Origin, err))
	}
	if safe.Time < l1Origin.Time {
		return NewResetError(fmt.Errorf("cannot reset block derivation to start at L2 block %s with time %d older than its L1 origin %s with time %d, time invariant is broken",
			safe, safe.Time, l1Origin, l1Origin.Time))
	}

	// Walk back L2 chain to find the L1 origin that is old enough to start buffering channel data from.
	pipelineL2 := safe
	for {
		afterL2Genesis := pipelineL2.Number > eq.cfg.Genesis.L2.Number
		afterL1Genesis := pipelineL2.L1Origin.Number > eq.cfg.Genesis.L1.Number
		afterChannelTimeout := pipelineL2.L1Origin.Number+eq.cfg.ChannelTimeout > l1Origin.Number
		if afterL2Genesis && afterL1Genesis && afterChannelTimeout {
			parent, err := eq.engine.L2BlockRefByHash(ctx, pipelineL2.ParentHash)
			if err != nil {
				return NewResetError(fmt.Errorf("failed to fetch L2 parent block %s", pipelineL2.ParentID()))
			}
			pipelineL2 = parent
		} else {
			break
		}
	}
	pipelineOrigin, err := eq.l1Fetcher.L1BlockRefByHash(ctx, pipelineL2.L1Origin.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch the new L1 progress: origin: %s; err: %w", pipelineL2.L1Origin, err))
	}
	l1Cfg, err := eq.engine.SystemConfigByL2Hash(ctx, pipelineL2.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch L1 config of L2 block %s: %w", pipelineL2.ID(), err))
	}
	eq.log.Debug("Reset engine queue", "safeHead", safe, "unsafe", unsafe, "safe_timestamp", safe.Time, "unsafe_timestamp", unsafe.Time, "l1Origin", l1Origin)
	eq.unsafeHead = unsafe
	eq.safeHead = safe
	eq.safeAttributes = nil
	eq.finalized = finalized
	eq.resetBuildingState()
	eq.needForkchoiceUpdate = true
	eq.finalityData = eq.finalityData[:0]
	// note: finalizedL1 and triedFinalizeAt do not reset, since these do not change between reorgs.
	// note: we do not clear the unsafe payloads queue; if the payloads are not applicable anymore the parent hash checks will clear out the old payloads.
	eq.origin = pipelineOrigin
	eq.sysCfg = l1Cfg
	eq.metrics.RecordL2Ref("l2_finalized", finalized)
	eq.metrics.RecordL2Ref("l2_safe", safe)
	eq.metrics.RecordL2Ref("l2_unsafe", unsafe)
	eq.logSyncProgress("reset derivation work")
	return io.EOF
}

// UnsafeL2SyncTarget retrieves the first queued-up L2 unsafe payload, or a zeroed reference if there is none.
func (eq *EngineQueue) UnsafeL2SyncTarget() eth.L2BlockRef {
	if first := eq.unsafePayloads.Peek(); first != nil {
		ref, err := PayloadToBlockRef(first, &eq.cfg.Genesis)
		if err != nil {
			return eth.L2BlockRef{}
		}
		return ref
	} else {
		return eth.L2BlockRef{}
	}
}
