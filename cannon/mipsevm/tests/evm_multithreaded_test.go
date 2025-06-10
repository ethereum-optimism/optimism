// These tests target architectures that are 64-bit or larger
package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mttestutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/register"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"
)

type Word = arch.Word

func TestEVM_MT_LL(t *testing.T) {
	// Set up some test values that will be reused
	posValue := uint64(0xAAAA_BBBB_1122_3344)
	posValueRet := uint64(0x1122_3344)
	negValue := uint64(0x1111_1111_8877_6655)
	negRetValue := uint64(0xFFFF_FFFF_8877_6655) // Sign extended version of negValue

	// Note: parameters are written as 64-bit values. For 32-bit architectures, these values are downcast to 32-bit
	cases := []struct {
		name         string
		base         uint64
		offset       int
		expectedAddr uint64
		memValue     uint64
		retVal       uint64
		rtReg        int
	}{
		{name: "Aligned addr", base: 0x01, offset: 0x0133, expectedAddr: 0x0134, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Aligned addr, negative value", base: 0x01, offset: 0x0133, expectedAddr: 0x0134, memValue: negValue, retVal: negRetValue, rtReg: 5},
		{name: "Aligned addr, addr signed extended", base: 0x01, offset: 0xFF33, expectedAddr: 0xFFFF_FFFF_FFFF_FF34, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Unaligned addr", base: 0xFF12_0001, offset: 0x3405, expectedAddr: 0xFF12_3406, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Unaligned addr, addr sign extended w overflow", base: 0xFF12_0001, offset: 0x8405, expectedAddr: 0xFF11_8406, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Return register set to 0", base: 0xFF12_0001, offset: 0x7404, expectedAddr: 0xFF12_7405, memValue: posValue, retVal: 0, rtReg: 0},
	}
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			for _, withExistingReservation := range []bool{true, false} {
				tName := fmt.Sprintf("%v (vm = %v, withExistingReservation = %v)", c.name, ver.Name, withExistingReservation)
				t.Run(tName, func(t *testing.T) {
					rtReg := c.rtReg
					baseReg := 6
					insn := uint32((0b11_0000 << 26) | (baseReg & 0x1F << 21) | (rtReg & 0x1F << 16) | (0xFFFF & c.offset))
					goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)), testutil.WithPCAndNextPC(0x40))
					state := mttestutil.GetMtState(t, goVm)
					step := state.GetStep()

					// Set up state
					testutil.SetMemoryUint64(t, state.GetMemory(), Word(c.expectedAddr), c.memValue)
					testutil.StoreInstruction(state.GetMemory(), state.GetPC(), insn)
					state.GetRegistersRef()[baseReg] = Word(c.base)
					if withExistingReservation {
						state.LLReservationStatus = multithreaded.LLStatusActive32bit
						state.LLAddress = Word(c.expectedAddr + 1)
						state.LLOwnerThread = 123
					} else {
						state.LLReservationStatus = multithreaded.LLStatusNone
						state.LLAddress = 0
						state.LLOwnerThread = 0
					}

					// Set up expectations
					expected := mttestutil.NewExpectedMTState(state)
					expected.ExpectStep()
					expected.LLReservationStatus = multithreaded.LLStatusActive32bit
					expected.LLAddress = Word(c.expectedAddr)
					expected.LLOwnerThread = state.GetCurrentThread().ThreadId
					if rtReg != 0 {
						expected.ActiveThread().Registers[rtReg] = Word(c.retVal)
					}

					stepWitness, err := goVm.Step(true)
					require.NoError(t, err)

					// Check expectations
					expected.Validate(t, state)
					testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
				})
			}
		}
	}
}

