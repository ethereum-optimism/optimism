package controller

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

const enginePollDuration = time.Second * 30

func (c *RWELState) maybePollLocalUnsafe() {
	c.PollLocalUnsafe.AssertNotBusy()
	if c.localUnsafe != (eth.BlockRef{}) && !c.pollState.NeedPoll(enginePollDuration, c.state.Now()) {
		return
	}
	c.pollState.RegisterPoll(c.state.Now())
	c.PollLocalUnsafe.Emit(c.rootCtx, rwel.PollLocalUnsafeRequestEvent{}, c.awaitPollLocalUnsafe)
}

func (c *RWELState) awaitPollLocalUnsafe(ctx context.Context, ev event.Event) {
	switch x := ev.(type) {
	case rwel.LocalUnsafeUpdateEvent:
		c.localUnsafe = x.Ref
	case Err:
	}
	c.PollLocalUnsafe.Reset()
}
