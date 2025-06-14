package controller

import (
	"github.com/ethereum-optimism/optimism/op-node/opnv2/service/backend/rwel"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

func (c *RWELState) maybePollFinalized() {
	if c.PollFinalized.IsBusy() {
		return
	}
	if c.finalized != (eth.BlockRef{}) && !c.pollState.NeedPoll(enginePollDuration, c.state.Now()) {
		return
	}
	c.pollState.RegisterPoll(c.state.Now())
	c.PollFinalized.Emit(c.rootCtx, rwel.PollFinalizedRequestEvent{}, nil)
}
