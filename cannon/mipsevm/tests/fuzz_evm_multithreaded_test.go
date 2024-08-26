package tests

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	mttestutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func FuzzStateSyscallCloneMT(f *testing.F) {
	v := GetMultiThreadedTestCase(f)
	f.Fuzz(func(t *testing.T, nextThreadId, stackPtr uint32, seed int64) {
		goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(seed))
		state := mttestutil.GetMtState(t, goVm)
		// Update existing threads to avoid collision with nextThreadId
		if mttestutil.FindThread(state, nextThreadId) != nil {
			for i, t := range mttestutil.GetAllThreads(state) {
				t.ThreadId = nextThreadId - uint32(i+1)
			}
		}

		// Setup
		state.NextThreadId = nextThreadId
		state.GetMemory().SetMemory(state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = exec.SysClone
		state.GetRegistersRef()[4] = exec.ValidCloneFlags
		state.GetRegistersRef()[5] = stackPtr
		step := state.GetStep()

		// Set up expectations
		expected := mttestutil.NewExpectedMTState(state)
		expected.Step += 1
		// Set original thread expectations
		expected.PrestateActiveThread().PC = state.GetCpu().NextPC
		expected.PrestateActiveThread().NextPC = state.GetCpu().NextPC + 4
		expected.PrestateActiveThread().Registers[2] = nextThreadId
		expected.PrestateActiveThread().Registers[7] = 0
		// Set expectations for new, cloned thread
		expected.ActiveThreadId = nextThreadId
		epxectedNewThread := expected.ExpectNewThread()
		epxectedNewThread.PC = state.GetCpu().NextPC
		epxectedNewThread.NextPC = state.GetCpu().NextPC + 4
		epxectedNewThread.Registers[2] = 0
		epxectedNewThread.Registers[7] = 0
		epxectedNewThread.Registers[29] = stackPtr
		expected.NextThreadId = nextThreadId + 1
		expected.StepsSinceLastContextSwitch = 0
		if state.TraverseRight {
			expected.RightStackSize += 1
		} else {
			expected.LeftStackSize += 1
		}

		stepWitness, err := goVm.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		expected.Validate(t, state)

		evm := testutil.NewMIPSEVM(v.Contracts)
		evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
		goPost, _ := goVm.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}
