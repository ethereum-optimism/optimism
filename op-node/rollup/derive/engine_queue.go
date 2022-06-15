package derive

import (
	"context"
	"fmt"
	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"io"
	"math/big"
	"time"
)

type Engine interface {
	GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error)
	ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error)
	NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error)
	PayloadByHash(context.Context, common.Hash) (*eth.ExecutionPayload, error)
	PayloadByNumber(context.Context, *big.Int) (*eth.ExecutionPayload, error)
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

	safeAttributes []*eth.PayloadAttributes
	unsafePayloads []*eth.ExecutionPayload

	engine Engine
}

// NewEngineQueue creates a new EngineQueue, which should be Reset(origin) before use.
func NewEngineQueue(log log.Logger, cfg *rollup.Config, engine Engine) *EngineQueue {
	return &EngineQueue{log: log, cfg: cfg, engine: engine}
}

func (eq *EngineQueue) Reset(safeHead eth.L2BlockRef, unsafeL2Head eth.L2BlockRef) {
	eq.safeHead = safeHead
	eq.unsafeHead = unsafeL2Head
}

func (eq *EngineQueue) AddUnsafePayload(payload *eth.ExecutionPayload) {
	if len(eq.unsafePayloads) > maxUnsafePayloads {
		return // don't DoS ourselves by buffering too many unsafe payloads
	}
	eq.unsafePayloads = append(eq.unsafePayloads)
}

func (eq *EngineQueue) AddSafeAttributes(attributes *eth.PayloadAttributes) {
	eq.safeAttributes = append(eq.safeAttributes)
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
	if len(eq.safeAttributes) == 0 {
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

		payload, err := eq.engine.PayloadByNumber(ctx, new(big.Int).SetUint64(eq.safeHead.Number+1))
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
