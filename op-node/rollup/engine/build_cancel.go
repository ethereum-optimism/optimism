package engine

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type BuildCancelEvent struct {
	Info  eth.PayloadInfo
	Force bool
}

func (ev BuildCancelEvent) String() string {
	return "build-cancel"
}

func (eq *EngDeriver) onBuildCancel(ev BuildCancelEvent) {
	ctx, cancel := context.WithTimeout(eq.ctx, buildCancelTimeout)
	defer cancel()
	// the building job gets wrapped up as soon as the payload is retrieved, there's no explicit cancel in the Engine API
	eq.log.Warn("cancelling old block building job", "info", ev.Info)
	_, err := eq.ec.engine.GetPayload(ctx, ev.Info)
	if err != nil {
		if x, ok := err.(eth.InputError); ok && x.Code == eth.UnknownPayload { //nolint:all
			return // if unknown, then it did not need to be cancelled anymore.
		}
		eq.log.Error("failed to cancel block building job", "info", ev.Info, "err", err)
		if !ev.Force {
			eq.emitter.Emit(rollup.EngineTemporaryErrorEvent{Err: err})
		}
	}
}