func TestEVM_MT_SC(t *testing.T) {
	// Set up some test values that will be reused
	memValue := uint64(0x1122_3344_5566_7788)

	llVariations := []struct {
		name                string
		llReservationStatus multithreaded.LLReservationStatus
		matchThreadId       bool
		matchAddr           bool
		shouldSucceed       bool
	}{
		{name: "should succeed", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchAddr: true, shouldSucceed: true},
		{name: "mismatch thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchAddr: true, shouldSucceed: false},
		{name: "mismatched addr", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchAddr: false, shouldSucceed: false},
		{name: "mismatched addr & thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchAddr: false, shouldSucceed: false},
		{name: "mismatched status", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, matchAddr: true, shouldSucceed: false},
		{name: "no active reservation", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, matchAddr: true, shouldSucceed: false},
	}

	// Note: Some parameters are written as 64-bit values. For 32-bit architectures, these values are downcast to 32-bit
	cases := []struct {
		name         string
		base         Word
		offset       int
		expectedAddr uint64
		storeValue   uint32
		rtReg        int
		threadId     Word
	}{
		{name: "Aligned addr", base: 0x01, offset: 0x0133, expectedAddr: 0x0134, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Aligned addr, signed extended", base: 0x01, offset: 0xFF33, expectedAddr: 0xFFFF_FFFF_FFFF_FF34, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Unaligned addr", base: 0xFF12_0001, offset: 0x3404, expectedAddr: 0xFF12_3405, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, sign extended w overflow", base: 0xFF12_0001, offset: 0x8404, expectedAddr: 0xFF_11_8405, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Return register set to 0", base: 0xFF12_0001, offset: 0x7403, expectedAddr: 0xFF12_7404, storeValue: 0xAABB_CCDD, rtReg: 0, threadId: 4},
	}
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			for _, llVar := range llVariations {
				tName := fmt.Sprintf("%v (%v,%v)", c.name, ver.Name, llVar.name)
				t.Run(tName, func(t *testing.T) {
					rtReg := c.rtReg
					baseReg := 6
					insn := uint32((0b11_1000 << 26) | (baseReg & 0x1F << 21) | (rtReg & 0x1F << 16) | (0xFFFF & c.offset))
					goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)))
					state := mttestutil.GetMtState(t, goVm)
					mttestutil.InitializeSingleThread(i*23456, state, i%2 == 1, testutil.WithPCAndNextPC(0x40))
					step := state.GetStep()

					// Define LL-related params
					var llAddress, llOwnerThread Word
					if llVar.matchAddr {
						llAddress = Word(c.expectedAddr)
					} else {
						llAddress = Word(c.expectedAddr) + 1
					}
					if llVar.matchThreadId {
						llOwnerThread = c.threadId
					} else {
						llOwnerThread = c.threadId + 1
					}

					// Setup state
					testutil.SetMemoryUint64(t, state.GetMemory(), Word(c.expectedAddr), memValue)
					state.GetCurrentThread().ThreadId = c.threadId
					testutil.StoreInstruction(state.GetMemory(), state.GetPC(), insn)
					state.GetRegistersRef()[baseReg] = c.base
					state.GetRegistersRef()[rtReg] = Word(c.storeValue)
					state.LLReservationStatus = llVar.llReservationStatus
					state.LLAddress = llAddress
					state.LLOwnerThread = llOwnerThread

					// Setup expectations
					expected := mttestutil.NewExpectedMTState(state)
					expected.ExpectStep()
					var retVal Word
					if llVar.shouldSucceed {
						retVal = 1
						expected.ExpectMemoryWriteUint32(t, Word(c.expectedAddr), c.storeValue)
						expected.LLReservationStatus = multithreaded.LLStatusNone
						expected.LLAddress = 0
						expected.LLOwnerThread = 0
					} else {
						retVal = 0
					}
					if rtReg != 0 {
						expected.ActiveThread().Registers[rtReg] = retVal
					}

					stepWitness, err := goVm.Step(true)
					require.NoError(t, err)

					// Check expectations
					expected.Validate(t, state)
					testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
				})
			}
		}
	}
}

