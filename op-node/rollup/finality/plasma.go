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

func NewPlasmaFinalizer(log log.Logger, cfg *rollup.Config,
	l1Fetcher FinalizerL1Interface, ec FinalizerEngine,
	backend PlasmaBackend) *PlasmaFinalizer {

	inner := NewFinalizer(log, cfg, l1Fetcher, ec)

	// In plasma mode, the finalization signal is proxied through the plasma manager.
	// Finality signal will come from the DA contract or L1 finality whichever is last.
	// The plasma module will then call the inner.Finalize function when applicable.
	backend.OnFinalizedHeadSignal(func(ref eth.L1BlockRef) {
		inner.Finalize(context.Background(), ref) // plasma backend context passing can be improved
	})

	return &PlasmaFinalizer{
		Finalizer: inner,
		backend:   backend,
	}
}

func (fi *PlasmaFinalizer) Finalize(ctx context.Context, l1Origin eth.L1BlockRef) {
	fi.backend.Finalize(l1Origin)
}
