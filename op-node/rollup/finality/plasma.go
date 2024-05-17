package finality

import (
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	plasma "github.com/ethereum-optimism/optimism/op-plasma"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type PlasmaBackend interface {
	// Notify L1 finalized head so plasma finality is always behind L1
	Finalize(ref eth.L1BlockRef)
	// Set the engine finalization signal callback
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
	backend.OnFinalizedHeadSignal(inner.Finalize)

	return &PlasmaFinalizer{
		Finalizer: inner,
		backend:   backend,
	}
}

func (eq *PlasmaFinalizer) Finalize(l1Origin eth.L1BlockRef) {
	eq.backend.Finalize(l1Origin)
}
