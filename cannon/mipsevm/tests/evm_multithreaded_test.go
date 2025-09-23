// These tests target architectures that are 64-bit or larger
package tests

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/register"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type Word = arch.Word

func TestEVM_MT_LL(t *testing.T) {
	type testVariation struct {
		name                    string
		withExistingReservation bool
	}
	testVariations := []testVariation{
		{"with existing reservation", true},
		{"without existing reservation", false},
	}

	// Set up some test values that will be reused
	posValue := uint64(0xAAAA_BBBB_1122_3344)
	posValueRet := uint64(0x1122_3344)
	negValue := uint64(0x1111_1111_8877_6655)
	negRetValue := uint64(0xFFFF_FFFF_8877_6655) // Sign extended version of negValue

	type baseTest struct {
		name         string
		base         uint64
		offset       int
		expectedAddr uint64
		memValue     uint64
		retVal       uint64
		rtReg        int
	}
	baseTests := []baseTest{
		{name: "Aligned addr", base: 0x01, offset: 0x0133, expectedAddr: 0x0134, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Aligned addr, negative value", base: 0x01, offset: 0x0133, expectedAddr: 0x0134, memValue: negValue, retVal: negRetValue, rtReg: 5},
		{name: "Aligned addr, addr signed extended", base: 0x01, offset: 0xFF33, expectedAddr: 0xFFFF_FFFF_FFFF_FF34, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Unaligned addr", base: 0xFF12_0001, offset: 0x3405, expectedAddr: 0xFF12_3406, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Unaligned addr, addr sign extended w overflow", base: 0xFF12_0001, offset: 0x8405, expectedAddr: 0xFF11_8406, memValue: posValue, retVal: posValueRet, rtReg: 5},
		{name: "Return register set to 0", base: 0xFF12_0001, offset: 0x7404, expectedAddr: 0xFF12_7405, memValue: posValue, retVal: 0, rtReg: 0},
	}

	type testCase = testutil.TestCaseVariation[baseTest, testVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v-%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, testVariations)

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base

		baseReg := 6
		insn := uint32((0b11_0000 << 26) | (baseReg & 0x1F << 21) | (c.rtReg & 0x1F << 16) | (0xFFFF & c.offset))

		// Set up state
		testutil.SetMemoryUint64(t, state.GetMemory(), Word(c.expectedAddr), c.memValue)
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetRegistersRef()[baseReg] = Word(c.base)
		if tt.Variation.withExistingReservation {
			state.LLReservationStatus = multithreaded.LLStatusActive32bit
			state.LLAddress = Word(c.expectedAddr + 1)
			state.LLOwnerThread = 123
		} else {
			state.LLReservationStatus = multithreaded.LLStatusNone
			state.LLAddress = 0
			state.LLOwnerThread = 0
		}
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		c := tt.Base
		expected.ExpectStep()
		expected.LLReservationStatus = multithreaded.LLStatusActive32bit
		expected.LLAddress = Word(c.expectedAddr)
		expected.LLOwnerThread = expected.ActiveThreadId()
		if c.rtReg != 0 {
			expected.ActiveThread().Registers[c.rtReg] = Word(c.retVal)
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithPCAndNextPC(0x40)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_MT_SC(t *testing.T) {
	type llVariation struct {
		name                string
		llReservationStatus multithreaded.LLReservationStatus
		matchThreadId       bool
		matchAddr           bool
		shouldSucceed       bool
	}
	llVariations := []llVariation{
		{name: "should succeed", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchAddr: true, shouldSucceed: true},
		{name: "mismatch thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchAddr: true, shouldSucceed: false},
		{name: "mismatched addr", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchAddr: false, shouldSucceed: false},
		{name: "mismatched addr & thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchAddr: false, shouldSucceed: false},
		{name: "mismatched status", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, matchAddr: true, shouldSucceed: false},
		{name: "no active reservation", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, matchAddr: true, shouldSucceed: false},
	}

	type baseTest struct {
		name         string
		base         Word
		offset       int
		expectedAddr uint64
		storeValue   uint32
		rtReg        int
		threadId     Word
	}
	baseTests := []baseTest{
		{name: "Aligned addr", base: 0x01, offset: 0x0133, expectedAddr: 0x0134, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Aligned addr, signed extended", base: 0x01, offset: 0xFF33, expectedAddr: 0xFFFF_FFFF_FFFF_FF34, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Unaligned addr", base: 0xFF12_0001, offset: 0x3404, expectedAddr: 0xFF12_3405, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, sign extended w overflow", base: 0xFF12_0001, offset: 0x8404, expectedAddr: 0xFF_11_8405, storeValue: 0xAABB_CCDD, rtReg: 5, threadId: 4},
		{name: "Return register set to 0", base: 0xFF12_0001, offset: 0x7403, expectedAddr: 0xFF12_7404, storeValue: 0xAABB_CCDD, rtReg: 0, threadId: 4},
	}

	type testCase = testutil.TestCaseVariation[baseTest, llVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v_%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, llVariations)

	// Set up some test values that will be reused
	memValue := uint64(0x1122_3344_5566_7788)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		llVar := tt.Variation

		traverseRight := r.Intn(2) == 1
		mtutil.InitializeSingleThread(r.Intn(10000), state, traverseRight, mtutil.WithPCAndNextPC(0x40))

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
		baseReg := 6
		insn := uint32((0b11_1000 << 26) | (baseReg & 0x1F << 21) | (c.rtReg & 0x1F << 16) | (0xFFFF & c.offset))
		testutil.SetMemoryUint64(t, state.GetMemory(), Word(c.expectedAddr), memValue)
		state.GetCurrentThread().ThreadId = c.threadId
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetRegistersRef()[baseReg] = c.base
		state.GetRegistersRef()[c.rtReg] = Word(c.storeValue)
		state.LLReservationStatus = llVar.llReservationStatus
		state.LLAddress = llAddress
		state.LLOwnerThread = llOwnerThread
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		c := tt.Base
		llVar := tt.Variation

		expected.ExpectStep()
		var retVal Word
		if llVar.shouldSucceed {
			retVal = 1
			expected.ExpectMemoryWriteUint32(t, Word(c.expectedAddr), c.storeValue)
			expected.ExpectMemoryReservationCleared()
		} else {
			retVal = 0
		}
		if c.rtReg != 0 {
			expected.ActiveThread().Registers[c.rtReg] = retVal
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases, SkipAutomaticMemoryReservationTests())
}

func TestEVM_SysClone_FlagHandling(t *testing.T) {
	type testCase struct {
		name  string
		flags Word
		valid bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
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

	stackPtr := Word(204)
	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		mtutil.InitializeSingleThread(r.Intn(10000), state, true)
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysClone // Set syscall number
		state.GetRegistersRef()[4] = c.flags       // Set first argument
		state.GetRegistersRef()[5] = stackPtr      // a1 - the stack pointer
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		if !c.valid {
			// The VM should exit
			expected.Step += 1
			expected.ExpectNoContextSwitch()
			expected.Exited = true
			expected.ExitCode = uint8(mipsevm.VMStatusPanic)
		} else {
			// Otherwise, we should clone the thread as normal
			setCloneExpectations(expected, stackPtr)
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SysClone_Successful(t *testing.T) {
	type testCase struct {
		name          string
		traverseRight bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{"traverse left", false},
		{"traverse right", true},
	}

	stackPtr := Word(100)
	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		mtutil.InitializeSingleThread(r.Intn(10000), state, c.traverseRight)
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysClone        // the syscall number
		state.GetRegistersRef()[4] = exec.ValidCloneFlags // a0 - first argument, clone flags
		state.GetRegistersRef()[5] = stackPtr             // a1 - the stack pointer

		// Sanity-check assumptions
		require.Equal(t, Word(1), state.NextThreadId)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		return setCloneExpectations(expected, stackPtr)
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

// setCloneExpectations sets state expectations assuming we start with 1 thread
func setCloneExpectations(expected *mtutil.ExpectedState, stackPointer Word) ExpectedExecResult {
	expected.Step += 1
	expectedNewThread := expected.ExpectNewThread()
	expected.ExpectActiveThreadId(expectedNewThread.ThreadId)
	expected.ExpectContextSwitch()

	// Original thread expectations
	prestateNextPC := expected.PrestateActiveThread().NextPC
	expected.PrestateActiveThread().PC = prestateNextPC
	expected.PrestateActiveThread().NextPC = prestateNextPC + 4
	expected.PrestateActiveThread().Registers[2] = 1
	expected.PrestateActiveThread().Registers[7] = 0
	// New thread expectations
	expectedNewThread.PC = prestateNextPC
	expectedNewThread.NextPC = prestateNextPC + 4
	expectedNewThread.ThreadId = 1
	expectedNewThread.Registers[register.RegSyscallRet1] = 0
	expectedNewThread.Registers[register.RegSyscallErrno] = 0
	expectedNewThread.Registers[register.RegSP] = stackPointer

	return ExpectNormalExecution()
}

func TestEVM_SysGetTID(t *testing.T) {
	type testCase struct {
		name     string
		threadId Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{"zero", 0},
		{"non-zero", 11},
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		mtutil.InitializeSingleThread(r.Intn(10000), state, false)
		state.GetCurrentThread().ThreadId = c.threadId
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysGetTID // Set syscall number
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = c.threadId
		expected.ActiveThread().Registers[7] = 0
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SysExit(t *testing.T) {
	type testVariation struct {
		name          string
		traverseRight bool
	}
	testVariations := []testVariation{
		{name: "traverse right", traverseRight: true},
		{name: "traverse left", traverseRight: false},
	}

	type baseTest struct {
		name               string
		threadCount        int
		shouldExitGlobally bool
	}
	baseTests := []baseTest{
		// If we exit the last thread, the whole process should exit
		{name: "one thread", threadCount: 1, shouldExitGlobally: true},
		{name: "two threads ", threadCount: 2},
		{name: "three threads ", threadCount: 3},
	}

	type testCase = testutil.TestCaseVariation[baseTest, testVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v-%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, testVariations)

	exitCode := uint8(3)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		mtutil.SetupThreads(r.Int64(10000), state, tt.Variation.traverseRight, c.threadCount, 0)
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysExit   // Set syscall number
		state.GetRegistersRef()[4] = Word(exitCode) // The first argument (exit code)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.Step += 1
		expected.ExpectNoContextSwitch()
		expected.ActiveThread().Exited = true
		expected.ActiveThread().ExitCode = exitCode
		if c.Base.shouldExitGlobally {
			expected.Exited = true
			expected.ExitCode = exitCode
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_PopExitedThread(t *testing.T) {
	type testCase struct {
		name                         string
		traverseRight                bool
		activeStackThreadCount       int
		expectTraverseRightPostState bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "traverse right", traverseRight: true, activeStackThreadCount: 2, expectTraverseRightPostState: true},
		{name: "traverse right, switch directions", traverseRight: true, activeStackThreadCount: 1, expectTraverseRightPostState: false},
		{name: "traverse left", traverseRight: false, activeStackThreadCount: 2, expectTraverseRightPostState: false},
		{name: "traverse left, switch directions", traverseRight: false, activeStackThreadCount: 1, expectTraverseRightPostState: true},
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		mtutil.SetupThreads(r.Int64(1000), state, c.traverseRight, c.activeStackThreadCount, 1)
		threadToPop := state.GetCurrentThread()
		threadToPop.Exited = true
		threadToPop.ExitCode = 1
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.Step += 1
		expected.ExpectPoppedThread()
		expected.ExpectContextSwitch()
		expected.ExpectTraverseRight(c.expectTraverseRightPostState)
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SysFutex_WaitPrivate(t *testing.T) {
	type testCase struct {
		name         string
		addressParam uint64
		effAddr      uint64
		targetValue  uint32
		actualValue  uint32
		timeout      uint64
		shouldFail   bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
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

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		testutil.RandomizeWordAndSetUint32(state.GetMemory(), Word(c.effAddr), c.actualValue, r.Int64(1000))
		state.GetRegistersRef()[2] = arch.SysFutex // Set syscall number
		state.GetRegistersRef()[4] = Word(c.addressParam)
		state.GetRegistersRef()[5] = exec.FutexWaitPrivate
		// Randomize upper bytes of futex target
		state.GetRegistersRef()[6] = (rand.Word() & ^Word(0xFF_FF_FF_FF)) | Word(c.targetValue)
		state.GetRegistersRef()[7] = Word(c.timeout)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.Step += 1
		expected.ActiveThread().PC = expected.ActiveThread().NextPC
		expected.ActiveThread().NextPC = expected.ActiveThread().NextPC + 4
		if c.shouldFail {
			expected.ExpectNoContextSwitch()
			expected.ActiveThread().Registers[2] = exec.MipsEAGAIN
			expected.ActiveThread().Registers[7] = exec.SysErrorSignal
		} else {
			// Return empty result and preempt thread
			expected.ActiveThread().Registers[2] = 0
			expected.ActiveThread().Registers[7] = 0
			expected.ExpectPreemption()
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SysFutex_WakePrivate(t *testing.T) {
	type testCase struct {
		name                string
		addressParam        uint64
		effAddr             uint64
		activeThreadCount   int
		inactiveThreadCount int
		traverseRight       bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
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

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		mtutil.SetupThreads(r.Int64(1000), state, c.traverseRight, c.activeThreadCount, c.inactiveThreadCount)
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysFutex // Set syscall number
		state.GetRegistersRef()[4] = Word(c.addressParam)
		state.GetRegistersRef()[5] = exec.FutexWakePrivate
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = 0
		expected.ActiveThread().Registers[7] = 0
		expected.ExpectPreemption()
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
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

	type testCase struct {
		name string
		op   Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{"FUTEX_WAIT", FUTEX_WAIT},
		{"FUTEX_WAKE", FUTEX_WAKE},
		{"FUTEX_FD", FUTEX_FD},
		{"FUTEX_REQUEUE", FUTEX_REQUEUE},
		{"FUTEX_CMP_REQUEUE", FUTEX_CMP_REQUEUE},
		{"FUTEX_WAKE_OP", FUTEX_WAKE_OP},
		{"FUTEX_LOCK_PI", FUTEX_LOCK_PI},
		{"FUTEX_UNLOCK_PI", FUTEX_UNLOCK_PI},
		{"FUTEX_TRYLOCK_PI", FUTEX_TRYLOCK_PI},
		{"FUTEX_WAIT_BITSET", FUTEX_WAIT_BITSET},
		{"FUTEX_WAKE_BITSET", FUTEX_WAKE_BITSET},
		{"FUTEX_WAIT_REQUEUE_PI", FUTEX_WAIT_REQUEUE_PI},
		{"FUTEX_CMP_REQUEUE_PI", FUTEX_CMP_REQUEUE_PI},
		{"FUTEX_LOCK_PI2", FUTEX_LOCK_PI2},
		{"FUTEX_REQUEUE_PRIVATE", (FUTEX_REQUEUE | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_CMP_REQUEUE_PRIVATE", (FUTEX_CMP_REQUEUE | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_WAKE_OP_PRIVATE", (FUTEX_WAKE_OP | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_LOCK_PI_PRIVATE", (FUTEX_LOCK_PI | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_LOCK_PI2_PRIVATE", (FUTEX_LOCK_PI2 | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_UNLOCK_PI_PRIVATE", (FUTEX_UNLOCK_PI | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_TRYLOCK_PI_PRIVATE", (FUTEX_TRYLOCK_PI | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_WAIT_BITSET_PRIVATE", (FUTEX_WAIT_BITSET | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_WAKE_BITSET_PRIVATE", (FUTEX_WAKE_BITSET | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_WAIT_REQUEUE_PI_PRIVATE", (FUTEX_WAIT_REQUEUE_PI | FUTEX_PRIVATE_FLAG)},
		{"FUTEX_CMP_REQUEUE_PI_PRIVATE", (FUTEX_CMP_REQUEUE_PI | FUTEX_PRIVATE_FLAG)},
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysFutex // Set syscall number
		state.GetRegistersRef()[5] = c.op
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = exec.MipsEINVAL
		expected.ActiveThread().Registers[7] = exec.SysErrorSignal
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SysYield(t *testing.T) {
	runPreemptSyscall(t, "SysSchedYield", arch.SysSchedYield)
}

func TestEVM_SysNanosleep(t *testing.T) {
	runPreemptSyscall(t, "SysNanosleep", arch.SysNanosleep)
}

func runPreemptSyscall(t *testing.T, syscallName string, syscallNum uint32) {
	type testVariation struct {
		name          string
		traverseRight bool
	}
	testVariations := []testVariation{
		{"Traverse right", true},
		{"Traverse left", false},
	}

	type baseTest struct {
		name            string
		activeThreads   int
		inactiveThreads int
	}
	baseTests := []baseTest{
		{name: "Last active thread", activeThreads: 1, inactiveThreads: 2},
		{name: "Only thread", activeThreads: 1, inactiveThreads: 0},
		{name: "Do not change directions", activeThreads: 2, inactiveThreads: 2},
		{name: "Do not change directions", activeThreads: 3, inactiveThreads: 0},
	}

	type testCase = testutil.TestCaseVariation[baseTest, testVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v-%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, testVariations)

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		mtutil.SetupThreads(r.Int64(1000), state, tt.Variation.traverseRight, c.activeThreads, c.inactiveThreads)
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = Word(syscallNum) // Set syscall number
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ExpectPreemption()
		expected.PrestateActiveThread().Registers[2] = 0
		expected.PrestateActiveThread().Registers[7] = 0
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SysOpen(t *testing.T) {
	initState := func(t require.TestingT, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysOpen // Set syscall number
	}

	setExpectations := func(t require.TestingT, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = exec.MipsEBADF
		expected.ActiveThread().Registers[7] = exec.SysErrorSignal
		return ExpectNormalExecution()
	}

	NewSimpleDiffTester().
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t)
}

func TestEVM_SysGetPID(t *testing.T) {
	initState := func(t require.TestingT, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysGetpid // Set syscall number
	}

	setExpectations := func(t require.TestingT, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = 0
		expected.ActiveThread().Registers[7] = 0
		return ExpectNormalExecution()
	}

	NewSimpleDiffTester().
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t)
}

func TestEVM_SysClockGettimeMonotonic(t *testing.T) {
	testEVM_SysClockGettime(t, exec.ClockGettimeMonotonicFlag)
}

func TestEVM_SysClockGettimeRealtime(t *testing.T) {
	testEVM_SysClockGettime(t, exec.ClockGettimeRealtimeFlag)
}

func testEVM_SysClockGettime(t *testing.T, clkid Word) {
	type llVariation struct {
		name                   string
		llReservationStatus    multithreaded.LLReservationStatus
		matchThreadId          bool
		matchEffAddr           bool
		matchEffAddr2          bool
		shouldClearReservation bool
	}
	llVariations := []llVariation{
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

	type baseTest struct {
		name         string
		timespecAddr Word
	}
	baseTests := []baseTest{
		{"aligned timespec address", 0x1000},
		{"unaligned timespec address", 0x1003},
	}

	type testCase = testutil.TestCaseVariation[baseTest, llVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v_%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, llVariations)

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		llVar := tt.Variation

		traverseRight := r.Intn(2) == 1
		mtutil.InitializeSingleThread(r.Intn(10000), state, traverseRight)
		effAddr := c.timespecAddr & arch.AddressMask
		effAddr2 := effAddr + arch.WordSizeBytes

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

		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysClockGetTime // Set syscall number
		state.GetRegistersRef()[4] = clkid                // a0
		state.GetRegistersRef()[5] = c.timespecAddr       // a1
		state.LLReservationStatus = llVar.llReservationStatus
		state.LLAddress = llAddress
		state.LLOwnerThread = llOwnerThread
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		c := tt.Base
		llVar := tt.Variation

		effAddr := c.timespecAddr & arch.AddressMask
		effAddr2 := effAddr + arch.WordSizeBytes

		expected.ExpectStep()
		incrementedStep := expected.Step
		expected.ActiveThread().Registers[2] = 0
		expected.ActiveThread().Registers[7] = 0
		var secs, nsecs Word
		if clkid == exec.ClockGettimeMonotonicFlag {
			secs = Word(incrementedStep / exec.HZ)
			nsecs = Word((incrementedStep % exec.HZ) * (1_000_000_000 / exec.HZ))
		}
		expected.ExpectMemoryWrite(effAddr, secs)
		expected.ExpectMemoryWrite(effAddr2, nsecs)
		if llVar.shouldClearReservation {
			expected.ExpectMemoryReservationCleared()
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases, SkipAutomaticMemoryReservationTests())
}

func TestEVM_SysClockGettimeNonMonotonic(t *testing.T) {
	initState := func(t require.TestingT, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		timespecAddr := Word(0x1000)
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysClockGetTime // Set syscall number
		state.GetRegistersRef()[4] = 0xDEAD               // a0 - invalid clockid
		state.GetRegistersRef()[5] = timespecAddr         // a1
	}

	setExpectations := func(t require.TestingT, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = exec.MipsEINVAL
		expected.ActiveThread().Registers[7] = exec.SysErrorSignal
		return ExpectNormalExecution()
	}

	NewSimpleDiffTester().
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t)
}

func TestEVM_EmptyThreadStacks(t *testing.T) {
	t.Parallel()

	proofVariations := GenerateEmptyThreadProofVariations(t)

	type baseTest struct {
		name           string
		otherStackSize int
		traverseRight  bool
	}
	baseTests := []baseTest{
		{name: "Traverse right with empty stacks", otherStackSize: 0, traverseRight: true},
		{name: "Traverse left with empty stacks", otherStackSize: 0, traverseRight: false},
		{name: "Traverse right with one non-empty stack on the other side", otherStackSize: 1, traverseRight: true},
		{name: "Traverse left with one non-empty stack on the other side", otherStackSize: 1, traverseRight: false},
	}

	type testCase = testutil.TestCaseVariation[baseTest, threadProofTestcase]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v-%v", tc.Base.name, tc.Variation.Name)
	}

	cases := testutil.TestVariations(baseTests, proofVariations)

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		b := c.Base
		mtutil.SetupThreads(r.Int64(1000), state, b.traverseRight, 0, b.otherStackSize)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		goPanic := "Active thread stack is empty"
		evmErr := "active thread stack is empty"
		return ExpectVmPanic(goPanic, evmErr, WithProofData(c.Variation.Proof))
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_NormalTraversal_Full(t *testing.T) {
	type testVariation struct {
		name          string
		traverseRight bool
	}
	testVariations := []testVariation{
		{"Traverse right", true},
		{"Traverse left", false},
	}

	type baseTest struct {
		name        string
		threadCount int
	}
	baseTests := []baseTest{
		{"1 thread", 1},
		{"2 threads", 2},
		{"3 threads", 3},
	}

	type testCase = testutil.TestCaseVariation[baseTest, testVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v-%v", tc.Base.name, tc.Variation.name)
	}

	syscallNumReg := 2
	// The ori (or immediate) instruction sets register 2 to SysSchedYield
	oriInsn := uint32((0b001101 << 26) | (syscallNumReg & 0x1F << 16) | (0xFFFF & arch.SysSchedYield))

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		traverseRight := tt.Variation.traverseRight
		mtutil.SetupThreads(r.Int64(1000), state, traverseRight, c.threadCount, 0)
		state.Step = 0

		// Set up each thread with a sequence of instructions
		threads, _ := mtutil.GetThreadStacks(state)
		for i := 0; i < c.threadCount; i++ {
			thread := threads[i]
			pc := thread.Cpu.PC
			// Each thread will be accessed twice
			for j := 0; j < 2; j++ {
				// First run the ori instruction to set the syscall register
				// Then run the syscall (yield)
				testutil.StoreInstruction(state.Memory, pc, oriInsn)
				testutil.StoreInstruction(state.Memory, pc+4, syscallInsn)
				pc += 8
			}
		}
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		if expected.Step%2 == 0 {
			// Even instructions will be the "or immediate" insn that sets our yield syscall num
			expected.ExpectStep()
			expected.ActiveThread().Registers[syscallNumReg] = arch.SysSchedYield
		} else {
			// Odd instructions will cause a yield
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = 0
			expected.ActiveThread().Registers[7] = 0
			expected.ExpectPreemption()
		}
		return ExpectNormalExecution()
	}

	diffTester := NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations)

	for _, bt := range baseTests {
		// Loop through all the threads to get back to the starting state
		// We want to loop 2x for each thread, where each loop takes 2 instructions
		steps := bt.threadCount * 4

		cases := testutil.TestVariations([]baseTest{bt}, testVariations)
		diffTester.Run(t, cases, WithSteps(steps))
	}
}

func TestEVM_SchedQuantumThreshold(t *testing.T) {
	type testCase struct {
		name                        string
		stepsSinceLastContextSwitch uint64
		shouldPreempt               bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "just under threshold", stepsSinceLastContextSwitch: exec.SchedQuantum - 1},
		{name: "at threshold", stepsSinceLastContextSwitch: exec.SchedQuantum, shouldPreempt: true},
		{name: "beyond threshold", stepsSinceLastContextSwitch: exec.SchedQuantum + 1, shouldPreempt: true},
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		// Setup basic getThreadId syscall instruction
		testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysGetTID // Set syscall number
		state.StepsSinceLastContextSwitch = c.stepsSinceLastContextSwitch
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		if c.shouldPreempt {
			expected.Step += 1
			expected.ExpectPreemption()
		} else {
			// Otherwise just expect a normal step
			expected.ExpectStep()
			expected.ActiveThread().Registers[2] = expected.ActiveThreadId()
			expected.ActiveThread().Registers[7] = 0
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}
