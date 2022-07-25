package derive

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Engine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayload, error)
	L2BlockRefHead(ctx context.Context) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

// Max number of unsafe payloads that may be queued up for execution
const maxUnsafePayloads = 50

// EngineQueue queues up payload attributes to consolidate or process with the provided Engine
type EngineQueue struct {
	log log.Logger
	cfg *rollup.Config

	finalized  eth.L2BlockRef
	safeHead   eth.L2BlockRef
	unsafeHead eth.L2BlockRef

	toFinalize eth.BlockID

	progress Progress

	safeAttributes []*eth.PayloadAttributes
	unsafePayloads []*eth.ExecutionPayload

	engine Engine
}

var _ AttributesQueueOutput = (*EngineQueue)(nil)

// NewEngineQueue creates a new EngineQueue, which should be Reset(origin) before use.
func NewEngineQueue(log log.Logger, cfg *rollup.Config, engine Engine) *EngineQueue {
	return &EngineQueue{log: log, cfg: cfg, engine: engine}
}

func (eq *EngineQueue) Progress() Progress {
	return eq.progress
}

func (eq *EngineQueue) SetUnsafeHead(head eth.L2BlockRef) {
	eq.unsafeHead = head
}

func (eq *EngineQueue) AddUnsafePayload(payload *eth.ExecutionPayload) {
	if len(eq.unsafePayloads) > maxUnsafePayloads {
		eq.log.Debug("Refusing to add unsafe payload", "hash", payload.BlockHash, "number", uint64(payload.BlockNumber))
		return // don't DoS ourselves by buffering too many unsafe payloads
	}
	eq.log.Trace("Adding unsafe payload", "hash", payload.BlockHash, "number", uint64(payload.BlockNumber), "timestamp", uint64(payload.Timestamp))
	eq.unsafePayloads = append(eq.unsafePayloads, payload)
}

func (eq *EngineQueue) AddSafeAttributes(attributes *eth.PayloadAttributes) {
	eq.log.Trace("Adding next safe attributes", "timestamp", attributes.Timestamp)
	eq.safeAttributes = append(eq.safeAttributes, attributes)
}