func TestEVM_SysClone_FlagHandling(t *testing.T) {

	cases := []struct {
		name  string
		flags Word
		valid bool
	}{
		{"the supported flags bitmask", exec.ValidCloneFlags, true},
		{"no flags", 0, false},
		{"all flags", ^Word(0), false},
		{"all unsupported flags", ^Word(exec.ValidCloneFlags), false},
		{"a few supported flags", exec.CloneFs | exec.CloneSysvsem, false},
		{"one supported flag", exec.CloneFs, false},
		{"mixed supported and unsupported flags", exec.CloneFs | exec.CloneParentSettid, false},
		{"a single unsupported flag", exec.CloneUntraced, false},
		{"multiple unsupported flags", exec.CloneUntraced | exec.CloneParentSettid, false},
	}

	for _, c := range cases {
		c := c
		for _, version := range GetMipsVersionTestCases(t) {
			version := version
			t.Run(fmt.Sprintf("%v-%v", version.Name, c.name), func(t *testing.T) {
				state := multithreaded.CreateEmptyState()
				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = arch.SysClone // Set syscall number
				state.GetRegistersRef()[4] = c.flags       // Set first argument
				curStep := state.Step

				var err error
				var stepWitness *mipsevm.StepWitness
				goVm := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil, nil, versions.FeaturesForVersion(version.Version))
				if !c.valid {
					// The VM should exit
					stepWitness, err = goVm.Step(true)
					require.NoError(t, err)
					require.Equal(t, curStep+1, state.GetStep())
					require.Equal(t, true, goVm.GetState().GetExited())
					require.Equal(t, uint8(mipsevm.VMStatusPanic), goVm.GetState().GetExitCode())
					require.Equal(t, 1, state.ThreadCount())
				} else {
					stepWitness, err = goVm.Step(true)
					require.NoError(t, err)
					require.Equal(t, curStep+1, state.GetStep())
					require.Equal(t, false, goVm.GetState().GetExited())
					require.Equal(t, uint8(0), goVm.GetState().GetExitCode())
					require.Equal(t, 2, state.ThreadCount())
				}

				testutil.ValidateEVM(t, stepWitness, curStep, goVm, multithreaded.GetStateHashFn(), version.Contracts)
			})
		}
	}
}

func TestEVM_SysClone_Successful(t *testing.T) {
	cases := []struct {
		name          string
		traverseRight bool
	}{
		{"traverse left", false},
		{"traverse right", true},
	}

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				stackPtr := Word(100)

				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)))
				state := mttestutil.GetMtState(t, goVm)
				mttestutil.InitializeSingleThread(i*333, state, c.traverseRight)
				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = arch.SysClone        // the syscall number
				state.GetRegistersRef()[4] = exec.ValidCloneFlags // a0 - first argument, clone flags
				state.GetRegistersRef()[5] = stackPtr             // a1 - the stack pointer
				step := state.GetStep()

				// Sanity-check assumptions
				require.Equal(t, Word(1), state.NextThreadId)

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
				expectedNewThread.Registers[register.RegSyscallRet1] = 0
				expectedNewThread.Registers[register.RegSyscallErrno] = 0
				expectedNewThread.Registers[register.RegSP] = stackPtr

				var err error
				var stepWitness *mipsevm.StepWitness
				stepWitness, err = goVm.Step(true)
				require.NoError(t, err)

				expected.Validate(t, state)
				activeStack, inactiveStack := mttestutil.GetThreadStacks(state)
				require.Equal(t, 2, len(activeStack))
				require.Equal(t, 0, len(inactiveStack))
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}

func TestEVM_SysGetTID(t *testing.T) {
	cases := []struct {
		name     string
		threadId Word
	}{
		{"zero", 0},
		{"non-zero", 11},
	}

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*789)))
				state := mttestutil.GetMtState(t, goVm)
				mttestutil.InitializeSingleThread(i*789, state, false)

				state.GetCurrentThread().ThreadId = c.threadId
				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = arch.SysGetTID // Set syscall number
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
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}

func TestEVM_SysExit(t *testing.T) {
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

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				exitCode := uint8(3)

				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*133)))
				state := mttestutil.GetMtState(t, goVm)
				mttestutil.SetupThreads(int64(i*1111), state, i%2 == 0, c.threadCount, 0)

				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = arch.SysExit   // Set syscall number
				state.GetRegistersRef()[4] = Word(exitCode) // The first argument (exit code)
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
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}

func TestEVM_PopExitedThread(t *testing.T) {
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

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*133)))
				state := mttestutil.GetMtState(t, goVm)
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
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}

