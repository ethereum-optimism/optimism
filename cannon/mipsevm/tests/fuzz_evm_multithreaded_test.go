package tests

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func FuzzStateSyscallCloneMT(f *testing.F) {
	v := GetMultiThreadedTestCase(f)
	f.Fuzz(func(t *testing.T, nextThreadId, stackPtr uint32, seed int64) {
		goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(seed))
		state := goVm.GetState()
		mtState, ok := state.(*multithreaded.State)
		if !ok {
			require.Fail(t, "Failed to cast FPVMState to multithreaded State type")
		}

		mtState.NextThreadId = nextThreadId
		state.GetRegistersRef()[2] = exec.SysClone
		state.GetRegistersRef()[4] = exec.ValidCloneFlags
		state.GetRegistersRef()[5] = stackPtr
		state.GetMemory().SetMemory(state.GetPC(), syscallInsn)
		step := state.GetStep()

		expected := testutil.CreateExpectedState(state)
		expected.Step += 1
		expected.PC = state.GetCpu().NextPC
		expected.NextPC = state.GetCpu().NextPC + 4
		expected.Registers[2] = 0
		expected.Registers[7] = 0
		expected.Registers[29] = stackPtr

		stepWitness, err := goVm.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		expected.Validate(t, state)
		// Check mt-specific fields
		newThread := mtState.GetCurrentThread()
		require.Equal(t, nextThreadId, newThread.ThreadId)
		require.Equal(t, nextThreadId+1, mtState.NextThreadId)

		evm := testutil.NewMIPSEVM(v.Contracts)
		evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
		goPost, _ := goVm.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}