func (eq *EngineQueue) Finalize(l1Origin eth.BlockID) {
	eq.toFinalize = l1Origin
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

func (eq *EngineQueue) Step(ctx context.Context, outer Progress) error {
	if changed, err := eq.progress.Update(outer); err != nil || changed {
		return err
	}

	// TODO: check if engine unsafehead/safehead/finalized data match, return error and reset pipeline if not.
	// maybe better to do in the driver instead.

	//  TODO: implement finalization
	//if eq.finalized.ID() != eq.toFinalize {
	//	return eq.tryFinalize(ctx)
	//}
	if len(eq.safeAttributes) > 0 {
		return eq.tryNextSafeAttributes(ctx)
	}
	if len(eq.unsafePayloads) > 0 {
		return eq.tryNextUnsafePayload(ctx)
	}
	return io.EOF
}

//  TODO: implement finalization
//func (eq *EngineQueue) tryFinalize(ctx context.Context) error {
//	// find last l2 block ref that references the toFinalize origin, and is lower or equal to the safehead
//	var finalizedL2 eth.L2BlockRef
//	eq.finalized = finalizedL2
//	return nil
//}

func (eq *EngineQueue) tryNextUnsafePayload(ctx context.Context) error {
	first := eq.unsafePayloads[0]

	if uint64(first.BlockNumber) <= eq.safeHead.Number {
		eq.log.Info("skipping unsafe payload, since it is older than safe head", "safe", eq.safeHead.ID(), "unsafe", first.ID(), "payload", first.ID())
		eq.unsafePayloads = eq.unsafePayloads[1:]
		return nil
	}

	// TODO: once we support snap-sync we can remove this condition, and handle the "SYNCING" status of the execution engine.
	if first.ParentHash != eq.unsafeHead.Hash {
		eq.log.Info("skipping unsafe payload, since it does not build onto the existing unsafe chain", "safe", eq.safeHead.ID(), "unsafe", first.ID(), "payload", first.ID())
		eq.unsafePayloads = eq.unsafePayloads[1:]
		return nil
	}

	ref, err := PayloadToBlockRef(first, &eq.cfg.Genesis)
	if err != nil {
		eq.log.Error("failed to decode L2 block ref from payload", "err", err)
		eq.unsafePayloads = eq.unsafePayloads[1:]
		return nil
	}

	// Note: the parent hash does not have to equal the existing unsafe head,
	// the unsafe part of the chain may reorg freely without resetting the derivation pipeline.

	// prepare for processing the unsafe payload
	fc := eth.ForkchoiceState{
		HeadBlockHash:      first.ParentHash,
		SafeBlockHash:      eq.safeHead.Hash, // this should guarantee we do not reorg past the safe head
		FinalizedBlockHash: eq.finalized.Hash,
	}
	fcRes, err := eq.engine.ForkchoiceUpdate(ctx, &fc, nil)
	if err != nil {
		eq.log.Error("failed to update forkchoice to prepare for new unsafe payload", "err", err)
		return nil // we can try again later
	}
	if fcRes.PayloadStatus.Status != eth.ExecutionValid {
		eq.log.Error("cannot prepare unsafe chain for new payload", "new", first.ID(), "parent", first.ParentID(), "err", eth.ForkchoiceUpdateErr(fcRes.PayloadStatus))
		eq.unsafePayloads = eq.unsafePayloads[1:]
		return nil
	}
	status, err := eq.engine.NewPayload(ctx, first)
	if err != nil {
		eq.log.Error("failed to update insert payload", "err", err)
		return nil // we can try again later
	}
	if status.Status != eth.ExecutionValid {
		eq.log.Error("cannot process unsafe payload", "new", first.ID(), "parent", first.ParentID(), "err", eth.ForkchoiceUpdateErr(fcRes.PayloadStatus))
		eq.unsafePayloads = eq.unsafePayloads[1:]
		return nil
	}
	eq.unsafeHead = ref
	eq.unsafePayloads = eq.unsafePayloads[1:]
	eq.log.Trace("Executed unsafe payload", "hash", ref.Hash, "number", ref.Number, "timestamp", ref.Time, "l1Origin", ref.L1Origin)

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
		eq.log.Error("failed to get existing unsafe payload to compare against derived attributes from L1", "err", err)
		return nil
	}
	if err := AttributesMatchBlock(eq.safeAttributes[0], eq.safeHead.Hash, payload); err != nil {
		eq.log.Warn("L2 reorg: existing unsafe block does not match derived attributes from L1", "err", err)
		// geth cannot wind back a chain without reorging to a new, previously non-canonical, block
		return eq.forceNextSafeAttributes(ctx)
	}
	ref, err := PayloadToBlockRef(payload, &eq.cfg.Genesis)
	if err != nil {
		eq.log.Error("failed to decode L2 block ref from payload", "err", err)
		return nil
	}
	eq.safeHead = ref
	// unsafe head stays the same, we did not reorg the chain.
	eq.safeAttributes = eq.safeAttributes[1:]
	eq.log.Trace("Reconciled safe payload", "hash", ref.Hash, "number", ref.Number, "timestamp", ref.Time, "l1Origin", ref.L1Origin)

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
	payload, rpcErr, payloadErr := InsertHeadBlock(ctx, eq.log, eq.engine, fc, eq.safeAttributes[0], true)
	if rpcErr != nil {
		// RPC errors are recoverable, we can retry the buffered payload attributes later.
		eq.log.Error("failed to insert new block", "err", rpcErr)
		return nil
	}
	if payloadErr != nil {
		// invalid payloads are dropped, we move on to the next attributes
		eq.log.Warn("could not derive valid payload from L1 data", "err", payloadErr)
		eq.safeAttributes = eq.safeAttributes[1:]
		return nil
	}
	ref, err := PayloadToBlockRef(payload, &eq.cfg.Genesis)
	if err != nil {
		eq.log.Error("failed to decode L2 block ref from payload", "err", err)
		return nil
	}
	eq.safeHead = ref
	eq.unsafeHead = ref
	eq.safeAttributes = eq.safeAttributes[1:]
	eq.log.Trace("Inserted safe block", "hash", ref.Hash, "number", ref.Number, "timestamp", ref.Time, "l1Origin", ref.L1Origin)

	return nil
}

// ResetStep Walks the L2 chain backwards until it finds an L2 block whose L1 origin is canonical.
// The unsafe head is set to the head of the L2 chain, unless the existing safe head is not canonical.
func (eq *EngineQueue) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {

	l2Head, err := eq.engine.L2BlockRefHead(ctx)
	if err != nil {
		eq.log.Error("failed to find the L2 Head block", "err", err)
		return nil
	}
	unsafe, safe, err := sync.FindL2Heads(ctx, l2Head, eq.cfg.SeqWindowSize, l1Fetcher, eq.engine, &eq.cfg.Genesis)
	if err != nil {
		eq.log.Error("failed to find the L2 Heads to start from", "err", err)
		return nil
	}
	l1Origin, err := l1Fetcher.L1BlockRefByHash(ctx, safe.L1Origin.Hash)
	if err != nil {
		eq.log.Error("failed to fetch the new L1 progress", "err", err, "origin", safe.L1Origin)
		return nil
	}
	if safe.Time < l1Origin.Time {
		return fmt.Errorf("cannot reset block derivation to start at L2 block %s with time %d older than its L1 origin %s with time %d, time invariant is broken",
			safe, safe.Time, l1Origin, l1Origin.Time)
	}
	eq.unsafeHead = unsafe
	eq.safeHead = safe
	eq.progress = Progress{
		Origin: l1Origin,
		Closed: false,
	}
	return io.EOF

}