func TestEVM_SysFutex_WaitPrivate(t *testing.T) {
	// Note: parameters are written as 64-bit values. For 32-bit architectures, these values are downcast to 32-bit
	cases := []struct {
		name         string
		addressParam uint64
		effAddr      uint64
		targetValue  uint32
		actualValue  uint32
		timeout      uint64
		shouldFail   bool
	}{
		{name: "successful wait, no timeout", addressParam: 0xFF_FF_FF_FF_FF_FF_12_38, effAddr: 0xFF_FF_FF_FF_FF_FF_12_38, targetValue: 0xFF_FF_FF_01, actualValue: 0xFF_FF_FF_01},
		{name: "successful wait, no timeout, unaligned addr #1", addressParam: 0xFF_FF_FF_FF_FF_FF_12_33, effAddr: 0xFF_FF_FF_FF_FF_FF_12_30, targetValue: 0x01, actualValue: 0x01},
		{name: "successful wait, no timeout, unaligned addr #2", addressParam: 0xFF_FF_FF_FF_FF_FF_12_37, effAddr: 0xFF_FF_FF_FF_FF_FF_12_34, targetValue: 0x01, actualValue: 0x01},
		{name: "successful wait, no timeout, unaligned addr #3", addressParam: 0xFF_FF_FF_FF_FF_FF_12_3A, effAddr: 0xFF_FF_FF_FF_FF_FF_12_38, targetValue: 0x01, actualValue: 0x01},
		{name: "successful wait, no timeout, unaligned addr #4", addressParam: 0xFF_FF_FF_FF_FF_FF_12_3F, effAddr: 0xFF_FF_FF_FF_FF_FF_12_3C, targetValue: 0x01, actualValue: 0x01},
		{name: "memory mismatch, no timeout", addressParam: 0xFF_FF_FF_FF_FF_FF_12_00, effAddr: 0xFF_FF_FF_FF_FF_FF_12_00, targetValue: 0xFF_FF_FF_01, actualValue: 0xFF_FF_FF_02, shouldFail: true},
		{name: "memory mismatch, no timeout, unaligned", addressParam: 0xFF_FF_FF_FF_FF_FF_12_05, effAddr: 0xFF_FF_FF_FF_FF_FF_12_04, targetValue: 0x01, actualValue: 0x02, shouldFail: true},
		{name: "successful wait w timeout", addressParam: 0xFF_FF_FF_FF_FF_FF_12_38, effAddr: 0xFF_FF_FF_FF_FF_FF_12_38, targetValue: 0xFF_FF_FF_01, actualValue: 0xFF_FF_FF_01, timeout: 1000000},
		{name: "successful wait w timeout, unaligned", addressParam: 0xFF_FF_FF_FF_FF_FF_12_37, effAddr: 0xFF_FF_FF_FF_FF_FF_12_34, targetValue: 0xFF_FF_FF_01, actualValue: 0xFF_FF_FF_01, timeout: 1000000},
		{name: "memory mismatch w timeout", addressParam: 0xFF_FF_FF_FF_FF_FF_12_00, effAddr: 0xFF_FF_FF_FF_FF_FF_12_00, targetValue: 0xFF_FF_FF_F8, actualValue: 0xF8, timeout: 2000000, shouldFail: true},
		{name: "memory mismatch w timeout, unaligned", addressParam: 0xFF_FF_FF_FF_FF_FF_12_0F, effAddr: 0xFF_FF_FF_FF_FF_FF_12_0C, targetValue: 0xFF_FF_FF_01, actualValue: 0xFF_FF_FF_02, timeout: 2000000, shouldFail: true},
	}
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				rand := testutil.NewRandHelper(int64(i * 33))
				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*1234)), testutil.WithPCAndNextPC(0x04))
				state := mttestutil.GetMtState(t, goVm)
				step := state.GetStep()

				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				testutil.RandomizeWordAndSetUint32(state.GetMemory(), Word(c.effAddr), c.actualValue, int64(i+22))
				state.GetRegistersRef()[2] = arch.SysFutex // Set syscall number
				state.GetRegistersRef()[4] = Word(c.addressParam)
				state.GetRegistersRef()[5] = exec.FutexWaitPrivate
				// Randomize upper bytes of futex target
				state.GetRegistersRef()[6] = (rand.Word() & ^Word(0xFF_FF_FF_FF)) | Word(c.targetValue)
				state.GetRegistersRef()[7] = Word(c.timeout)

				// Setup expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.Step += 1
				expected.ActiveThread().PC = state.GetCpu().NextPC
				expected.ActiveThread().NextPC = state.GetCpu().NextPC + 4
				if c.shouldFail {
					expected.StepsSinceLastContextSwitch += 1
					expected.ActiveThread().Registers[2] = exec.MipsEAGAIN
					expected.ActiveThread().Registers[7] = exec.SysErrorSignal
				} else {
					// Return empty result and preempt thread
					expected.ActiveThread().Registers[2] = 0
					expected.ActiveThread().Registers[7] = 0
					expected.ExpectPreemption(state)
				}

				// State transition
				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Validate post-state
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}

