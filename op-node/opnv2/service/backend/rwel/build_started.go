package rwel

import (
	"context"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type BuildStartedEvent struct {
	Info eth.PayloadInfo

	BuildStarted time.Time

	Parent eth.BlockRef

	// payload is promoted to pending-safe if non-zero
	DerivedFrom eth.L1BlockRef
}

func (ev BuildStartedEvent) String() string {
	return "build-started"
}

func (eq *RWEL) onBuildStarted(ctx context.Context, ev BuildStartedEvent) {
	// If a (pending) safe block, immediately seal the block
	if ev.DerivedFrom != (eth.L1BlockRef{}) {
		eq.emitter.Emit(ctx, BuildSealEvent{
			Info:         ev.Info,
			BuildStarted: ev.BuildStarted,
			DerivedFrom:  ev.DerivedFrom,
		})
	}
}
