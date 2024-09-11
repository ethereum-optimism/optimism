package tests

import (
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mttestutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
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
			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil, nil)
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
	var tracer *tracing.Hooks
	cases := []struct {
		name          string
		traverseRight bool
	}{
		{"traverse left", false},
		{"traverse right", true},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			stackPtr := uint32(100)

			goVm, state, contracts := setup(t, i)
			mttestutil.InitializeSingleThread(i*333, state, c.traverseRight)
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysClone        // the syscall number
			state.GetRegistersRef()[4] = exec.ValidCloneFlags // a0 - first argument, clone flags
			state.GetRegistersRef()[5] = stackPtr             // a1 - the stack pointer
			step := state.GetStep()

			// Sanity-check assumptions
			require.Equal(t, uint32(1), state.NextThreadId)

			// Setup expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.Step += 1
			expectedNewThread := expected.ExpectNewThread()
			expected.ActiveThreadId = expectedNewThread.ThreadId
			expected.StepsSinceLastContextSwitch = 0
			if c.traverseRight {
				expected.RightStackSize += 1
			} else {
				expected.LeftStackSize += 1
			}
			// Original thread expectations
			expected.PrestateActiveThread().PC = state.GetCpu().NextPC
			expected.PrestateActiveThread().NextPC = state.GetCpu().NextPC + 4
			expected.PrestateActiveThread().Registers[2] = 1
			expected.PrestateActiveThread().Registers[7] = 0
			// New thread expectations
			expectedNewThread.PC = state.GetCpu().NextPC
			expectedNewThread.NextPC = state.GetCpu().NextPC + 4
			expectedNewThread.ThreadId = 1
			expectedNewThread.Registers[2] = 0
			expectedNewThread.Registers[7] = 0
			expectedNewThread.Registers[29] = stackPtr

			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			expected.Validate(t, state)
			activeStack, inactiveStack := mttestutil.GetThreadStacks(state)
			require.Equal(t, 2, len(activeStack))
			require.Equal(t, 0, len(inactiveStack))
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_SysGetTID(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name     string
		threadId uint32
	}{
		{"zero", 0},
		{"non-zero", 11},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goVm, state, contracts := setup(t, i*789)
			mttestutil.InitializeSingleThread(i*789, state, false)

			state.GetCurrentThread().ThreadId = c.threadId
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysGetTID // Set syscall number
			step := state.Step

			// Set up post-state expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = c.threadId
			expected.ActiveThread().Registers[7] = 0

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_SysExit(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name               string
		threadCount        int
		shouldExitGlobally bool
	}{
		// If we exit the last thread, the whole process should exit
		{name: "one thread", threadCount: 1, shouldExitGlobally: true},
		{name: "two threads ", threadCount: 2},
		{name: "three threads ", threadCount: 3},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			exitCode := uint8(3)

			goVm, state, contracts := setup(t, i*133)
			mttestutil.SetupThreads(int64(i*1111), state, i%2 == 0, c.threadCount, 0)

			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysExit     // Set syscall number
			state.GetRegistersRef()[4] = uint32(exitCode) // The first argument (exit code)
			step := state.Step

			// Set up expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.Step += 1
			expected.StepsSinceLastContextSwitch += 1
			expected.ActiveThread().Exited = true
			expected.ActiveThread().ExitCode = exitCode
			if c.shouldExitGlobally {
				expected.Exited = true
				expected.ExitCode = exitCode
			}

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_PopExitedThread(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name                         string
		traverseRight                bool
		activeStackThreadCount       int
		expectTraverseRightPostState bool
	}{
		{name: "traverse right", traverseRight: true, activeStackThreadCount: 2, expectTraverseRightPostState: true},
		{name: "traverse right, switch directions", traverseRight: true, activeStackThreadCount: 1, expectTraverseRightPostState: false},
		{name: "traverse left", traverseRight: false, activeStackThreadCount: 2, expectTraverseRightPostState: false},
		{name: "traverse left, switch directions", traverseRight: false, activeStackThreadCount: 1, expectTraverseRightPostState: true},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goVm, state, contracts := setup(t, i*133)
			mttestutil.SetupThreads(int64(i*222), state, c.traverseRight, c.activeStackThreadCount, 1)
			step := state.Step

			// Setup thread to be dropped
			threadToPop := state.GetCurrentThread()
			threadToPop.Exited = true
			threadToPop.ExitCode = 1

			// Set up expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.Step += 1
			expected.ActiveThreadId = mttestutil.FindNextThreadExcluding(state, threadToPop.ThreadId).ThreadId
			expected.StepsSinceLastContextSwitch = 0
			expected.ThreadCount -= 1
			expected.TraverseRight = c.expectTraverseRightPostState
			expected.Thread(threadToPop.ThreadId).Dropped = true
			if c.traverseRight {
				expected.RightStackSize -= 1
			} else {
				expected.LeftStackSize -= 1
			}

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_SysFutex_WaitPrivate(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name             string
		address          uint32
		targetValue      uint32
		actualValue      uint32
		timeout          uint32
		shouldFail       bool
		shouldSetTimeout bool
	}{
		{name: "successful wait, no timeout", address: 0x1234, targetValue: 0x01, actualValue: 0x01},
		{name: "memory mismatch, no timeout", address: 0x1200, targetValue: 0x01, actualValue: 0x02, shouldFail: true},
		{name: "successful wait w timeout", address: 0x1234, targetValue: 0x01, actualValue: 0x01, timeout: 1000000, shouldSetTimeout: true},
		{name: "memory mismatch w timeout", address: 0x1200, targetValue: 0x01, actualValue: 0x02, timeout: 2000000, shouldFail: true},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goVm, state, contracts := setup(t, i*1234)
			step := state.GetStep()

			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.Memory.SetMemory(c.address, c.actualValue)
			state.GetRegistersRef()[2] = exec.SysFutex // Set syscall number
			state.GetRegistersRef()[4] = c.address
			state.GetRegistersRef()[5] = exec.FutexWaitPrivate
			state.GetRegistersRef()[6] = c.targetValue
			state.GetRegistersRef()[7] = c.timeout

			// Setup expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.Step += 1
			expected.StepsSinceLastContextSwitch += 1
			if c.shouldFail {
				expected.ActiveThread().PC = state.GetCpu().NextPC
				expected.ActiveThread().NextPC = state.GetCpu().NextPC + 4
				expected.ActiveThread().Registers[2] = exec.SysErrorSignal
				expected.ActiveThread().Registers[7] = exec.MipsEAGAIN
			} else {
				// PC and return registers should not update on success, updates happen when wait completes
				expected.ActiveThread().FutexAddr = c.address
				expected.ActiveThread().FutexVal = c.targetValue
				expected.ActiveThread().FutexTimeoutStep = exec.FutexNoTimeout
				if c.shouldSetTimeout {
					expected.ActiveThread().FutexTimeoutStep = step + exec.FutexTimeoutSteps + 1
				}
			}

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_SysFutex_WakePrivate(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name                string
		address             uint32
		activeThreadCount   int
		inactiveThreadCount int
		traverseRight       bool
		expectTraverseRight bool
	}{
		{name: "Traverse right", address: 0x6789, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: true},
		{name: "Traverse right, no left threads", address: 0x6789, activeThreadCount: 2, inactiveThreadCount: 0, traverseRight: true},
		{name: "Traverse right, single thread", address: 0x6789, activeThreadCount: 1, inactiveThreadCount: 0, traverseRight: true},
		{name: "Traverse left", address: 0x6789, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: false},
		{name: "Traverse left, switch directions", address: 0x6789, activeThreadCount: 1, inactiveThreadCount: 1, traverseRight: false, expectTraverseRight: true},
		{name: "Traverse left, single thread", address: 0x6789, activeThreadCount: 1, inactiveThreadCount: 0, traverseRight: false, expectTraverseRight: true},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goVm, state, contracts := setup(t, i*1122)
			mttestutil.SetupThreads(int64(i*2244), state, c.traverseRight, c.activeThreadCount, c.inactiveThreadCount)
			step := state.Step

			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysFutex // Set syscall number
			state.GetRegistersRef()[4] = c.address
			state.GetRegistersRef()[5] = exec.FutexWakePrivate

			// Set up post-state expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = 0
			expected.ActiveThread().Registers[7] = 0
			expected.Wakeup = c.address
			expected.ExpectPreemption(state)
			expected.TraverseRight = c.expectTraverseRight
			if c.traverseRight != c.expectTraverseRight {
				// If we preempt the current thread and then switch directions, the same
				// thread will remain active
				expected.ActiveThreadId = state.GetCurrentThread().ThreadId
			}

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_SysFutex_UnsupportedOp(t *testing.T) {
	var tracer *tracing.Hooks

	// From: https://github.com/torvalds/linux/blob/5be63fc19fcaa4c236b307420483578a56986a37/include/uapi/linux/futex.h
	const FUTEX_PRIVATE_FLAG = 128
	const FUTEX_WAIT = 0
	const FUTEX_WAKE = 1
	const FUTEX_FD = 2
	const FUTEX_REQUEUE = 3
	const FUTEX_CMP_REQUEUE = 4
	const FUTEX_WAKE_OP = 5
	const FUTEX_LOCK_PI = 6
	const FUTEX_UNLOCK_PI = 7
	const FUTEX_TRYLOCK_PI = 8
	const FUTEX_WAIT_BITSET = 9
	const FUTEX_WAKE_BITSET = 10
	const FUTEX_WAIT_REQUEUE_PI = 11
	const FUTEX_CMP_REQUEUE_PI = 12
	const FUTEX_LOCK_PI2 = 13

	unsupportedFutexOps := map[string]uint32{
		"FUTEX_WAIT":                    FUTEX_WAIT,
		"FUTEX_WAKE":                    FUTEX_WAKE,
		"FUTEX_FD":                      FUTEX_FD,
		"FUTEX_REQUEUE":                 FUTEX_REQUEUE,
		"FUTEX_CMP_REQUEUE":             FUTEX_CMP_REQUEUE,
		"FUTEX_WAKE_OP":                 FUTEX_WAKE_OP,
		"FUTEX_LOCK_PI":                 FUTEX_LOCK_PI,
		"FUTEX_UNLOCK_PI":               FUTEX_UNLOCK_PI,
		"FUTEX_TRYLOCK_PI":              FUTEX_TRYLOCK_PI,
		"FUTEX_WAIT_BITSET":             FUTEX_WAIT_BITSET,
		"FUTEX_WAKE_BITSET":             FUTEX_WAKE_BITSET,
		"FUTEX_WAIT_REQUEUE_PI":         FUTEX_WAIT_REQUEUE_PI,
		"FUTEX_CMP_REQUEUE_PI":          FUTEX_CMP_REQUEUE_PI,
		"FUTEX_LOCK_PI2":                FUTEX_LOCK_PI2,
		"FUTEX_REQUEUE_PRIVATE":         (FUTEX_REQUEUE | FUTEX_PRIVATE_FLAG),
		"FUTEX_CMP_REQUEUE_PRIVATE":     (FUTEX_CMP_REQUEUE | FUTEX_PRIVATE_FLAG),
		"FUTEX_WAKE_OP_PRIVATE":         (FUTEX_WAKE_OP | FUTEX_PRIVATE_FLAG),
		"FUTEX_LOCK_PI_PRIVATE":         (FUTEX_LOCK_PI | FUTEX_PRIVATE_FLAG),
		"FUTEX_LOCK_PI2_PRIVATE":        (FUTEX_LOCK_PI2 | FUTEX_PRIVATE_FLAG),
		"FUTEX_UNLOCK_PI_PRIVATE":       (FUTEX_UNLOCK_PI | FUTEX_PRIVATE_FLAG),
		"FUTEX_TRYLOCK_PI_PRIVATE":      (FUTEX_TRYLOCK_PI | FUTEX_PRIVATE_FLAG),
		"FUTEX_WAIT_BITSET_PRIVATE":     (FUTEX_WAIT_BITSET | FUTEX_PRIVATE_FLAG),
		"FUTEX_WAKE_BITSET_PRIVATE":     (FUTEX_WAKE_BITSET | FUTEX_PRIVATE_FLAG),
		"FUTEX_WAIT_REQUEUE_PI_PRIVATE": (FUTEX_WAIT_REQUEUE_PI | FUTEX_PRIVATE_FLAG),
		"FUTEX_CMP_REQUEUE_PI_PRIVATE":  (FUTEX_CMP_REQUEUE_PI | FUTEX_PRIVATE_FLAG),
	}

	for name, op := range unsupportedFutexOps {
		t.Run(name, func(t *testing.T) {
			goVm, state, contracts := setup(t, int(op))
			step := state.GetStep()

			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysFutex // Set syscall number
			state.GetRegistersRef()[5] = op

			// Setup expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.Step += 1
			expected.StepsSinceLastContextSwitch += 1
			expected.ActiveThread().PC = state.GetCpu().NextPC
			expected.ActiveThread().NextPC = state.GetCpu().NextPC + 4
			expected.ActiveThread().Registers[2] = exec.SysErrorSignal
			expected.ActiveThread().Registers[7] = exec.MipsEINVAL

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_SysYield(t *testing.T) {
	runPreemptSyscall(t, "SysSchedYield", exec.SysSchedYield)
}

func TestEVM_SysNanosleep(t *testing.T) {
	runPreemptSyscall(t, "SysNanosleep", exec.SysNanosleep)
}

func runPreemptSyscall(t *testing.T, syscallName string, syscallNum uint32) {
	var tracer *tracing.Hooks
	cases := []struct {
		name            string
		traverseRight   bool
		activeThreads   int
		inactiveThreads int
	}{
		{name: "Last active thread", activeThreads: 1, inactiveThreads: 2},
		{name: "Only thread", activeThreads: 1, inactiveThreads: 0},
		{name: "Do not change directions", activeThreads: 2, inactiveThreads: 2},
		{name: "Do not change directions", activeThreads: 3, inactiveThreads: 0},
	}

	for i, c := range cases {
		for _, traverseRight := range []bool{true, false} {
			testName := fmt.Sprintf("%v: %v (traverseRight = %v)", syscallName, c.name, traverseRight)
			t.Run(testName, func(t *testing.T) {
				goVm, state, contracts := setup(t, i*789)
				mttestutil.SetupThreads(int64(i*3259), state, traverseRight, c.activeThreads, c.inactiveThreads)

				state.Memory.SetMemory(state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = syscallNum // Set syscall number
				step := state.Step

				// Set up post-state expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.ExpectStep()
				expected.ExpectPreemption(state)
				expected.PrestateActiveThread().Registers[2] = 0
				expected.PrestateActiveThread().Registers[7] = 0

				// State transition
				var err error
				var stepWitness *mipsevm.StepWitness
				stepWitness, err = goVm.Step(true)
				require.NoError(t, err)

				// Validate post-state
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
			})
		}
	}
}

func TestEVM_SysOpen(t *testing.T) {
	var tracer *tracing.Hooks

	goVm, state, contracts := setup(t, 5512)

	state.Memory.SetMemory(state.GetPC(), syscallInsn)
	state.GetRegistersRef()[2] = exec.SysOpen // Set syscall number
	step := state.Step

	// Set up post-state expectations
	expected := mttestutil.NewExpectedMTState(state)
	expected.ExpectStep()
	expected.ActiveThread().Registers[2] = exec.SysErrorSignal
	expected.ActiveThread().Registers[7] = exec.MipsEBADF

	// State transition
	var err error
	var stepWitness *mipsevm.StepWitness
	stepWitness, err = goVm.Step(true)
	require.NoError(t, err)

	// Validate post-state
	expected.Validate(t, state)
	testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
}

func TestEVM_SysGetPID(t *testing.T) {
	var tracer *tracing.Hooks
	goVm, state, contracts := setup(t, 1929)

	state.Memory.SetMemory(state.GetPC(), syscallInsn)
	state.GetRegistersRef()[2] = exec.SysGetpid // Set syscall number
	step := state.Step

	// Set up post-state expectations
	expected := mttestutil.NewExpectedMTState(state)
	expected.ExpectStep()
	expected.ActiveThread().Registers[2] = 0
	expected.ActiveThread().Registers[7] = 0

	// State transition
	var err error
	var stepWitness *mipsevm.StepWitness
	stepWitness, err = goVm.Step(true)
	require.NoError(t, err)

	// Validate post-state
	expected.Validate(t, state)
	testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
}

func TestEVM_SysClockGettimeMonotonic(t *testing.T) {
	testEVM_SysClockGettime(t, exec.ClockGettimeMonotonicFlag)
}

func TestEVM_SysClockGettimeRealtime(t *testing.T) {
	testEVM_SysClockGettime(t, exec.ClockGettimeRealtimeFlag)
}

func testEVM_SysClockGettime(t *testing.T, clkid uint32) {
	var tracer *tracing.Hooks

	cases := []struct {
		name         string
		timespecAddr uint32
	}{
		{"aligned timespec address", 0x1000},
		{"unaligned timespec address", 0x1003},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goVm, state, contracts := setup(t, 2101)

			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysClockGetTime // Set syscall number
			state.GetRegistersRef()[4] = clkid                // a0
			state.GetRegistersRef()[5] = c.timespecAddr       // a1
			step := state.Step

			expected := mttestutil.NewExpectedMTState(state)
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = 0
			expected.ActiveThread().Registers[7] = 0
			next := state.Step + 1
			var secs, nsecs uint32
			if clkid == exec.ClockGettimeMonotonicFlag {
				secs = uint32(next / exec.HZ)
				nsecs = uint32((next % exec.HZ) * (1_000_000_000 / exec.HZ))
			}
			effAddr := c.timespecAddr & 0xFFffFFfc
			expected.ExpectMemoryWrite(effAddr, secs)
			expected.ExpectMemoryWrite(effAddr+4, nsecs)

			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_SysClockGettimeNonMonotonic(t *testing.T) {
	var tracer *tracing.Hooks
	goVm, state, contracts := setup(t, 2101)

	timespecAddr := uint32(0x1000)
	state.Memory.SetMemory(state.GetPC(), syscallInsn)
	state.GetRegistersRef()[2] = exec.SysClockGetTime // Set syscall number
	state.GetRegistersRef()[4] = 0xDEAD               // a0 - invalid clockid
	state.GetRegistersRef()[5] = timespecAddr         // a1
	step := state.Step

	expected := mttestutil.NewExpectedMTState(state)
	expected.ExpectStep()
	expected.ActiveThread().Registers[2] = exec.SysErrorSignal
	expected.ActiveThread().Registers[7] = exec.MipsEINVAL

	var err error
	var stepWitness *mipsevm.StepWitness
	stepWitness, err = goVm.Step(true)
	require.NoError(t, err)

	// Validate post-state
	expected.Validate(t, state)
	testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
}

var NoopSyscalls = map[string]uint32{
	"SysGetAffinity":   4240,
	"SysMadvise":       4218,
	"SysRtSigprocmask": 4195,
	"SysSigaltstack":   4206,
	"SysRtSigaction":   4194,
	"SysPrlimit64":     4338,
	"SysClose":         4006,
	"SysPread64":       4200,
	"SysFstat64":       4215,
	"SysOpenAt":        4288,
	"SysReadlink":      4085,
	"SysReadlinkAt":    4298,
	"SysIoctl":         4054,
	"SysEpollCreate1":  4326,
	"SysPipe2":         4328,
	"SysEpollCtl":      4249,
	"SysEpollPwait":    4313,
	"SysGetRandom":     4353,
	"SysUname":         4122,
	"SysStat64":        4213,
	"SysGetuid":        4024,
	"SysGetgid":        4047,
	"SysLlseek":        4140,
	"SysMinCore":       4217,
	"SysTgkill":        4266,
	"SysMunmap":        4091,
	"SysSetITimer":     4104,
	"SysTimerCreate":   4257,
	"SysTimerSetTime":  4258,
	"SysTimerDelete":   4261,
}

func TestEVM_NoopSyscall(t *testing.T) {
	var tracer *tracing.Hooks
	for noopName, noopVal := range NoopSyscalls {
		t.Run(noopName, func(t *testing.T) {
			goVm, state, contracts := setup(t, int(noopVal))

			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = noopVal // Set syscall number
			step := state.Step

			// Set up post-state expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = 0
			expected.ActiveThread().Registers[7] = 0

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})

	}
}

func TestEVM_UnsupportedSyscall(t *testing.T) {
	t.Parallel()
	var tracer *tracing.Hooks

	var NoopSyscallNums = maps.Values(NoopSyscalls)
	var SupportedSyscalls = []uint32{exec.SysMmap, exec.SysBrk, exec.SysClone, exec.SysExitGroup, exec.SysRead, exec.SysWrite, exec.SysFcntl, exec.SysExit, exec.SysSchedYield, exec.SysGetTID, exec.SysFutex, exec.SysOpen, exec.SysNanosleep, exec.SysClockGetTime, exec.SysGetpid}
	unsupportedSyscalls := make([]uint32, 0, 400)
	for i := 4000; i < 4400; i++ {
		candidate := uint32(i)
		if slices.Contains(SupportedSyscalls, candidate) || slices.Contains(NoopSyscallNums, candidate) {
			continue
		}
		unsupportedSyscalls = append(unsupportedSyscalls, candidate)
	}

	for i, syscallNum := range unsupportedSyscalls {
		testName := fmt.Sprintf("Unsupported syscallNum %v", syscallNum)
		i := i
		syscallNum := syscallNum
		t.Run(testName, func(t *testing.T) {
			t.Parallel()
			goVm, state, contracts := setup(t, i*3434)
			// Setup basic getThreadId syscall instruction
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = syscallNum

			// Set up post-state expectations
			require.Panics(t, func() { _, _ = goVm.Step(true) })
			testutil.AssertEVMReverts(t, state, contracts, tracer)
		})
	}
}

func TestEVM_NormalTraversalStep_HandleWaitingThread(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name            string
		step            uint64
		activeStackSize int
		otherStackSize  int
		futexAddr       uint32
		targetValue     uint32
		actualValue     uint32
		timeoutStep     uint64
		shouldWakeup    bool
		shouldTimeout   bool
	}{
		{name: "Preempt, no timeout #1", step: 100, activeStackSize: 1, otherStackSize: 0, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x01, timeoutStep: exec.FutexNoTimeout},
		{name: "Preempt, no timeout #2", step: 100, activeStackSize: 1, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x01, timeoutStep: exec.FutexNoTimeout},
		{name: "Preempt, no timeout #3", step: 100, activeStackSize: 2, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x01, timeoutStep: exec.FutexNoTimeout},
		{name: "Preempt, with timeout #1", step: 100, activeStackSize: 2, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x01, timeoutStep: 101},
		{name: "Preempt, with timeout #2", step: 100, activeStackSize: 1, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x01, timeoutStep: 150},
		{name: "Wakeup, no timeout #1", step: 100, activeStackSize: 1, otherStackSize: 0, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x02, timeoutStep: exec.FutexNoTimeout, shouldWakeup: true},
		{name: "Wakeup, no timeout #2", step: 100, activeStackSize: 2, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x02, timeoutStep: exec.FutexNoTimeout, shouldWakeup: true},
		{name: "Wakeup with timeout #1", step: 100, activeStackSize: 2, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x01, actualValue: 0x02, timeoutStep: 100, shouldWakeup: true, shouldTimeout: true},
		{name: "Wakeup with timeout #2", step: 100, activeStackSize: 2, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x02, actualValue: 0x02, timeoutStep: 100, shouldWakeup: true, shouldTimeout: true},
		{name: "Wakeup with timeout #3", step: 100, activeStackSize: 2, otherStackSize: 1, futexAddr: 0x100, targetValue: 0x02, actualValue: 0x02, timeoutStep: 50, shouldWakeup: true, shouldTimeout: true},
	}

	for _, c := range cases {
		for i, traverseRight := range []bool{true, false} {
			testName := fmt.Sprintf("%v (traverseRight=%v)", c.name, traverseRight)
			t.Run(testName, func(t *testing.T) {
				// Sanity check
				if !c.shouldWakeup && c.shouldTimeout {
					require.Fail(t, "Invalid test case - cannot expect a timeout with no wakeup")
				}

				goVm, state, contracts := setup(t, i)
				mttestutil.SetupThreads(int64(i*101), state, traverseRight, c.activeStackSize, c.otherStackSize)
				state.Step = c.step

				activeThread := state.GetCurrentThread()
				activeThread.FutexAddr = c.futexAddr
				activeThread.FutexVal = c.targetValue
				activeThread.FutexTimeoutStep = c.timeoutStep
				state.GetMemory().SetMemory(c.futexAddr, c.actualValue)

				// Set up post-state expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.Step += 1
				if c.shouldWakeup {
					expected.ActiveThread().FutexAddr = exec.FutexEmptyAddr
					expected.ActiveThread().FutexVal = 0
					expected.ActiveThread().FutexTimeoutStep = 0
					// PC and return registers are updated onWaitComplete
					expected.ActiveThread().PC = state.GetCpu().NextPC
					expected.ActiveThread().NextPC = state.GetCpu().NextPC + 4
					if c.shouldTimeout {
						expected.ActiveThread().Registers[2] = exec.SysErrorSignal
						expected.ActiveThread().Registers[7] = exec.MipsETIMEDOUT
					} else {
						expected.ActiveThread().Registers[2] = 0
						expected.ActiveThread().Registers[7] = 0
					}
				} else {
					expected.ExpectPreemption(state)
				}

				// State transition
				var err error
				var stepWitness *mipsevm.StepWitness
				stepWitness, err = goVm.Step(true)
				require.NoError(t, err)

				// Validate post-state
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, c.step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
			})

		}
	}
}

func TestEVM_NormalTraversal_Full(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name        string
		threadCount int
	}{
		{"1 thread", 1},
		{"2 threads", 2},
		{"3 threads", 3},
	}

	for i, c := range cases {
		for _, traverseRight := range []bool{true, false} {
			testName := fmt.Sprintf("%v (traverseRight = %v)", c.name, traverseRight)
			t.Run(testName, func(t *testing.T) {
				// Setup
				goVm, state, contracts := setup(t, i*789)
				mttestutil.SetupThreads(int64(i*2947), state, traverseRight, c.threadCount, 0)
				// Put threads into a waiting state so that we just traverse through them
				for _, thread := range mttestutil.GetAllThreads(state) {
					thread.FutexAddr = 0x04
					thread.FutexTimeoutStep = exec.FutexNoTimeout
				}
				step := state.Step

				initialState := mttestutil.NewExpectedMTState(state)

				// Loop through all the threads to get back to the starting state
				iterations := c.threadCount * 2
				for i := 0; i < iterations; i++ {
					// Set up post-state expectations
					expected := mttestutil.NewExpectedMTState(state)
					expected.Step += 1
					expected.ExpectPreemption(state)

					// State transition
					var err error
					var stepWitness *mipsevm.StepWitness
					stepWitness, err = goVm.Step(true)
					require.NoError(t, err)

					// Validate post-state
					expected.Validate(t, state)
					testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
				}

				// We should be back to the original state with only a few modifications
				initialState.Step += uint64(iterations)
				initialState.StepsSinceLastContextSwitch = 0
				initialState.Validate(t, state)
			})
		}
	}
}

func TestEVM_WakeupTraversalStep(t *testing.T) {
	wakeupAddr := uint32(0x1234)
	wakeupVal := uint32(0x999)
	var tracer *tracing.Hooks
	cases := []struct {
		name              string
		futexAddr         uint32
		targetVal         uint32
		traverseRight     bool
		activeStackSize   int
		otherStackSize    int
		shouldClearWakeup bool
		shouldPreempt     bool
	}{
		{name: "Matching addr, not wakeable, first thread", futexAddr: wakeupAddr, targetVal: wakeupVal, traverseRight: false, activeStackSize: 3, otherStackSize: 0, shouldClearWakeup: true},
		{name: "Matching addr, wakeable, first thread", futexAddr: wakeupAddr, targetVal: wakeupVal + 1, traverseRight: false, activeStackSize: 3, otherStackSize: 0, shouldClearWakeup: true},
		{name: "Matching addr, not wakeable, last thread", futexAddr: wakeupAddr, targetVal: wakeupVal, traverseRight: true, activeStackSize: 1, otherStackSize: 2, shouldClearWakeup: true},
		{name: "Matching addr, wakeable, last thread", futexAddr: wakeupAddr, targetVal: wakeupVal + 1, traverseRight: true, activeStackSize: 1, otherStackSize: 2, shouldClearWakeup: true},
		{name: "Matching addr, not wakeable, intermediate thread", futexAddr: wakeupAddr, targetVal: wakeupVal, traverseRight: false, activeStackSize: 2, otherStackSize: 2, shouldClearWakeup: true},
		{name: "Matching addr, wakeable, intermediate thread", futexAddr: wakeupAddr, targetVal: wakeupVal + 1, traverseRight: true, activeStackSize: 2, otherStackSize: 2, shouldClearWakeup: true},
		{name: "Mismatched addr, last thread", futexAddr: wakeupAddr + 4, traverseRight: true, activeStackSize: 1, otherStackSize: 2, shouldPreempt: true, shouldClearWakeup: true},
		{name: "Mismatched addr", futexAddr: wakeupAddr + 4, traverseRight: true, activeStackSize: 2, otherStackSize: 2, shouldPreempt: true},
		{name: "Mismatched addr", futexAddr: wakeupAddr + 4, traverseRight: false, activeStackSize: 2, otherStackSize: 0, shouldPreempt: true},
		{name: "Mismatched addr", futexAddr: wakeupAddr + 4, traverseRight: false, activeStackSize: 1, otherStackSize: 0, shouldPreempt: true},
		{name: "Non-waiting thread", futexAddr: exec.FutexEmptyAddr, traverseRight: false, activeStackSize: 1, otherStackSize: 0, shouldPreempt: true},
		{name: "Non-waiting thread", futexAddr: exec.FutexEmptyAddr, traverseRight: true, activeStackSize: 2, otherStackSize: 1, shouldPreempt: true},
		{name: "Non-waiting thread, last thread", futexAddr: exec.FutexEmptyAddr, traverseRight: true, activeStackSize: 1, otherStackSize: 1, shouldPreempt: true, shouldClearWakeup: true},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goVm, state, contracts := setup(t, i*2000)
			mttestutil.SetupThreads(int64(i*101), state, c.traverseRight, c.activeStackSize, c.otherStackSize)
			step := state.Step

			state.Wakeup = wakeupAddr
			state.GetMemory().SetMemory(wakeupAddr, wakeupVal)
			activeThread := state.GetCurrentThread()
			activeThread.FutexAddr = c.futexAddr
			activeThread.FutexVal = c.targetVal
			activeThread.FutexTimeoutStep = exec.FutexNoTimeout

			// Set up post-state expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.Step += 1
			if c.shouldClearWakeup {
				expected.Wakeup = exec.FutexEmptyAddr
			}
			if c.shouldPreempt {
				// Just preempt the current thread
				expected.ExpectPreemption(state)
			}

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func TestEVM_WakeupTraversal_Full(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name        string
		threadCount int
	}{
		{"1 thread", 1},
		{"2 threads", 2},
		{"3 threads", 3},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Setup
			goVm, state, contracts := setup(t, i*789)
			mttestutil.SetupThreads(int64(i*2947), state, false, c.threadCount, 0)
			state.Wakeup = 0x08
			step := state.Step

			initialState := mttestutil.NewExpectedMTState(state)

			// Loop through all the threads to get back to the starting state
			iterations := c.threadCount * 2
			for i := 0; i < iterations; i++ {
				// Set up post-state expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.Step += 1
				expected.ExpectPreemption(state)

				// State transition
				var err error
				var stepWitness *mipsevm.StepWitness
				stepWitness, err = goVm.Step(true)
				require.NoError(t, err)

				// We should clear the wakeup on the last step
				if i == iterations-1 {
					expected.Wakeup = exec.FutexEmptyAddr
				}

				// Validate post-state
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
			}

			// We should be back to the original state with only a few modifications
			initialState.Step += uint64(iterations)
			initialState.StepsSinceLastContextSwitch = 0
			initialState.Wakeup = exec.FutexEmptyAddr
			initialState.Validate(t, state)
		})
	}
}

func TestEVM_SchedQuantumThreshold(t *testing.T) {
	var tracer *tracing.Hooks
	cases := []struct {
		name                        string
		stepsSinceLastContextSwitch uint64
		shouldPreempt               bool
	}{
		{name: "just under threshold", stepsSinceLastContextSwitch: exec.SchedQuantum - 1},
		{name: "at threshold", stepsSinceLastContextSwitch: exec.SchedQuantum, shouldPreempt: true},
		{name: "beyond threshold", stepsSinceLastContextSwitch: exec.SchedQuantum + 1, shouldPreempt: true},
	}

	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			goVm, state, contracts := setup(t, i*789)
			// Setup basic getThreadId syscall instruction
			state.Memory.SetMemory(state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = exec.SysGetTID // Set syscall number
			state.StepsSinceLastContextSwitch = c.stepsSinceLastContextSwitch
			step := state.Step

			// Set up post-state expectations
			expected := mttestutil.NewExpectedMTState(state)
			if c.shouldPreempt {
				expected.Step += 1
				expected.ExpectPreemption(state)
			} else {
				// Otherwise just expect a normal step
				expected.ExpectStep()
				expected.ActiveThread().Registers[2] = state.GetCurrentThread().ThreadId
				expected.ActiveThread().Registers[7] = 0
			}

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
		})
	}
}

func setup(t require.TestingT, randomSeed int) (mipsevm.FPVM, *multithreaded.State, *testutil.ContractMetadata) {
	v := GetMultiThreadedTestCase(t)
	vm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(randomSeed)))
	state := mttestutil.GetMtState(t, vm)

	return vm, state, v.Contracts

}
