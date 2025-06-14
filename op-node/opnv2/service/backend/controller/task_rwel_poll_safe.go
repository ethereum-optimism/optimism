package controller

import (
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func (c *RWELState) maybePollCrossSafe() {
	if c.PollCrossSafe.IsBusy() {
		return
	}
	if c.crossSafe != (eth.BlockRef{}) && !c.pollState.NeedPoll(enginePollDuration, c.state.Now()) {
		return
	}
	c.pollState.RegisterPoll(c.state.Now())
	c.PollCrossSafe.Emit(c.rootCtx, rwel.PollLocalUnsafeRequestEvent{}, nil)
}
