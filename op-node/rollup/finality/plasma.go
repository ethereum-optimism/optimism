package finality

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type PlasmaBackend interface {
	// Finalize notifies the L1 finalized head so plasma finality is always behind L1.
	Finalize(ref eth.L1BlockRef)
	// OnFinalizedHeadSignal sets the engine finalization signal callback.
	OnFinalizedHeadSignal(f plasma.HeadSignalFn)
}

// PlasmaFinalizer is a special type of Finalizer, wrapping a regular Finalizer,
// but overriding the finality signal handling:
// it proxies L1 finality signals to the plasma backend,
// and relies on the backend to then signal when finality is really applicable.
type PlasmaFinalizer struct {
	*Finalizer
	backend PlasmaBackend
}

func NewPlasmaFinalizer(ctx context.Context, log log.Logger, cfg *rollup.Config,
	l1Fetcher FinalizerL1Interface, emitter rollup.EventEmitter,
	backend PlasmaBackend) *PlasmaFinalizer {

	inner := NewFinalizer(ctx, log, cfg, l1Fetcher, emitter)

	// In alt-da mode, the finalization signal is proxied through the plasma manager.
	// Finality signal will come from the DA contract or L1 finality whichever is last.
	// The plasma module will then call the inner.Finalize function when applicable.
	backend.OnFinalizedHeadSignal(func(ref eth.L1BlockRef) {
		inner.OnEvent(FinalizeL1Event{FinalizedL1: ref})
	})

	return &PlasmaFinalizer{
		Finalizer: inner,
		backend:   backend,
	}
}

func (fi *PlasmaFinalizer) OnEvent(ev rollup.Event) {
	switch x := ev.(type) {
	case FinalizeL1Event:
		fi.backend.Finalize(x.FinalizedL1)
	default:
		fi.Finalizer.OnEvent(ev)
	}
}
