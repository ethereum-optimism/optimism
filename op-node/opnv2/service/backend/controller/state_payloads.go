package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type PayloadsState struct {
	min   eth.BlockID
	max   eth.BlockID
	count uint64

	chainIDState
}

func NewPayloadsState(rootCtx context.Context, emitter event.Emitter, chainID eth.ChainID) *PayloadsState {
	out := new(PayloadsState)
	// TODO
	return out
}
