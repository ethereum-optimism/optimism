package versions

import (
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum/go-ethereum/log"
)

type MultiThreadedState struct {
	*multithreaded.State
}

var _ VersionedState = (*MultiThreadedState)(nil)

func (m *MultiThreadedState) CreateVM(logger log.Logger, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) mipsevm.FPVM {
	logger.Info("Using cannon multithreaded VM")
	return multithreaded.NewInstrumentedState(m.State, po, stdOut, stdErr, logger)
}
