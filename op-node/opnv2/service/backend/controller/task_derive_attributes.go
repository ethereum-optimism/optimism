package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/derive2"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// If there is more data to turn into block attributes, do it
func (c *PipelineState) maybeDerive() {
	if c.deriveTask.IsBusy() {
		return
	}
	if !c.more {
		return
	}
	if c.attributes != nil {
		return // need to not be processing attributes already
	}
	if c.lastLocalSafe == (eth.L2BlockRef{}) {
		return // need a starting point for derivation work
	}
	c.deriveTask.Emit(c.rootCtx, derive2.StepRequestEvent{}, c.awaitDerivedAttributes)
}

func (c *PipelineState) awaitDerivedAttributes(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case derive.DeriverMoreEvent:
		c.more = true
	case derive.DerivedAttributesEvent:
		c.attributes = x.Attributes
		c.deriveTask.Reset()
	case derive.ExhaustedL1Event:
		// do not reset nextL1, we may already have prefetched it
		c.more = false
	case rollup.L1TemporaryErrorEvent:
		// TODO L1 backoff
	case rollup.EngineTemporaryErrorEvent:
		// TODO L2 backoff
	default:
	}
}
