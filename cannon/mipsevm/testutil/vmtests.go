package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

type VMFactory[T mipsevm.FPVMState] func(state T, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, meta *program.Metadata) mipsevm.FPVM
type StateFactory[T mipsevm.FPVMState] func() T

func RunVMTest_Hello[T mipsevm.FPVMState](t *testing.T, initState program.CreateInitialFPVMState[T], vmFactory VMFactory[T]) {
	state, meta := LoadELFProgram(t, ProgramPath("hello"), initState)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := vmFactory(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), CreateLogger(), meta)

	maxSteps := 450_000
	for i := 0; i < maxSteps; i++ {
		if us.GetState().GetExited() {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}

	require.Truef(t, state.GetExited(), "must complete program. reached %d of max %d steps", state.GetStep(), maxSteps)
	require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

	require.Equal(t, "hello world!\n", stdOutBuf.String(), "stdout says hello")
	require.Equal(t, "", stdErrBuf.String(), "stderr silent")
}

func RunVMTest_Claim[T mipsevm.FPVMState](t *testing.T, initState program.CreateInitialFPVMState[T], vmFactory VMFactory[T]) {
	state, meta := LoadELFProgram(t, ProgramPath("claim"), initState)

	oracle, expectedStdOut, expectedStdErr := ClaimTestOracle(t)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := vmFactory(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), CreateLogger(), meta)

	for i := 0; i < 2000_000; i++ {
		if us.GetState().GetExited() {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}

	require.True(t, state.GetExited(), "must complete program")
	require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

	require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
	require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
}
