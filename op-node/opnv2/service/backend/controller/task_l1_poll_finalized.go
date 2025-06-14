package controller

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1access"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

const l1FinalizedPollDuration = time.Second * 60

func (c *L1AccessState) maybePollFinalized() {
	c.pollL1FinalizedTask.AssertNotBusy()
	now := c.state.Now()
	if c.finalizedPoll.NeedPoll(l1FinalizedPollDuration, now) {
		c.finalizedPoll.RegisterPoll(now)
		c.pollL1FinalizedTask.Emit(c.rootCtx, l1access.FinalizedL1RequestEvent{}, c.awaitL1Finalized)
	}
}

func (c *L1AccessState) awaitL1Finalized(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case l1access.FinalizedL1UpdateEvent:
		c.finalizedL1 = x.FinalizedL1
		c.pollL1FinalizedTask.Reset()
	case Err:
	}
}