func TestEVM_SysFutex_WakePrivate(t *testing.T) {
	// Note: parameters are written as 64-bit values. For 32-bit architectures, these values are downcast to 32-bit
	cases := []struct {
		name                string
		addressParam        uint64
		effAddr             uint64
		activeThreadCount   int
		inactiveThreadCount int
		traverseRight       bool
	}{
		{name: "Traverse right", addressParam: 0xFF_FF_FF_FF_FF_FF_67_00, effAddr: 0xFF_FF_FF_FF_FF_FF_67_00, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: true},
		{name: "Traverse right, unaligned addr #1", addressParam: 0xFF_FF_FF_FF_FF_FF_67_83, effAddr: 0xFF_FF_FF_FF_FF_FF_67_80, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: true},
		{name: "Traverse right, unaligned addr #2", addressParam: 0xFF_FF_FF_FF_FF_FF_67_87, effAddr: 0xFF_FF_FF_FF_FF_FF_67_84, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: true},
		{name: "Traverse right, unaligned addr #3", addressParam: 0xFF_FF_FF_FF_FF_FF_67_89, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: true},
		{name: "Traverse right, unaligned addr #4", addressParam: 0xFF_FF_FF_FF_FF_FF_67_8F, effAddr: 0xFF_FF_FF_FF_FF_FF_67_8C, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: true},
		{name: "Traverse right, no left threads", addressParam: 0xFF_FF_FF_FF_FF_FF_67_84, effAddr: 0xFF_FF_FF_FF_FF_FF_67_84, activeThreadCount: 2, inactiveThreadCount: 0, traverseRight: true},
		{name: "Traverse right, no left threads, unaligned addr", addressParam: 0xFF_FF_FF_FF_FF_FF_67_8E, effAddr: 0xFF_FF_FF_FF_FF_FF_67_8C, activeThreadCount: 2, inactiveThreadCount: 0, traverseRight: true},
		{name: "Traverse right, single thread", addressParam: 0xFF_FF_FF_FF_FF_FF_67_88, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 1, inactiveThreadCount: 0, traverseRight: true},
		{name: "Traverse right, single thread, unaligned", addressParam: 0xFF_FF_FF_FF_FF_FF_67_89, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 1, inactiveThreadCount: 0, traverseRight: true},
		{name: "Traverse left", addressParam: 0xFF_FF_FF_FF_FF_FF_67_88, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: false},
		{name: "Traverse left, unaliagned", addressParam: 0xFF_FF_FF_FF_FF_FF_67_89, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 2, inactiveThreadCount: 1, traverseRight: false},
		{name: "Traverse left, switch directions", addressParam: 0xFF_FF_FF_FF_FF_FF_67_88, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 1, inactiveThreadCount: 1, traverseRight: false},
		{name: "Traverse left, switch directions, unaligned", addressParam: 0xFF_FF_FF_FF_FF_FF_67_8F, effAddr: 0xFF_FF_FF_FF_FF_FF_67_8C, activeThreadCount: 1, inactiveThreadCount: 1, traverseRight: false},
		{name: "Traverse left, single thread", addressParam: 0xFF_FF_FF_FF_FF_FF_67_88, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 1, inactiveThreadCount: 0, traverseRight: false},
		{name: "Traverse left, single thread, unaligned", addressParam: 0xFF_FF_FF_FF_FF_FF_67_89, effAddr: 0xFF_FF_FF_FF_FF_FF_67_88, activeThreadCount: 1, inactiveThreadCount: 0, traverseRight: false},
	}
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*1122)))
				state := mttestutil.GetMtState(t, goVm)
				mttestutil.SetupThreads(int64(i*2244), state, c.traverseRight, c.activeThreadCount, c.inactiveThreadCount)
				step := state.Step

				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = arch.SysFutex // Set syscall number
				state.GetRegistersRef()[4] = Word(c.addressParam)
				state.GetRegistersRef()[5] = exec.FutexWakePrivate

				// Set up post-state expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.ExpectStep()
				expected.ActiveThread().Registers[2] = 0
				expected.ActiveThread().Registers[7] = 0
				expected.ExpectPreemption(state)

				// State transition
				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Validate post-state
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}

