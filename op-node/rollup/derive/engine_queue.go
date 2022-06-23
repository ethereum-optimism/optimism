package derive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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
	UnsafeBlockIDs(ctx context.Context, safeHead eth.BlockID, max uint64) ([]eth.BlockID, error)
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

	resetting bool

	toFinalize eth.BlockID

	safeAttributes []*eth.PayloadAttributes
	unsafePayloads []*eth.ExecutionPayload

	engine Engine
}

var _ BatchQueueOutput = (*EngineQueue)(nil)

// NewEngineQueue creates a new EngineQueue, which should be Reset(origin) before use.
func NewEngineQueue(log log.Logger, cfg *rollup.Config, engine Engine) *EngineQueue {
	return &EngineQueue{log: log, cfg: cfg, engine: engine}
}

func (eq *EngineQueue) SetUnsafeHead(head eth.L2BlockRef) {
	eq.unsafeHead = head
}

func (eq *EngineQueue) AddUnsafePayload(payload *eth.ExecutionPayload) {
	if len(eq.unsafePayloads) > maxUnsafePayloads {
		return // don't DoS ourselves by buffering too many unsafe payloads
	}
	eq.unsafePayloads = append(eq.unsafePayloads, payload)
}

func (eq *EngineQueue) AddSafeAttributes(attributes *eth.PayloadAttributes) {
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

func (eq *EngineQueue) Step(ctx context.Context) error {
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
		eq.log.Error("skipping unsafe payload, since it is older than safe head", "safe", eq.safeHead.ID(), "unsafe", first.ID())
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
	return nil
}

func (eq *EngineQueue) tryNextSafeAttributes(ctx context.Context) error {
	first := eq.safeAttributes[0]
	if eq.safeHead.Number < eq.unsafeHead.Number {
		ctx, cancel := context.WithTimeout(ctx, time.Second*10)
		defer cancel()

		payload, err := eq.engine.PayloadByNumber(ctx, eq.safeHead.Number+1)
		if err != nil {
			eq.log.Error("failed to get existing unsafe payload to compare against derived attributes from L1", "err", err)
			return nil
		}
		if err := AttributesMatchBlock(first, eq.safeHead.Hash, payload); err != nil {
			eq.log.Warn("existing unsafe block does not match derived attributes from L1", "err", err)
			fc := eth.ForkchoiceState{
				HeadBlockHash:      eq.safeHead.Hash, // undo the unsafe chain when the safe chain does not match.
				SafeBlockHash:      eq.safeHead.Hash,
				FinalizedBlockHash: eq.finalized.Hash,
			}
			status, err := eq.engine.ForkchoiceUpdate(ctx, &fc, nil)
			if err != nil {
				eq.log.Error("failed to update forkchoice to revert mismatching L2 block", "err", err)
				return nil // we will find the mismatch again, and retry the forkchoice update
			}
			if status.PayloadStatus.Status != eth.ExecutionValid {
				// deep reorg, if we can't revert the unsafe chain as an extension of known safe chain, then we have to reset the derivation pipeline
				return fmt.Errorf("cannot revert unsafe chain to safe head: %w", eth.ForkchoiceUpdateErr(status.PayloadStatus))
			}
			eq.unsafeHead = eq.safeHead
			return nil
		}
		ref, err := PayloadToBlockRef(payload, &eq.cfg.Genesis)
		if err != nil {
			eq.log.Error("failed to decode L2 block ref from payload", "err", err)
			return nil
		}
		eq.safeHead = ref
		// unsafe head stays the same, we did not reorg the chain.
		eq.safeAttributes = eq.safeAttributes[1:]
		return nil
	} else if eq.safeHead.Number == eq.unsafeHead.Number {
		fc := eth.ForkchoiceState{
			HeadBlockHash:      eq.unsafeHead.Hash,
			SafeBlockHash:      eq.safeHead.Hash,
			FinalizedBlockHash: eq.finalized.Hash,
		}
		payload, rpcErr, payloadErr := InsertHeadBlock(ctx, eq.log, eq.engine, fc, first, true)
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
		return nil
	} else {
		// For some reason the unsafe head is behind the safe head. Log it, and correct it.
		eq.log.Error("invalid sync state, unsafe head is behind safe head", "unsafe", eq.unsafeHead, "safe", eq.safeHead)
		eq.unsafeHead = eq.safeHead
		return nil
	}
}

// ResetStep Walks the L2 chain backwards until it finds an L2 block whose L1 origin is canonical.
// The unsafe head is set to the head of the L2 chain, unless the existing safe head is not canonical.
func (eq *EngineQueue) ResetStep(ctx context.Context, l1Fetcher L1Fetcher) error {
	if !eq.resetting {
		eq.resetting = true

		head, err := eq.engine.L2BlockRefHead(ctx)
		if err != nil {
			eq.log.Error("failed to get L2 engine head to start finding reset point from", "err", err)
			return nil
		}
		eq.unsafeHead = head

		// TODO: this should be different for safe head.
		// We can't trust the origin data of the unsafe chain.
		// We should query the engine for its current safe-head.
		eq.safeHead = head
		return nil
	}

	// check if the block origin is canonical
	if canonicalRef, err := l1Fetcher.L1BlockRefByNumber(ctx, eq.safeHead.L1Origin.Number); errors.Is(err, ethereum.NotFound) {
		// if our view of the l1 chain is lagging behind, we may get this error
		eq.log.Warn("engine safe head is ahead of L1 view", "block", eq.safeHead, "origin", eq.safeHead.L1Origin)
	} else if err != nil {
		eq.log.Warn("failed to get L1 block ref to check if origin of l2 block is canonical", "err", err, "num", eq.safeHead.L1Origin.Number)
	} else {
		// if we find the safe head, then we found the canon chain
		if canonicalRef.Hash == eq.safeHead.L1Origin.Hash {
			eq.resetting = false
			// if the unsafe head was broken, then restore it to start from the safe head
			if eq.unsafeHead == (eth.L2BlockRef{}) {
				eq.unsafeHead = eq.safeHead
			}
			return io.EOF
		} else {
			// if the safe head is not canonical, then the unsafe head will not be either
			eq.unsafeHead = eth.L2BlockRef{}
		}
	}

	// Don't walk past genesis. If we were at the L2 genesis, but could not find its L1 origin,
	// the L2 chain is building on the wrong L1 branch.
	if eq.safeHead.Hash == eq.cfg.Genesis.L2.Hash || eq.safeHead.Number == eq.cfg.Genesis.L2.Number {
		return fmt.Errorf("the L2 engine is coupled to unrecognized L1 chain: %v", eq.cfg.Genesis)
	}

	// Pull L2 parent for next iteration
	block, err := eq.engine.L2BlockRefByHash(ctx, eq.safeHead.ParentHash)
	if err != nil {
		eq.log.Error("failed to fetch L2 block by hash during reset", "parent", eq.safeHead.ParentHash, "err", err)
		return nil
	}
	eq.safeHead = block
	return nil
}
