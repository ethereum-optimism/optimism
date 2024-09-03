package multithreaded

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func vmFactory(state *State, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) mipsevm.FPVM {
	return NewInstrumentedState(state, po, stdOut, stdErr, log)
}

func TestInstrumentedState_OpenMips(t *testing.T) {
	// TODO(cp-903): Add mt-specific tests here
	testutil.RunVMTests_OpenMips(t, CreateEmptyState, vmFactory, "clone.bin")
}

func TestInstrumentedState_Hello(t *testing.T) {
	testutil.RunVMTest_Hello(t, CreateInitialState, vmFactory, false)
}

func TestInstrumentedState_Claim(t *testing.T) {
	testutil.RunVMTest_Claim(t, CreateInitialState, vmFactory, false)
}

func TestInstrumentedState_MultithreadedProgram(t *testing.T) {
	state, _ := testutil.LoadELFProgram(t, "../../testdata/example/bin/multithreaded.elf", CreateInitialState, false)
	oracle := testutil.StaticOracle(t, []byte{})

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := NewInstrumentedState(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
	for i := 0; i < 1_000_000; i++ {
		if us.GetState().GetExited() {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}
	t.Logf("Completed in %d steps", state.Step)

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")
	require.Contains(t, "waitgroup result: 42", stdErrBuf.String())
	require.Contains(t, "channels result: 1234", stdErrBuf.String())
	require.Equal(t, "", stdErrBuf.String(), "should not print any errors")
}

func TestInstrumentedState_Alloc(t *testing.T) {
	t.Skip("TODO(client-pod#906): Currently failing - need to debug.")

	state, _ := testutil.LoadELFProgram(t, "../../testdata/example/bin/alloc.elf", CreateInitialState, false)
	const numAllocs = 100 // where each alloc is a 32 MiB chunk
	oracle := testutil.AllocOracle(t, numAllocs)

	// completes in ~870 M steps
	us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr, testutil.CreateLogger())
	for i := 0; i < 20_000_000_000; i++ {
		if us.GetState().GetExited() {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
		if state.Step%10_000_000 == 0 {
			t.Logf("Completed %d steps", state.Step)
		}
	}
	t.Logf("Completed in %d steps", state.Step)
	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")
	require.Less(t, state.Memory.PageCount()*memory.PageSize, 1*1024*1024*1024, "must not allocate more than 1 GiB")
}
