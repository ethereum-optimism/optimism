package rwel

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/attributes"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type TryConsolidateAttributesEvent struct {
	Attributes *derive.AttributesWithParent
}

func (ev TryConsolidateAttributesEvent) String() string {
	return "try-consolidate-attributes"
}

type FailConsolidateAttributesEvent struct {
	Attributes *derive.AttributesWithParent
	Got        *eth.ExecutionPayloadEnvelope
	Ref        eth.L2BlockRef
}

func (ev FailConsolidateAttributesEvent) String() string {
	return "fail-consolidate-attributes"
}

type ConfirmConsolidateAttributesEvent struct {
	Attributes *derive.AttributesWithParent
	Got        *eth.ExecutionPayloadEnvelope
	Ref        eth.L2BlockRef
}

func (ev ConfirmConsolidateAttributesEvent) String() string {
	return "confirm-consolidate-attributes"
}

func (eq *RWEL) onTryConsolidateAttributes(ctx context.Context, ev TryConsolidateAttributesEvent) {
	onto := ev.Attributes.Parent
	attrib := ev.Attributes.Attributes

	rpcCtx, cancel := context.WithTimeout(ctx, buildStartTimeout)
	defer cancel()
	envelope, err := eq.engine.PayloadByNumber(rpcCtx, onto.Number+1)
	if err != nil {
		eq.emitter.Emit(eq.ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to get existing unsafe payload to compare against derived attributes from L1: %w", err),
		})
		return
	}

	if envelope.ExecutionPayload.ParentHash != ev.Attributes.Parent.Hash {
		eq.emitter.Emit(eq.ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("cannot consolidate block (t=%d) when parent-block (%s) does not match expected block (%s)",
				ev.Attributes.Attributes.Timestamp, ev.Attributes.Parent, envelope.ExecutionPayload.ParentID()),
		})
		return
	}

	ref, err := derive.PayloadToBlockRef(eq.cfg, envelope.ExecutionPayload)
	if err != nil {
		eq.log.Error("Failed to compute block-ref from execution payload")
		return
	}

	// TODO: inspect if the envelope is a replacement block
	// If it is, then the engine may already have replaced, while the attributes may get invalidated later.
	// We should emit an event to signal that, the controller can proceed.

	if err := attributes.AttributesMatchBlock(eq.cfg, attrib, onto.Hash, envelope, eq.log); err != nil {
		eq.log.Warn("L2 reorg: existing unsafe block does not match derived attributes from L1",
			"err", err, "unsafe", envelope.ExecutionPayload.ID(), "pending_safe", onto)

		// No need to immediately reorg. Just signal that we failed to consolidate.
		eq.emitter.Emit(ctx, FailConsolidateAttributesEvent{
			Attributes: ev.Attributes,
			Got:        envelope,
			Ref:        ref,
		})

		return
	} else {
		eq.emitter.Emit(ctx, ConfirmConsolidateAttributesEvent{
			Attributes: ev.Attributes,
			Got:        envelope,
			Ref:        ref,
		})
	}
}
