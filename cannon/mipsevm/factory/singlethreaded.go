package factory

import (
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum/go-ethereum/log"
)

type SingleThreadedFactory struct {
	state *singlethreaded.State
}

var _ VMFactory = (*SingleThreadedFactory)(nil)

func (s *SingleThreadedFactory) CreateVM(logger log.Logger, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) mipsevm.FPVM {
	logger.Info("Using cannon VM")
	return singlethreaded.NewInstrumentedState(s.state, po, stdOut, stdErr)
}

func (s *SingleThreadedFactory) State() mipsevm.FPVMState {
	return s.state
}
