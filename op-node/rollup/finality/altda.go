package finality

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	altda "github.com/ethereum-optimism/optimism/op-alt-da"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/event"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type AltDABackend interface {
	// Finalize notifies the L1 finalized head so AltDA finality is always behind L1.
	Finalize(ref eth.L1BlockRef)
	// OnFinalizedHeadSignal sets the engine finalization signal callback.
	OnFinalizedHeadSignal(f altda.HeadSignalFn)
}

// AltDAFinalizer is a special type of Finalizer, wrapping a regular Finalizer,
// but overriding the finality signal handling:
// it proxies L1 finality signals to the AltDA backend,
// and relies on the backend to then signal when finality is really applicable.
type AltDAFinalizer struct {
	*Finalizer
	backend AltDABackend
}

func NewAltDAFinalizer(ctx context.Context, log log.Logger, cfg *rollup.Config,
	l1Fetcher FinalizerL1Interface,
	backend AltDABackend) *AltDAFinalizer {

	inner := NewFinalizer(ctx, log, cfg, l1Fetcher)

	// In alt-da mode, the finalization signal is proxied through the AltDA manager.
	// Finality signal will come from the DA contract or L1 finality whichever is last.
	// The AltDA module will then call the inner.Finalize function when applicable.
	backend.OnFinalizedHeadSignal(func(ref eth.L1BlockRef) {
		inner.OnEvent(FinalizeL1Event{FinalizedL1: ref})
	})

	return &AltDAFinalizer{
		Finalizer: inner,
		backend:   backend,
	}
}

func (fi *AltDAFinalizer) OnEvent(ev event.Event) bool {
	switch x := ev.(type) {
	case FinalizeL1Event:
		fi.backend.Finalize(x.FinalizedL1)
		return true
	default:
		return fi.Finalizer.OnEvent(ev)
	}
}