func TestEVM_SysFutex_UnsupportedOp(t *testing.T) {
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

	unsupportedFutexOps := map[string]Word{
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

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for name, op := range unsupportedFutexOps {
			testName := fmt.Sprintf("%v (%v)", name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(op)))
				state := mttestutil.GetMtState(t, goVm)
				step := state.GetStep()

				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = arch.SysFutex // Set syscall number
				state.GetRegistersRef()[5] = op

				// Setup expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.Step += 1
				expected.StepsSinceLastContextSwitch += 1
				expected.ActiveThread().PC = state.GetCpu().NextPC
				expected.ActiveThread().NextPC = state.GetCpu().NextPC + 4
				expected.ActiveThread().Registers[2] = exec.MipsEINVAL
				expected.ActiveThread().Registers[7] = exec.SysErrorSignal

				// State transition
				var err error
				var stepWitness *mipsevm.StepWitness
				stepWitness, err = goVm.Step(true)
				require.NoError(t, err)

				// Validate post-state
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}

func TestEVM_SysYield(t *testing.T) {
	runPreemptSyscall(t, "SysSchedYield", arch.SysSchedYield)
}

func TestEVM_SysNanosleep(t *testing.T) {
	runPreemptSyscall(t, "SysNanosleep", arch.SysNanosleep)
}

func runPreemptSyscall(t *testing.T, syscallName string, syscallNum uint32) {
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

	versions := GetMipsVersionTestCases(t)
	for _, ver := range versions {
		for i, c := range cases {
			for _, traverseRight := range []bool{true, false} {
				testName := fmt.Sprintf("%v: %v (vm = %v, traverseRight = %v)", syscallName, c.name, ver.Name, traverseRight)
				t.Run(testName, func(t *testing.T) {
					goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*789)))
					state := mttestutil.GetMtState(t, goVm)
					mttestutil.SetupThreads(int64(i*3259), state, traverseRight, c.activeThreads, c.inactiveThreads)

					testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
					state.GetRegistersRef()[2] = Word(syscallNum) // Set syscall number
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
					testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
				})
			}
		}
	}
}

func TestEVM_SysOpen(t *testing.T) {
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		t.Run(ver.Name, func(t *testing.T) {
			goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(5512)))
			state := mttestutil.GetMtState(t, goVm)

			testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = arch.SysOpen // Set syscall number
			step := state.Step

			// Set up post-state expectations
			expected := mttestutil.NewExpectedMTState(state)
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = exec.MipsEBADF
			expected.ActiveThread().Registers[7] = exec.SysErrorSignal

			// State transition
			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
		})
	}

}

func TestEVM_SysGetPID(t *testing.T) {
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		t.Run(ver.Name, func(t *testing.T) {
			goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(1929)))
			state := mttestutil.GetMtState(t, goVm)

			testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = arch.SysGetpid // Set syscall number
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
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
		})
	}
}

func TestEVM_SysClockGettimeMonotonic(t *testing.T) {
	testEVM_SysClockGettime(t, exec.ClockGettimeMonotonicFlag)
}

func TestEVM_SysClockGettimeRealtime(t *testing.T) {
	testEVM_SysClockGettime(t, exec.ClockGettimeRealtimeFlag)
}

