package singlethreaded

import (
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm32"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm32/testutil"
)

func vmFactory(state *State, po mipsevm32.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm32.FPVM {
	return NewInstrumentedState(state, po, stdOut, stdErr, nil)
}

func TestInstrumentedState_OpenMips(t *testing.T) {
	testutil.RunVMTests_OpenMips(t, CreateEmptyState, vmFactory)
}

func TestInstrumentedState_Hello(t *testing.T) {
	testutil.RunVMTest_Hello(t, CreateInitialState, vmFactory, true)
}

func TestInstrumentedState_Claim(t *testing.T) {
	testutil.RunVMTest_Claim(t, CreateInitialState, vmFactory, true)
}
