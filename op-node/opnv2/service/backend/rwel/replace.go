package rwel

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-node/rollup/interop/indexing"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	supervisortypes "github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

type InvalidateBlockRequestEvent struct {
	Invalidated supervisortypes.BlockSeal
}

func (ev InvalidateBlockRequestEvent) String() string {
	return "invalidate-block-request"
}

// BuildReplacementBlockEvent is emitted when a block needs to be invalidated, and a replacement is needed.
type BuildReplacementBlockEvent struct {
	Invalidated eth.BlockRef
	Attributes  *derive.AttributesWithParent
}

func (ev BuildReplacementBlockEvent) String() string {
	return "build-replacement-block"
}

func (m *RWEL) onInvalidateBlockRequest(ctx context.Context, ev InvalidateBlockRequestEvent) {
	m.log.Info("Invalidating block", "block", ev.Invalidated)

	// Fetch the block we invalidate, so we can re-use the attributes that stay.
	block, err := m.engine.PayloadByHash(ctx, ev.Invalidated.Hash)
	if err != nil { // cannot invalidate if it wasn't there.
		m.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to get block: %w", err),
		})
		return
	}

	parentRef, err := m.engine.L2BlockRefByHash(ctx, block.ExecutionPayload.ParentHash)
	if err != nil {
		m.emitter.Emit(ctx, rollup.EngineTemporaryErrorEvent{
			Err: fmt.Errorf("failed to get parent of invalidated block: %w", err),
		})
		return
	}

	ref := block.ExecutionPayload.BlockRef()

	// Create the attributes that we build the replacement block with.
	attributes := indexing.AttributesToReplaceInvalidBlock(block)
	annotated := &derive.AttributesWithParent{
		Attributes:  attributes,
		Parent:      parentRef,
		DerivedFrom: engine.ReplaceBlockSource,
	}

	m.emitter.Emit(ctx, BuildReplacementBlockEvent{
		Invalidated: ref, Attributes: annotated})
}
