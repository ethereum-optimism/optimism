package tests

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func TestEVM_CloneFlags(t *testing.T) {
	contracts := testutil.TestContractsSetup(t, testutil.MipsMultithreaded)
	var tracer *tracing.Hooks

	cases := []struct {
		name  string
		flags uint32
		valid bool
	}{
		{"the supported flags bitmask", exec.ValidCloneFlags, true},
		{"no flags", 0, false},
		{"all flags", ^uint32(0), false},
		{"all unsupported flags", ^uint32(exec.ValidCloneFlags), false},
		{"a few supported flags", exec.CloneFs | exec.CloneSysvsem, false},
		{"one supported flag", exec.CloneFs, false},
		{"mixed supported and unsupported flags", exec.CloneFs | exec.CloneParentSettid, false},
		{"a single unsupported flag", exec.CloneUntraced, false},
		{"multiple unsupported flags", exec.CloneUntraced | exec.CloneParentSettid, false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			state := multithreaded.CreateEmptyState()
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysClone // Set syscall number
			state.GetRegistersRef()[4] = tt.flags      // Set first argument
			curStep := state.Step

			var err error
			var stepWitness *mipsevm.StepWitness
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			if !tt.valid {
				// The VM should exit
				stepWitness, err = us.Step(true)
				require.NoError(t, err)
				require.Equal(t, true, us.GetState().GetExited())
				require.Equal(t, uint8(mipsevm.VMStatusPanic), us.GetState().GetExitCode())
				require.Equal(t, 1, state.ThreadCount())
			} else {
				stepWitness, err = us.Step(true)
				require.NoError(t, err)
				require.Equal(t, false, us.GetState().GetExited())
				require.Equal(t, uint8(0), us.GetState().GetExitCode())
				require.Equal(t, 2, state.ThreadCount())
			}

			evm := testutil.NewMIPSEVM(contracts)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			evmPost := evm.Step(t, stepWitness, curStep, multithreaded.GetStateHashFn())
			goPost, _ := us.GetState().EncodeWitness()
			require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
				"mipsevm produced different state than EVM")
		})
	}
}

func TestEVM_CloneSuccessful(t *testing.T) {
	contracts := testutil.TestContractsSetup(t, testutil.MipsMultithreaded)
	var tracer *tracing.Hooks

	cases := []struct {
		name          string
		traverseRight bool
	}{
		{"traverse left", false},
		{"traverse right", true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stackPtr := uint32(100)
			pc := uint32(200)
			hi := uint32(300)
			lo := uint32(400)

			state := multithreaded.CreateEmptyState()
			if tt.traverseRight {
				// Reorganize threads
				state.RightThreadStack = []*multithreaded.ThreadState{multithreaded.CreateEmptyThread()}
				state.LeftThreadStack = []*multithreaded.ThreadState{}
				state.TraverseRight = true
			} else {
				// Sanity-check we are already traversing left
				require.Equal(t, false, state.TraverseRight)
			}

			state.GetCurrentThread().Cpu.PC = pc
			state.GetCurrentThread().Cpu.NextPC = pc + 4
			state.GetCurrentThread().Cpu.HI = hi
			state.GetCurrentThread().Cpu.LO = lo
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			*state.GetRegistersRef() = RandomRegisters(1)
			state.GetRegistersRef()[2] = exec.SysClone        // the syscall number
			state.GetRegistersRef()[4] = exec.ValidCloneFlags // a0 - first argument, clone flags
			state.GetRegistersRef()[5] = stackPtr             // a1 - the stack pointer

			curStep := state.Step
			origThread := state.GetCurrentThread()
			origThreadExpectedRegisters := *testutil.CopyRegisters(state)
			origThreadExpectedRegisters[2] = 1
			origThreadExpectedRegisters[7] = 0
			newThreadExpectedRegisters := *testutil.CopyRegisters(state)
			newThreadExpectedRegisters[2] = 0
			newThreadExpectedRegisters[7] = 0
			newThreadExpectedRegisters[29] = stackPtr

			var err error
			var stepWitness *mipsevm.StepWitness
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)

			stepWitness, err = us.Step(true)
			require.NoError(t, err)

			var activeStack, inactiveStack []*multithreaded.ThreadState
			if tt.traverseRight {
				activeStack = state.RightThreadStack
				inactiveStack = state.LeftThreadStack
			} else {
				activeStack = state.LeftThreadStack
				inactiveStack = state.RightThreadStack
			}

			// Check a new thread was added where we expect
			require.Equal(t, tt.traverseRight, state.TraverseRight)
			require.Equal(t, 2, state.ThreadCount())
			require.Equal(t, 2, len(activeStack))
			require.Equal(t, 0, len(inactiveStack))
			require.Equal(t, uint32(2), state.NextThreadId)

			// Validate new thread
			newThread := state.GetCurrentThread()
			require.Equal(t, uint32(1), newThread.ThreadId)
			require.Equal(t, pc+4, newThread.Cpu.PC)
			require.Equal(t, pc+8, newThread.Cpu.NextPC)
			require.Equal(t, hi, newThread.Cpu.HI)
			require.Equal(t, lo, newThread.Cpu.LO)
			require.Equal(t, false, newThread.Exited)
			require.Equal(t, uint8(0), newThread.ExitCode)
			require.Equal(t, exec.FutexEmptyAddr, newThread.FutexAddr)
			require.Equal(t, uint32(0), newThread.FutexVal)
			require.Equal(t, uint64(0), newThread.FutexTimeoutStep)
			require.Equal(t, newThreadExpectedRegisters, newThread.Registers)

			// Validate parent thread
			require.Equal(t, uint32(0), origThread.ThreadId)
			require.Equal(t, origThreadExpectedRegisters, origThread.Registers)
			require.Equal(t, pc+4, newThread.Cpu.PC)
			require.Equal(t, pc+8, newThread.Cpu.NextPC)

			evm := testutil.NewMIPSEVM(contracts)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			evmPost := evm.Step(t, stepWitness, curStep, multithreaded.GetStateHashFn())
			goPost, _ := us.GetState().EncodeWitness()
			require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
				"mipsevm produced different state than EVM")
		})
	}
}

func TestEVM_GetTID(t *testing.T) {
	var tracer *tracing.Hooks
	contracts := testutil.TestContractsSetup(t, testutil.MipsMultithreaded)
	cases := []struct {
		name     string
		threadId uint32
	}{
		{"zero", 0},
		{"non-zero", 11},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			state := multithreaded.CreateEmptyState()
			state.GetCurrentThread().ThreadId = tt.threadId
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			*state.GetRegistersRef() = RandomRegisters(int64(tt.threadId))
			state.GetRegistersRef()[2] = exec.SysGetTID // Set syscall number
			curStep := state.Step

			nextPC := state.GetCpu().NextPC
			expectedRegisters := testutil.CopyRegisters(state)
			expectedRegisters[2] = tt.threadId // tid return value
			expectedRegisters[7] = 0           // no error

			var err error
			var stepWitness *mipsevm.StepWitness
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			stepWitness, err = us.Step(true)
			require.NoError(t, err)

			// Validate post-state
			require.Equal(t, 1, state.ThreadCount())
			require.Equal(t, expectedRegisters, state.GetRegistersRef())
			require.Equal(t, nextPC, state.GetPC())
			require.Equal(t, nextPC+4, state.GetCpu().NextPC)

			evm := testutil.NewMIPSEVM(contracts)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			evmPost := evm.Step(t, stepWitness, curStep, multithreaded.GetStateHashFn())
			goPost, _ := us.GetState().EncodeWitness()
			require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
				"mipsevm produced different state than EVM")
		})
	}
}
