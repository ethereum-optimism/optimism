package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum/go-ethereum/log"
)

// BuildSuccess bundles the information required by callers to emit
// success events and update chain heads.
type BuildSuccess struct {
	FCEvent       *ForkchoiceUpdateEvent
	Concluding    bool
	DerivedFrom   eth.L1BlockRef
	BuildStarted  time.Time
	InsertStarted time.Time
	Envelope      *eth.ExecutionPayloadEnvelope
	Ref           eth.L2BlockRef
}

// BlockBuilder encapsulates the builder flow and synchronizes access to controller state.
type BlockBuilder struct {
	rollupCfg *rollup.Config
	engine    ExecEngine
	emitter   event.Emitter
	log       log.Logger
}

func NewBlockBuilder(rollupCfg *rollup.Config, engine ExecEngine, emitter event.Emitter, log log.Logger) *BlockBuilder {
	return &BlockBuilder{rollupCfg: rollupCfg, engine: engine, emitter: emitter, log: log}
}

func (b *BlockBuilder) BuildBlock(ctx context.Context, attributes *derive.AttributesWithParent, pending eth.L2BlockRef, safe eth.L2BlockRef, finalized eth.L2BlockRef, hook func(ctx context.Context)) (*BuildSuccess, error) {
	rpcCtx, cancel := context.WithTimeout(ctx, buildStartTimeout)
	defer cancel()

	// pending safe vs parent check under read lock
	if attributes.IsDerived() && pending.Hash != attributes.Parent.Hash {
		b.log.Warn("block-attributes derived from L1 do not build on pending safe head, likely reorg",
			"pending_safe", pending, "attributes_parent", attributes.Parent)
	}

	fcEvent := ForkchoiceUpdateEvent{
		UnsafeL2Head:    attributes.Parent,
		SafeL2Head:      safe,
		FinalizedL2Head: finalized,
	}
	if fcEvent.UnsafeL2Head.Number < fcEvent.FinalizedL2Head.Number {
		err := fmt.Errorf("invalid block-building pre-state, unsafe head %s is behind finalized head %s", fcEvent.UnsafeL2Head, fcEvent.FinalizedL2Head)
		return nil, &InvalidPrestateError{Err: err, FCEvent: &fcEvent}
	}

	fc := eth.ForkchoiceState{
		HeadBlockHash:      fcEvent.UnsafeL2Head.Hash,
		SafeBlockHash:      fcEvent.SafeL2Head.Hash,
		FinalizedBlockHash: fcEvent.FinalizedL2Head.Hash,
	}
	buildStartTime := time.Now()

	// engine ForkchoiceUpdate guarded by exclusive lock
	id, errTyp, err := startPayload(rpcCtx, b.engine, fc, attributes.Attributes)
	if err != nil {
		switch errTyp {
		case BlockInsertTemporaryErr:
			return nil, &BuildTemporaryError{Err: fmt.Errorf("temporarily cannot insert new safe block: %w", err), FCEvent: &fcEvent}
		case BlockInsertPrestateErr:
			return nil, &BuildPrestateError{Err: fmt.Errorf("need reset to resolve pre-state problem: %w", err), FCEvent: &fcEvent}
		case BlockInsertPayloadErr:
			return nil, &BuildInvalidAttributesError{Attributes: attributes, Err: err, FCEvent: &fcEvent}
		default:
			return nil, &BuildCriticalError{Err: fmt.Errorf("unknown error type %d: %w", errTyp, err), FCEvent: &fcEvent}
		}
	}
	// Start succeeded; capture the forkchoice event for the success/error results.
	attachedFC := &fcEvent

	// test hook: allow tests to enqueue transactions just before sealing
	if hook != nil {
		hook(ctx)
	}

	// Seal step with timeout and exclusive engine access
	sealCtx, sealCancel := context.WithTimeout(ctx, buildSealTimeout)
	defer sealCancel()

	envelope, sealErr := b.engine.GetPayload(sealCtx, eth.PayloadInfo{ID: id, Timestamp: uint64(attributes.Attributes.Timestamp)})
	if sealErr != nil {
		return nil, &SealExpiredError{FCEvent: attachedFC, Info: eth.PayloadInfo{ID: id, Timestamp: uint64(attributes.Attributes.Timestamp)}, Err: fmt.Errorf("failed to seal execution payload (ID: %s): %w", id, sealErr), Concluding: attributes.Concluding, DerivedFrom: attributes.DerivedFrom}
	}

	if err := sanityCheckPayload(envelope.ExecutionPayload); err != nil {
		return nil, &SealInvalidError{FCEvent: attachedFC, Info: eth.PayloadInfo{ID: id, Timestamp: uint64(attributes.Attributes.Timestamp)}, Err: fmt.Errorf("failed sanity-check of execution payload contents (ID: %s, blockhash: %s): %w", id, envelope.ExecutionPayload.BlockHash, err), Concluding: attributes.Concluding, DerivedFrom: attributes.DerivedFrom}
	}

	ref, refErr := derive.PayloadToBlockRef(b.rollupCfg, envelope.ExecutionPayload)
	if refErr != nil {
		return nil, &SealInvalidError{FCEvent: attachedFC, Info: eth.PayloadInfo{ID: id, Timestamp: uint64(attributes.Attributes.Timestamp)}, Err: fmt.Errorf("failed to decode L2 block ref from payload: %w", refErr), Concluding: attributes.Concluding, DerivedFrom: attributes.DerivedFrom}
	}

	// Process step with timeout and exclusive engine access
	procCtx, procCancel := context.WithTimeout(ctx, payloadProcessTimeout)
	defer procCancel()
	insertStart := time.Now()

	status, procErr := b.engine.NewPayload(procCtx, envelope.ExecutionPayload, envelope.ParentBeaconBlockRoot)
	if procErr != nil {
		return nil, &BuildTemporaryError{Err: fmt.Errorf("failed to insert execution payload: %w", procErr), FCEvent: attachedFC}
	}
	switch status.Status {
	case eth.ExecutionInvalid, eth.ExecutionInvalidBlockHash:
		if attributes.DerivedFrom != (eth.L1BlockRef{}) && b.rollupCfg.IsHolocene(attributes.DerivedFrom.Time) {
			return nil, &DepositsOnlyRequest{FCEvent: attachedFC, Parent: attributes.Parent.ID(), DerivedFrom: attributes.DerivedFrom}
		}
		return nil, &PayloadInvalidError{FCEvent: attachedFC, Envelope: envelope, Err: eth.NewPayloadErr(envelope.ExecutionPayload, status)}
	case eth.ExecutionValid:
		return &BuildSuccess{FCEvent: attachedFC, Concluding: attributes.Concluding, DerivedFrom: attributes.DerivedFrom, BuildStarted: buildStartTime, InsertStarted: insertStart, Envelope: envelope, Ref: ref}, nil
	default:
		return nil, &BuildTemporaryError{Err: eth.NewPayloadErr(envelope.ExecutionPayload, status), FCEvent: attachedFC}
	}
}
