package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type RunCfgState struct {
	state   State
	rootCtx context.Context
	chainIDState
	pollState
	backoffState
	TaskStateV2
}

func NewRunCfgState(rootCtx context.Context, emitter event.Emitter, chainID eth.ChainID) *RunCfgState {
	out := new(RunCfgState)
	return out
}
