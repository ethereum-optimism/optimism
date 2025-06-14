package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type L1AccessState struct {
	state ClockState

	rootCtx context.Context

	// latestL1 is the latest L1 block, ignoring the conf-depth.
	// This may be zero if not known yet.
	latestL1 eth.BlockRef

	// confirmedL1 is the latest L1 block that has passed the conf-depth, or is finalized.
	// This may be zero if not known yet, or if temporarily failing to confirm.
	confirmedL1 eth.BlockRef

	// finalizedL1 is the finalized L1 block.
	// This may be zero if not known yet.
	finalizedL1 eth.BlockRef

	pollL1FinalizedTask TaskStateV2
	finalizedPoll       pollState

	pollL1LatestTask TaskStateV2
	pollL1LatestPoll pollState

	// TODO backoff on last known err
}

func NewL1AccessState(rootCtx context.Context, emitter event.Emitter) *L1AccessState {
	out := new(L1AccessState)
	out.rootCtx = rootCtx
	out.pollL1LatestTask.Init(emitter, out.maybePollL1Latest)
	out.pollL1FinalizedTask.Init(emitter, out.maybePollFinalized)
	return out
}
