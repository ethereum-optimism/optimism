package attributes

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2 interface {
	PayloadByNumber(context.Context, uint64) (*eth.ExecutionPayloadEnvelope, error)
}

type AttributesHandler struct {
	log log.Logger
	cfg *rollup.Config

	// when the rollup node shuts down, stop any in-flight sub-processes of the attributes-handler
	ctx context.Context

	l2 L2

	mu sync.Mutex

	emitter rollup.EventEmitter

	attributes *derive.AttributesWithParent
}

func NewAttributesHandler(log log.Logger, cfg *rollup.Config, ctx context.Context, l2 L2, emitter rollup.EventEmitter) *AttributesHandler {
	return &AttributesHandler{
		log:        log,
		cfg:        cfg,
		ctx:        ctx,
		l2:         l2,
		emitter:    emitter,
		attributes: nil,
	}
}

func (eq *AttributesHandler) OnEvent(ev rollup.Event) {
	// Events may be concurrent in the future. Prevent unsafe concurrent modifications to the attributes.
	eq.mu.Lock()
	defer eq.mu.Unlock()

	switch x := ev.(type) {
	case engine.PendingSafeUpdateEvent:
		eq.onPendingSafeUpdate(x)
	case derive.DerivedAttributesEvent:
		eq.attributes = x.Attributes
		eq.emitter.Emit(derive.ConfirmReceivedAttributesEvent{})
		// to make sure we have a pre-state signal to process the attributes from
		eq.emitter.Emit(engine.PendingSafeRequestEvent{})
	case engine.InvalidPayloadAttributesEvent:
		// If the engine signals that attributes are invalid,
		// that should match our last applied attributes, which we should thus drop.
		eq.attributes = nil
		// Time to re-evaluate without attributes.
		// (the pending-safe state will then be forwarded to our source of attributes).
		eq.emitter.Emit(engine.PendingSafeRequestEvent{})
	}
}

// onPendingSafeUpdate applies the queued-up block attributes, if any, on top of the signaled pending state.
// The event is also used to clear the queued-up attributes, when successfully processed.
// On processing failure this may emit a temporary, reset, or critical error like other derivers.
func (eq *AttributesHandler) onPendingSafeUpdate(x engine.PendingSafeUpdateEvent) {
	if x.Unsafe.Number < x.PendingSafe.Number {
		// invalid chain state, reset to try and fix it
		eq.emitter.Emit(rollup.ResetEvent{Err: fmt.Errorf("pending-safe label (%d) may not be ahead of unsafe head label (%d)", x.PendingSafe.Number, x.Unsafe.Number)})
		return
	}

	if eq.attributes == nil {
		// Request new attributes to be generated, only if we don't currently have attributes that have yet to be processed.
		// It is safe to request the pipeline, the attributes-handler is the only user of it,
		// and the pipeline will not generate another set of attributes until the last set is recognized.
		eq.emitter.Emit(derive.PipelineStepEvent{PendingSafe: x.PendingSafe})
		return
	}

	// Drop attributes if they don't apply on top of the pending safe head
	if eq.attributes.Parent.Number != x.PendingSafe.Number {
		eq.log.Warn("dropping stale attributes",
			"pending", x.PendingSafe, "attributes_parent", eq.attributes.Parent)
		eq.attributes = nil
		return
	}

	if eq.attributes.Parent != x.PendingSafe {
		// If the attributes are supposed to follow the pending safe head, but don't build on the exact block,
		// then there's some reorg inconsistency. Either bad attributes, or bad pending safe head.
		// Trigger a reset, and the system can derive attributes on top of the pending safe head.
		// Until the reset is complete we don't clear the attributes state,
		// so we can re-emit the ResetEvent until the reset actually happens.

		eq.emitter.Emit(rollup.ResetEvent{Err: fmt.Errorf("pending safe head changed to %s with parent %s, conflicting with queued safe attributes on top of %s",
			x.PendingSafe, x.PendingSafe.ParentID(), eq.attributes.Parent)})
	} else {
		// if there already exists a block we can just consolidate it
		if x.PendingSafe.Number < x.Unsafe.Number {
			eq.consolidateNextSafeAttributes(eq.attributes, x.PendingSafe)
		} else {
			// append to tip otherwise
			eq.emitter.Emit(engine.ProcessAttributesEvent{Attributes: eq.attributes})
		}
	}
}

// consolidateNextSafeAttributes tries to match the next safe attributes against the existing unsafe chain,
// to avoid extra processing or unnecessary unwinding of the chain.
// However, if the attributes do not match, they will be forced to process the attributes.
func (eq *AttributesHandler) consolidateNextSafeAttributes(attributes *derive.AttributesWithParent, onto eth.L2BlockRef) {
	ctx, cancel := context.WithTimeout(eq.ctx, time.Second*10)
	defer cancel()

	envelope, err := eq.l2.PayloadByNumber(ctx, attributes.Parent.Number+1)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			// engine may have restarted, or inconsistent safe head. We need to reset
			eq.emitter.Emit(rollup.ResetEvent{Err: fmt.Errorf("expected engine was synced and had unsafe block to reconcile, but cannot find the block: %w", err)})
			return
		}
		eq.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: fmt.Errorf("failed to get existing unsafe payload to compare against derived attributes from L1: %w", err)})
		return
	}
	if err := AttributesMatchBlock(eq.cfg, attributes.Attributes, onto.Hash, envelope, eq.log); err != nil {
		eq.log.Warn("L2 reorg: existing unsafe block does not match derived attributes from L1",
			"err", err, "unsafe", envelope.ExecutionPayload.ID(), "pending_safe", onto)

		// geth cannot wind back a chain without reorging to a new, previously non-canonical, block
		eq.emitter.Emit(engine.ProcessAttributesEvent{Attributes: attributes})
		return
	} else {
		ref, err := derive.PayloadToBlockRef(eq.cfg, envelope.ExecutionPayload)
		if err != nil {
			eq.log.Error("Failed to compute block-ref from execution payload")
			return
		}
		eq.emitter.Emit(engine.PromotePendingSafeEvent{
			Ref:         ref,
			Safe:        attributes.IsLastInSpan,
			DerivedFrom: attributes.DerivedFrom,
		})
	}

	// unsafe head stays the same, we did not reorg the chain.
}