func testEVM_SysClockGettime(t *testing.T, clkid Word) {
	llVariations := []struct {
		name                   string
		llReservationStatus    multithreaded.LLReservationStatus
		matchThreadId          bool
		matchEffAddr           bool
		matchEffAddr2          bool
		shouldClearReservation bool
	}{
		{name: "matching reservation", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchEffAddr: true, shouldClearReservation: true},
		{name: "matching reservation, 64-bit", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, matchEffAddr: true, shouldClearReservation: true},
		{name: "matching reservation, 2nd word", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchEffAddr2: true, shouldClearReservation: true},
		{name: "matching reservation, 2nd word, 64-bit", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, matchEffAddr2: true, shouldClearReservation: true},
		{name: "matching reservation, diff thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchEffAddr: true, shouldClearReservation: true},
		{name: "matching reservation, diff thread, 2nd word", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchEffAddr2: true, shouldClearReservation: true},
		{name: "mismatched reservation", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchEffAddr: false, shouldClearReservation: false},
		{name: "mismatched reservation, diff thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchEffAddr: false, shouldClearReservation: false},
		{name: "no reservation, matching addr", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, matchEffAddr: true, shouldClearReservation: true},
		{name: "no reservation, matching addr2", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, matchEffAddr2: true, shouldClearReservation: true},
		{name: "no reservation, mismatched addr", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, matchEffAddr: false, shouldClearReservation: false},
	}

	cases := []struct {
		name         string
		timespecAddr Word
	}{
		{"aligned timespec address", 0x1000},
		{"unaligned timespec address", 0x1003},
	}
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			for _, llVar := range llVariations {
				tName := fmt.Sprintf("%v (%v,%v)", c.name, ver.Name, llVar.name)
				t.Run(tName, func(t *testing.T) {
					goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(2101)))
					state := mttestutil.GetMtState(t, goVm)
					mttestutil.InitializeSingleThread(2101+i, state, i%2 == 1)
					effAddr := c.timespecAddr & arch.AddressMask
					effAddr2 := effAddr + arch.WordSizeBytes
					step := state.Step

					// Define LL-related params
					var llAddress, llOwnerThread Word
					if llVar.matchEffAddr {
						llAddress = effAddr
					} else if llVar.matchEffAddr2 {
						llAddress = effAddr2
					} else {
						llAddress = effAddr2 + 8
					}
					if llVar.matchThreadId {
						llOwnerThread = state.GetCurrentThread().ThreadId
					} else {
						llOwnerThread = state.GetCurrentThread().ThreadId + 1
					}

					testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
					state.GetRegistersRef()[2] = arch.SysClockGetTime // Set syscall number
					state.GetRegistersRef()[4] = clkid                // a0
					state.GetRegistersRef()[5] = c.timespecAddr       // a1
					state.LLReservationStatus = llVar.llReservationStatus
					state.LLAddress = llAddress
					state.LLOwnerThread = llOwnerThread

					expected := mttestutil.NewExpectedMTState(state)
					expected.ExpectStep()
					expected.ActiveThread().Registers[2] = 0
					expected.ActiveThread().Registers[7] = 0
					next := state.Step + 1
					var secs, nsecs Word
					if clkid == exec.ClockGettimeMonotonicFlag {
						secs = Word(next / exec.HZ)
						nsecs = Word((next % exec.HZ) * (1_000_000_000 / exec.HZ))
					}
					expected.ExpectMemoryWordWrite(effAddr, secs)
					expected.ExpectMemoryWordWrite(effAddr2, nsecs)
					if llVar.shouldClearReservation {
						expected.LLReservationStatus = multithreaded.LLStatusNone
						expected.LLAddress = 0
						expected.LLOwnerThread = 0
					}

					var err error
					var stepWitness *mipsevm.StepWitness
					stepWitness, err = goVm.Step(true)
					require.NoError(t, err)

					// Validate post-state
					expected.Validate(t, state)
					testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
				})
			}
		}
	}
}

func TestEVM_SysClockGettimeNonMonotonic(t *testing.T) {
	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		t.Run(ver.Name, func(t *testing.T) {
			goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(2101)))
			state := mttestutil.GetMtState(t, goVm)

			timespecAddr := Word(0x1000)
			testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
			state.GetRegistersRef()[2] = arch.SysClockGetTime // Set syscall number
			state.GetRegistersRef()[4] = 0xDEAD               // a0 - invalid clockid
			state.GetRegistersRef()[5] = timespecAddr         // a1
			step := state.Step

			expected := mttestutil.NewExpectedMTState(state)
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = exec.MipsEINVAL
			expected.ActiveThread().Registers[7] = exec.SysErrorSignal

			var err error
			var stepWitness *mipsevm.StepWitness
			stepWitness, err = goVm.Step(true)
			require.NoError(t, err)

			// Validate post-state
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
		})
	}
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
	"SysStat":          4106,
	"SysFstat":         4108,
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
	"SysGetRLimit":     4076,
	"SysLseek":         4019,
	"SysMunmap":        4091,
	"SysSetITimer":     4104,
	"SysTimerCreate":   4257,
	"SysTimerSetTime":  4258,
	"SysTimerDelete":   4261,
}

