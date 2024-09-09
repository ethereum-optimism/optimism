package factory

import (
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum/go-ethereum/log"
)

type MultiThreadedFactory struct {
	state *multithreaded.State
}

var _ VMFactory = (*MultiThreadedFactory)(nil)

func (m *MultiThreadedFactory) CreateVM(logger log.Logger, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) mipsevm.FPVM {
	logger.Info("Using cannon multithreaded VM")
	return multithreaded.NewInstrumentedState(m.state, po, stdOut, stdErr, logger)
}

func (m *MultiThreadedFactory) State() mipsevm.FPVMState {
	return m.state
}
