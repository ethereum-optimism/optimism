package controller

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/l1access"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

const l1LatestPollDuration = time.Second * 12

func (c *L1AccessState) maybePollL1Latest() {
	c.pollL1LatestTask.AssertNotBusy()
	now := c.state.Now()
	if c.pollL1LatestPoll.NeedPoll(l1LatestPollDuration, now) {
		c.pollL1LatestPoll.RegisterPoll(now)
		c.pollL1LatestTask.Emit(c.rootCtx, l1access.LatestL1RequestEvent{}, c.awaitL1Latest)
	}
}

func (c *L1AccessState) awaitL1Latest(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case l1access.LatestL1UpdateEvent:
		c.latestL1 = x.LatestL1
		c.pollL1LatestTask.Reset()
	case Err:

	}
}