func TestEVM_EmptyThreadStacks(t *testing.T) {
	t.Parallel()
	var tracer *tracing.Hooks

	cases := []struct {
		name           string
		otherStackSize int
		traverseRight  bool
	}{
		{name: "Traverse right with empty stacks", otherStackSize: 0, traverseRight: true},
		{name: "Traverse left with empty stacks", otherStackSize: 0, traverseRight: false},
		{name: "Traverse right with one non-empty stack on the other side", otherStackSize: 1, traverseRight: true},
		{name: "Traverse left with one non-empty stack on the other side", otherStackSize: 1, traverseRight: false},
	}
	// Generate proof variations
	proofVariations := GenerateEmptyThreadProofVariations(t)

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			for _, proofCase := range proofVariations {
				testName := fmt.Sprintf("%v (vm=%v,proofCase=%v)", c.name, ver.Name, proofCase.Name)
				t.Run(testName, func(t *testing.T) {
					goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*123)))
					state := mttestutil.GetMtState(t, goVm)
					mttestutil.SetupThreads(int64(i*123), state, c.traverseRight, 0, c.otherStackSize)

					require.PanicsWithValue(t, "Active thread stack is empty", func() { _, _ = goVm.Step(false) })

					errorMessage := "active thread stack is empty"
					testutil.AssertEVMReverts(t, state, ver.Contracts, tracer, proofCase.Proof, testutil.CreateErrorStringMatcher(errorMessage))
				})
			}
		}
	}
}

func TestEVM_NormalTraversal_Full(t *testing.T) {
	cases := []struct {
		name        string
		threadCount int
	}{
		{"1 thread", 1},
		{"2 threads", 2},
		{"3 threads", 3},
	}

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			for _, traverseRight := range []bool{true, false} {
				testName := fmt.Sprintf("%v (vm = %v, traverseRight = %v)", c.name, ver.Name, traverseRight)
				t.Run(testName, func(t *testing.T) {
					// Setup
					goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*789)))
					state := mttestutil.GetMtState(t, goVm)
					mttestutil.SetupThreads(int64(i*2947), state, traverseRight, c.threadCount, 0)
					step := state.Step

					// Loop through all the threads to get back to the starting state
					iterations := c.threadCount * 2
					for i := 0; i < iterations; i++ {
						// Set up thread to yield
						testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
						state.GetRegistersRef()[2] = Word(arch.SysSchedYield)

						// Set up post-state expectations
						expected := mttestutil.NewExpectedMTState(state)
						expected.ActiveThread().Registers[2] = 0
						expected.ActiveThread().Registers[7] = 0
						expected.ExpectStep()
						expected.ExpectPreemption(state)

						// State transition
						var err error
						var stepWitness *mipsevm.StepWitness
						stepWitness, err = goVm.Step(true)
						require.NoError(t, err)

						// Validate post-state
						expected.Validate(t, state)
						testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
					}
				})
			}
		}
	}
}

func TestEVM_SchedQuantumThreshold(t *testing.T) {
	cases := []struct {
		name                        string
		stepsSinceLastContextSwitch uint64
		shouldPreempt               bool
	}{
		{name: "just under threshold", stepsSinceLastContextSwitch: exec.SchedQuantum - 1},
		{name: "at threshold", stepsSinceLastContextSwitch: exec.SchedQuantum, shouldPreempt: true},
		{name: "beyond threshold", stepsSinceLastContextSwitch: exec.SchedQuantum + 1, shouldPreempt: true},
	}

	vmVersions := GetMipsVersionTestCases(t)
	for _, ver := range vmVersions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, ver.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := ver.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i*789)))
				state := mttestutil.GetMtState(t, goVm)
				// Setup basic getThreadId syscall instruction
				testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = arch.SysGetTID // Set syscall number
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
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), ver.Contracts)
			})
		}
	}
}
