package versions

import (
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum/go-ethereum/log"
)

type SingleThreadedState struct {
	*singlethreaded.State
}

var _ VersionedState = (*SingleThreadedState)(nil)

func (s *SingleThreadedState) CreateVM(logger log.Logger, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) mipsevm.FPVM {
	logger.Info("Using cannon VM")
	return singlethreaded.NewInstrumentedState(s.State, po, stdOut, stdErr)
}
