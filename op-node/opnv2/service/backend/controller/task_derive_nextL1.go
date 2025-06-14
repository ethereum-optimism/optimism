package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1access"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

// if next L1 source block is needed, then make l1access provide that next L1 block
func (c *PipelineState) maybePrepareNextL1() {
	if c.lastL1Source == (eth.BlockRef{}) {
		return
	}
	if c.nextL1Source != (eth.BlockRef{}) {
		return
	}
	c.nextL1Task.Emit(c.rootCtx, l1access.ByNumberL1RequestEvent{
		Num: c.lastL1Source.Number + 1,
	}, c.awaitNextL1)
}

func (c *PipelineState) awaitNextL1(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case l1access.RetrievedL1BlockEvent:
		c.nextL1Source = x.Ref
		c.nextL1Task.Reset()
	case Err:

	}
}
