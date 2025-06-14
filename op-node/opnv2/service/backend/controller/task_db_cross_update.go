package controller

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/event"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/backend/superevents"
)

func (v *ChainDBState) maybeCrossSafeUpdate() {
	if v.crossSafeWork.IsBusy() {
		return
	}
	now := v.state.Now()
	if !v.crossSafeWork.IsBackedOff(now) {
		v.crossSafeWork.Emit(v.rootCtx, superevents.UpdateCrossSafeRequestEvent{
			ChainID: v.ChainID(),
		}, v.awaitCrossSafeUpdate)
	}
}

func (v *ChainDBState) awaitCrossSafeUpdate(ctx context.Context, ev event.Event) {
	// TODO handle error/success case
}

func (v *ChainDBState) maybeCrossUnsafeUpdate() {
	if v.crossUnsafeWork.IsBusy() {
		return
	}
	now := v.state.Now()
	if !v.crossUnsafeWork.IsBackedOff(now) {
		v.crossUnsafeWork.Emit(v.rootCtx, superevents.UpdateCrossUnsafeRequestEvent{
			ChainID: v.ChainID(),
		}, v.awaitCrossUnsafeUpdate)
	}
}

func (v *ChainDBState) awaitCrossUnsafeUpdate(ctx context.Context, ev event.Event) {
	// TODO handle error/success case
}
