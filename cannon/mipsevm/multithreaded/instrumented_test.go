package multithreaded

import (
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
	testutil.RunVMTests_OpenMips(t, CreateEmptyState, vmFactory)
}

func TestInstrumentedState_Hello(t *testing.T) {
	testutil.RunVMTest_Hello(t, CreateInitialState, vmFactory)
}

func TestInstrumentedState_Claim(t *testing.T) {
	testutil.RunVMTest_Claim(t, CreateInitialState, vmFactory)
}

func TestInstrumentedState_Alloc(t *testing.T) {
	t.Skip("TODO(client-pod#906): Currently fails on Single threaded Cannon. Re-enable for the MT FPVM")

	state := testutil.LoadELFProgram(t, "../../example/bin/alloc.elf", CreateInitialState)
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
