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

func TestEVM_SysClone_FlagHandling(t *testing.T) {
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

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := multithreaded.CreateEmptyState()
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysClone // Set syscall number
			state.GetRegistersRef()[4] = c.flags       // Set first argument
			curStep := state.Step

			var err error
			var stepWitness *mipsevm.StepWitness
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			if !c.valid {
				// The VM should exit
				stepWitness, err = us.Step(true)
				require.NoError(t, err)
				require.Equal(t, curStep+1, state.GetStep())
				require.Equal(t, true, us.GetState().GetExited())
				require.Equal(t, uint8(mipsevm.VMStatusPanic), us.GetState().GetExitCode())
				require.Equal(t, 1, state.ThreadCount())
			} else {
				stepWitness, err = us.Step(true)
				require.NoError(t, err)
				require.Equal(t, curStep+1, state.GetStep())
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

func TestEVM_SysClone_Successful(t *testing.T) {
	contracts := testutil.TestContractsSetup(t, testutil.MipsMultithreaded)
	var tracer *tracing.Hooks

	cases := []struct {
		name          string
		traverseRight bool
	}{
		{"traverse left", false},
		{"traverse right", true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stackPtr := uint32(100)
			pc := uint32(200)
			hi := uint32(300)
			lo := uint32(400)

			state := multithreaded.CreateEmptyState()
			if c.traverseRight {
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
			if c.traverseRight {
				activeStack = state.RightThreadStack
				inactiveStack = state.LeftThreadStack
			} else {
				activeStack = state.LeftThreadStack
				inactiveStack = state.RightThreadStack
			}

			require.Equal(t, curStep+1, state.GetStep())
			// Check a new thread was added where we expect
			require.Equal(t, c.traverseRight, state.TraverseRight)
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

func TestEVM_SysGetTID(t *testing.T) {
	var tracer *tracing.Hooks
	contracts := testutil.TestContractsSetup(t, testutil.MipsMultithreaded)
	cases := []struct {
		name     string
		threadId uint32
	}{
		{"zero", 0},
		{"non-zero", 11},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			state := multithreaded.CreateEmptyState()
			state.GetCurrentThread().ThreadId = c.threadId
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			*state.GetRegistersRef() = RandomRegisters(int64(c.threadId))
			state.GetRegistersRef()[2] = exec.SysGetTID // Set syscall number
			curStep := state.Step

			// Set up post-state expectations
			nextPC := state.GetCpu().NextPC
			expectedRegisters := testutil.CopyRegisters(state)
			expectedRegisters[2] = c.threadId // tid return value
			expectedRegisters[7] = 0          // no error

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			stepWitness, err = us.Step(true)
			require.NoError(t, err)

			// Validate post-state
			require.Equal(t, curStep+1, state.GetStep())
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

func TestEVM_SysExit(t *testing.T) {
	var tracer *tracing.Hooks
	contracts := testutil.TestContractsSetup(t, testutil.MipsMultithreaded)
	cases := []struct {
		name        string
		threadCount int
	}{
		{"one thread", 1},
		{"two threads ", 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exitCode := uint8(3)
			state := multithreaded.CreateEmptyState()
			for i := 0; i < c.threadCount-1; i++ {
				newThread := multithreaded.CreateEmptyThread()
				newThread.ThreadId = uint32(i + 1)
				state.LeftThreadStack = append(state.LeftThreadStack, newThread)
			}

			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			*state.GetRegistersRef() = RandomRegisters(int64(c.threadCount))
			state.GetRegistersRef()[2] = exec.SysExit     // Set syscall number
			state.GetRegistersRef()[4] = uint32(exitCode) // The first argument (exit code)
			curStep := state.Step

			// Set up post-state expectations
			pc := state.GetCpu().PC
			nextPC := state.GetCpu().NextPC
			expectedRegisters := testutil.CopyRegisters(state) // No change

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			stepWitness, err = us.Step(true)
			require.NoError(t, err)

			// Validate post-state
			thread := state.GetCurrentThread()
			require.Equal(t, curStep+1, state.GetStep())
			require.Equal(t, c.threadCount, state.ThreadCount())
			require.Equal(t, expectedRegisters, state.GetRegistersRef())
			require.Equal(t, pc, state.GetPC())
			require.Equal(t, nextPC, state.GetCpu().NextPC)
			require.Equal(t, true, thread.Exited)
			require.Equal(t, exitCode, thread.ExitCode)
			if c.threadCount == 1 {
				// If we exit the last thread, the whole process should exit
				require.Equal(t, true, state.Exited)
				require.Equal(t, exitCode, state.ExitCode)
			} else {
				require.Equal(t, false, state.Exited)
				require.Equal(t, uint8(0), state.ExitCode)
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

func TestEVM_PopExitedThread(t *testing.T) {
	var tracer *tracing.Hooks
	contracts := testutil.TestContractsSetup(t, testutil.MipsMultithreaded)
	cases := []struct {
		name                   string
		traverseRight          bool
		activeStackThreadCount int
	}{
		{"traverse right, pop last thread", true, 1},
		{"traverse right, pop penultimate thread", true, 2},
		{"traverse left, pop last thread", false, 1},
		{"traverse left, pop penultimate thread", false, 2},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var state *multithreaded.State
			if c.traverseRight {
				state = setupThreads(c.traverseRight, c.activeStackThreadCount, 1)
			} else {
				state = setupThreads(c.traverseRight, 1, c.activeStackThreadCount)
			}
			threadToPop := state.GetCurrentThread()
			threadToPop.Exited = true
			threadToPop.ExitCode = 1

			// Record current state, expectations
			curStep := state.Step
			initThreadCount := state.ThreadCount()
			shouldChangeDirections := c.activeStackThreadCount == 1
			postShouldTraverseRight := c.traverseRight && !shouldChangeDirections || !c.traverseRight && shouldChangeDirections
			// Sanity check
			require.Equal(t, c.activeStackThreadCount+1, initThreadCount)
			require.Equal(t, c.traverseRight, state.TraverseRight)

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			stepWitness, err = us.Step(true)
			require.NoError(t, err)

			// Validate post-state
			require.Equal(t, curStep+1, state.GetStep())
			require.Equal(t, initThreadCount-1, state.ThreadCount())
			require.False(t, checkStateContainsThread(state, threadToPop.ThreadId))
			require.Equal(t, postShouldTraverseRight, state.TraverseRight)

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

func setupThreads(traverseRight bool, rightThreadCount, leftThreadCount int) *multithreaded.State {
	state := multithreaded.CreateEmptyState()
	var leftThreads, rightThreads []*multithreaded.ThreadState

	tid := uint32(0)
	for i := 0; i < rightThreadCount; i++ {
		thread := multithreaded.CreateEmptyThread()
		thread.ThreadId = tid
		rightThreads = append(rightThreads, thread)
		tid++
	}

	for i := 0; i < leftThreadCount; i++ {
		thread := multithreaded.CreateEmptyThread()
		thread.ThreadId = tid
		leftThreads = append(leftThreads, thread)
		tid++
	}

	state.LeftThreadStack = leftThreads
	state.RightThreadStack = rightThreads
	state.TraverseRight = traverseRight

	return state
}

func checkStateContainsThread(state *multithreaded.State, threadId uint32) bool {
	for i := 0; i < len(state.RightThreadStack); i++ {
		if state.RightThreadStack[i].ThreadId == threadId {
			return true
		}
	}

	for i := 0; i < len(state.LeftThreadStack); i++ {
		if state.LeftThreadStack[i].ThreadId == threadId {
			return true
		}
	}

	return false
}
