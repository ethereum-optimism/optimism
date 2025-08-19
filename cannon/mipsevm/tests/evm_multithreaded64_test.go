// These tests target architectures that are 64-bit or larger
package tests

import (
	"encoding/binary"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/maps"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

func TestEVM_MT64_LL(t *testing.T) {
	type llVariation struct {
		name                    string
		withExistingReservation bool
	}
	llVariations := []llVariation{
		{"with existing reservation", true},
		{"without existing reservation", false},
	}

	type baseTest struct {
		name   string
		base   Word
		offset int
		addr   Word
		memVal Word
		retReg int
		retVal Word
	}

	memVal := Word(0x11223344_55667788)
	memValNeg := Word(0xF1223344_F5667788)
	baseTests := []baseTest{
		{name: "8-byte-aligned addr", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "8-byte-aligned addr, neg value", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memValNeg, retVal: 0xFFFFFFFF_F1223344, retReg: 5},
		{name: "8-byte-aligned addr, extra bits", base: 0x01, offset: 0x0109, addr: 0x010A, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "8-byte-aligned addr, addr signed extended", base: 0x01, offset: 0xFF37, addr: 0xFFFF_FFFF_FFFF_FF38, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "8-byte-aligned addr, addr signed extended w overflow", base: 0x1000_0001, offset: 0xFF07, addr: 0x0000_0000_0FFF_FF08, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "4-byte-aligned addr", base: 0x01, offset: 0x0103, addr: 0x0104, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "4-byte-aligned addr, neg value", base: 0x01, offset: 0x0104, addr: 0x0105, memVal: memValNeg, retVal: 0xFFFFFFFF_F5667788, retReg: 5},
		{name: "4-byte-aligned addr, extra bits", base: 0x01, offset: 0x0105, addr: 0x0106, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "4-byte-aligned addr, addr signed extended", base: 0x01, offset: 0xFF33, addr: 0xFFFF_FFFF_FFFF_FF34, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "4-byte-aligned addr, addr signed extended w overflow", base: 0x1000_0001, offset: 0xFF03, addr: 0x0000_0000_0FFF_FF04, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "Return register set to 0", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memVal, retVal: 0x11223344, retReg: 0},
	}

	type testCase = testutil.TestCaseVariation[baseTest, llVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v-%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, llVariations)

	initState := func(t require.TestingT, testCase testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := testCase.Base
		retReg := c.retReg
		baseReg := 6
		insn := uint32((0b11_0000 << 26) | (baseReg & 0x1F << 21) | (retReg & 0x1F << 16) | (0xFFFF & c.offset))

		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetMemory().SetWord(testutil.EffAddr(c.addr), c.memVal)
		state.GetRegistersRef()[baseReg] = c.base
		if testCase.Variation.withExistingReservation {
			state.LLReservationStatus = multithreaded.LLStatusActive32bit
			state.LLAddress = c.addr + 1
			state.LLOwnerThread = 123
		} else {
			state.LLReservationStatus = multithreaded.LLStatusNone
			state.LLAddress = 0
			state.LLOwnerThread = 0
		}
	}

	setExpectations := func(t require.TestingT, testCase testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.LLReservationStatus = multithreaded.LLStatusActive32bit
		expected.LLAddress = testCase.Base.addr
		expected.LLOwnerThread = expected.ActiveThreadId()

		retReg := testCase.Base.retReg
		if retReg != 0 {
			expected.ActiveThread().Registers[retReg] = testCase.Base.retVal
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithPCAndNextPC(0x40)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_MT64_SC(t *testing.T) {
	type llVariation struct {
		name                string
		llReservationStatus multithreaded.LLReservationStatus
		matchThreadId       bool
		matchAddr           bool
		shouldSucceed       bool
	}
	llVariations := []llVariation{
		{name: "should succeed", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchAddr: true, shouldSucceed: true},
		{name: "mismatch addr", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchAddr: true, shouldSucceed: false},
		{name: "mismatched thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchAddr: false, shouldSucceed: false},
		{name: "mismatched addr & thread", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: false, matchAddr: false, shouldSucceed: false},
		{name: "mismatched reservation status", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, matchAddr: true, shouldSucceed: false},
		{name: "no active reservation", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, matchAddr: true, shouldSucceed: false},
	}

	type baseTest struct {
		name           string
		base           Word
		offset         int
		addr           Word
		value          Word
		expectedMemVal Word
		rtReg          int
		threadId       Word
	}
	baseTests := []baseTest{
		{name: "8-byte-aligned addr", base: 0x01, offset: 0x0137, addr: 0x0138, value: 0xABCD, expectedMemVal: 0xABCD_0000_0000, rtReg: 5, threadId: 4},
		{name: "8-byte-aligned addr, extra bits", base: 0x01, offset: 0x0138, addr: 0x0139, value: 0xABCD, expectedMemVal: 0xABCD_0000_0000, rtReg: 5, threadId: 4},
		{name: "8-byte-aligned addr, signed extended", base: 0x01, offset: 0xFF37, addr: 0xFFFF_FFFF_FFFF_FF38, value: 0xABCD, expectedMemVal: 0xABCD_0000_0000, rtReg: 5, threadId: 4},
		{name: "8-byte-aligned addr, signed extended w overflow", base: 0x1000_0001, offset: 0xFF37, addr: 0x0FFF_FF38, value: 0xABCD, expectedMemVal: 0xABCD_0000_0000, rtReg: 5, threadId: 4},
		{name: "4-byte-aligned addr", base: 0x01, offset: 0x0133, addr: 0x0134, value: 0xABCD, expectedMemVal: 0x_0000_0000_0000_ABCD, rtReg: 5, threadId: 4},
		{name: "4-byte-aligned addr, extra bits", base: 0x01, offset: 0x0134, addr: 0x0135, value: 0xABCD, expectedMemVal: 0x_0000_0000_0000_ABCD, rtReg: 5, threadId: 4},
		{name: "4-byte-aligned addr, signed extended", base: 0x01, offset: 0xFF33, addr: 0xFFFF_FFFF_FFFF_FF34, value: 0xABCD, expectedMemVal: 0x_0000_0000_0000_ABCD, rtReg: 5, threadId: 4},
		{name: "4-byte-aligned addr, signed extended w overflow", base: 0x1000_0001, offset: 0xFF33, addr: 0x0FFF_FF34, value: 0xABCD, expectedMemVal: 0x_0000_0000_0000_ABCD, rtReg: 5, threadId: 4},
		{name: "Return register set to 0", base: 0x01, offset: 0x0138, addr: 0x0139, value: 0xABCD, expectedMemVal: 0xABCD_0000_0000, rtReg: 0, threadId: 4},
		{name: "Zero valued ll args", base: 0x0, offset: 0x0, value: 0xABCD, expectedMemVal: 0xABCD_0000_0000, rtReg: 5, threadId: 0},
	}

	type testCase = testutil.TestCaseVariation[baseTest, llVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v_%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, llVariations)

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		llVar := tt.Variation

		rtReg := c.rtReg
		baseReg := 6
		insn := uint32((0b11_1000 << 26) | (baseReg & 0x1F << 21) | (rtReg & 0x1F << 16) | (0xFFFF & c.offset))

		traverseRight := r.Intn(2) == 1
		mtutil.InitializeSingleThread(r.Intn(10000), state, traverseRight, mtutil.WithPCAndNextPC(0x40))

		// Define LL-related params
		var llAddress, llOwnerThread Word
		if llVar.matchAddr {
			llAddress = c.addr
		} else {
			llAddress = c.addr + 1
		}
		if llVar.matchThreadId {
			llOwnerThread = c.threadId
		} else {
			llOwnerThread = c.threadId + 1
		}

		// Setup state
		state.GetCurrentThread().ThreadId = c.threadId
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetRegistersRef()[baseReg] = c.base
		state.GetRegistersRef()[rtReg] = c.value
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
			expected.ExpectMemoryWrite(testutil.EffAddr(c.addr), c.expectedMemVal)
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

func TestEVM_MT64_LLD(t *testing.T) {
	type llVariation struct {
		name                    string
		withExistingReservation bool
	}
	llVariations := []llVariation{
		{"with existing reservation", true},
		{"without existing reservation", false},
	}

	memVal := Word(0x11223344_55667788)
	memValNeg := Word(0xF1223344_F5667788)
	type baseTest struct {
		name   string
		base   Word
		offset int
		addr   Word
		memVal Word
		retReg int
	}
	baseTests := []baseTest{
		{name: "Aligned addr", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memVal, retReg: 5},
		{name: "Aligned addr, neg value", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memValNeg, retReg: 5},
		{name: "Unaligned addr, offset=1", base: 0x01, offset: 0x0100, addr: 0x0101, memVal: memVal, retReg: 5},
		{name: "Unaligned addr, offset=2", base: 0x02, offset: 0x0100, addr: 0x0102, memVal: memVal, retReg: 5},
		{name: "Unaligned addr, offset=3", base: 0x03, offset: 0x0100, addr: 0x0103, memVal: memVal, retReg: 5},
		{name: "Unaligned addr, offset=4", base: 0x04, offset: 0x0100, addr: 0x0104, memVal: memVal, retReg: 5},
		{name: "Unaligned addr, offset=5", base: 0x05, offset: 0x0100, addr: 0x0105, memVal: memVal, retReg: 5},
		{name: "Unaligned addr, offset=6", base: 0x06, offset: 0x0100, addr: 0x0106, memVal: memVal, retReg: 5},
		{name: "Unaligned addr, offset=7", base: 0x07, offset: 0x0100, addr: 0x0107, memVal: memVal, retReg: 5},
		{name: "Aligned addr, signed extended", base: 0x01, offset: 0xFF37, addr: 0xFFFF_FFFF_FFFF_FF38, memVal: memVal, retReg: 5},
		{name: "Aligned addr, signed extended w overflow", base: 0x1000_0001, offset: 0xFF07, addr: 0x0000_0000_0FFF_FF08, memVal: memVal, retReg: 5},
		{name: "Return register set to 0", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memVal, retReg: 0},
	}

	type testCase = testutil.TestCaseVariation[baseTest, llVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v-%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, llVariations)

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		baseReg := 6
		insn := uint32((0b11_0100 << 26) | (baseReg & 0x1F << 21) | (c.retReg & 0x1F << 16) | (0xFFFF & c.offset))

		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetMemory().SetWord(testutil.EffAddr(c.addr), c.memVal)
		state.GetRegistersRef()[baseReg] = c.base
		if tt.Variation.withExistingReservation {
			state.LLReservationStatus = multithreaded.LLStatusActive64bit
			state.LLAddress = c.addr + 1
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
		expected.LLReservationStatus = multithreaded.LLStatusActive64bit
		expected.LLAddress = c.addr
		expected.LLOwnerThread = expected.ActiveThreadId()
		if c.retReg != 0 {
			expected.ActiveThread().Registers[c.retReg] = c.memVal
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithPCAndNextPC(0x40)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_MT64_SCD(t *testing.T) {
	type llVariation struct {
		name                string
		llReservationStatus multithreaded.LLReservationStatus
		matchThreadId       bool
		matchAddr           bool
		shouldSucceed       bool
	}
	llVariations := []llVariation{
		{name: "should succeed", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, matchAddr: true, shouldSucceed: true},
		{name: "mismatch addr", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: false, matchAddr: true, shouldSucceed: false},
		{name: "mismatched thread", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: true, matchAddr: false, shouldSucceed: false},
		{name: "mismatched addr & thread", llReservationStatus: multithreaded.LLStatusActive64bit, matchThreadId: false, matchAddr: false, shouldSucceed: false},
		{name: "mismatched status", llReservationStatus: multithreaded.LLStatusActive32bit, matchThreadId: true, matchAddr: true, shouldSucceed: false},
		{name: "no active reservation", llReservationStatus: multithreaded.LLStatusNone, matchThreadId: true, matchAddr: true, shouldSucceed: false},
	}

	type baseTest struct {
		name     string
		base     Word
		offset   int
		addr     Word
		rtReg    int
		threadId Word
	}
	baseTests := []baseTest{
		{name: "Aligned addr", base: 0x01, offset: 0x0137, addr: 0x0138, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, offset=1", base: 0x01, offset: 0x0100, addr: 0x0101, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, offset=2", base: 0x02, offset: 0x0100, addr: 0x0102, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, offset=3", base: 0x03, offset: 0x0100, addr: 0x0103, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, offset=4", base: 0x04, offset: 0x0100, addr: 0x0104, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, offset=5", base: 0x05, offset: 0x0100, addr: 0x0105, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, offset=6", base: 0x06, offset: 0x0100, addr: 0x0106, rtReg: 5, threadId: 4},
		{name: "Unaligned addr, offset=7", base: 0x07, offset: 0x0100, addr: 0x0107, rtReg: 5, threadId: 4},
		{name: "Aligned addr, signed extended", base: 0x01, offset: 0xFF37, addr: 0xFFFF_FFFF_FFFF_FF38, rtReg: 5, threadId: 4},
		{name: "Aligned addr, signed extended w overflow", base: 0x1000_0001, offset: 0xFF37, addr: 0x0FFF_FF38, rtReg: 5, threadId: 4},
		{name: "Return register set to 0", base: 0x01, offset: 0x0138, addr: 0x0139, rtReg: 0, threadId: 4},
		{name: "Zero valued ll args", base: 0x0, offset: 0x0, rtReg: 5, threadId: 0},
	}

	type testCase = testutil.TestCaseVariation[baseTest, llVariation]
	testNamer := func(tc testCase) string {
		return fmt.Sprintf("%v_%v", tc.Base.name, tc.Variation.name)
	}
	cases := testutil.TestVariations(baseTests, llVariations)

	value := Word(0x11223344_55667788)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		c := tt.Base
		llVar := tt.Variation

		traverseRight := r.Intn(2) == 1
		mtutil.InitializeSingleThread(r.Intn(10000), state, traverseRight, mtutil.WithPCAndNextPC(0x40))

		// Define LL-related params
		var llAddress, llOwnerThread Word
		if llVar.matchAddr {
			llAddress = c.addr
		} else {
			llAddress = c.addr + 1
		}
		if llVar.matchThreadId {
			llOwnerThread = c.threadId
		} else {
			llOwnerThread = c.threadId + 1
		}

		// Setup state
		baseReg := 6
		insn := uint32((0b11_1100 << 26) | (baseReg & 0x1F << 21) | (c.rtReg & 0x1F << 16) | (0xFFFF & c.offset))

		state.GetCurrentThread().ThreadId = c.threadId
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetRegistersRef()[baseReg] = c.base
		state.GetRegistersRef()[c.rtReg] = value
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
			expected.ExpectMemoryWrite(testutil.EffAddr(c.addr), value)
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

func TestEVM_MT_SysRead_Preimage64(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		addr           Word
		count          Word
		writeLen       Word
		preimageOffset Word
		prestateMem    Word
		postateMem     Word
		shouldPanic    bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	prestateMem := Word(0xEE_EE_EE_EE_FF_FF_FF_FF)
	cases := []testCase{
		{name: "Aligned addr, write 1 byte", addr: 0x00_00_FF_00, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_EE_EE_EE_FF_FF_FF_FF},
		{name: "Aligned addr, write 2 byte", addr: 0x00_00_FF_00, count: 2, writeLen: 2, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_EE_EE_FF_FF_FF_FF},
		{name: "Aligned addr, write 3 byte", addr: 0x00_00_FF_00, count: 3, writeLen: 3, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_56_EE_FF_FF_FF_FF},
		{name: "Aligned addr, write 4 byte", addr: 0x00_00_FF_00, count: 4, writeLen: 4, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_56_78_FF_FF_FF_FF},
		{name: "Aligned addr, write 5 byte", addr: 0x00_00_FF_00, count: 5, writeLen: 5, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_56_78_98_FF_FF_FF},
		{name: "Aligned addr, write 6 byte", addr: 0x00_00_FF_00, count: 6, writeLen: 6, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_56_78_98_76_FF_FF},
		{name: "Aligned addr, write 7 byte", addr: 0x00_00_FF_00, count: 7, writeLen: 7, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_56_78_98_76_54_FF},
		{name: "Aligned addr, write 8 byte", addr: 0x00_00_FF_00, count: 8, writeLen: 8, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_56_78_98_76_54_32},

		{name: "1-byte misaligned addr, write 1 byte", addr: 0x00_00_FF_01, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_12_EE_EE_FF_FF_FF_FF},
		{name: "1-byte misaligned addr, write 2 byte", addr: 0x00_00_FF_01, count: 2, writeLen: 2, preimageOffset: 9, prestateMem: prestateMem, postateMem: 0xEE_34_56_EE_FF_FF_FF_FF},
		{name: "1-byte misaligned addr, write 3 byte", addr: 0x00_00_FF_01, count: 3, writeLen: 3, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_12_34_56_FF_FF_FF_FF},
		{name: "1-byte misaligned addr, write 4 byte", addr: 0x00_00_FF_01, count: 4, writeLen: 4, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_12_34_56_78_FF_FF_FF},
		{name: "1-byte misaligned addr, write 5 byte", addr: 0x00_00_FF_01, count: 5, writeLen: 5, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_12_34_56_78_98_FF_FF},
		{name: "1-byte misaligned addr, write 6 byte", addr: 0x00_00_FF_01, count: 6, writeLen: 6, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_12_34_56_78_98_76_FF},
		{name: "1-byte misaligned addr, write 7 byte", addr: 0x00_00_FF_01, count: 7, writeLen: 7, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_12_34_56_78_98_76_54},

		{name: "2-byte misaligned addr, write 1 byte", addr: 0x00_00_FF_02, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_EE_12_EE_FF_FF_FF_FF},
		{name: "2-byte misaligned addr, write 2 byte", addr: 0x00_00_FF_02, count: 2, writeLen: 2, preimageOffset: 12, prestateMem: prestateMem, postateMem: 0xEE_EE_98_76_FF_FF_FF_FF},
		{name: "2-byte misaligned addr, write 2 byte", addr: 0x00_00_FF_02, count: 3, writeLen: 3, preimageOffset: 12, prestateMem: prestateMem, postateMem: 0xEE_EE_98_76_54_FF_FF_FF},
		{name: "2-byte misaligned addr, write 2 byte", addr: 0x00_00_FF_02, count: 4, writeLen: 4, preimageOffset: 12, prestateMem: prestateMem, postateMem: 0xEE_EE_98_76_54_32_FF_FF},

		{name: "3-byte misaligned addr, write 1 byte", addr: 0x00_00_FF_03, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_12_FF_FF_FF_FF},
		{name: "4-byte misaligned addr, write 1 byte", addr: 0x00_00_FF_04, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_EE_12_FF_FF_FF},
		{name: "5-byte misaligned addr, write 1 byte", addr: 0x00_00_FF_05, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_EE_FF_12_FF_FF},
		{name: "6-byte misaligned addr, write 1 byte", addr: 0x00_00_FF_06, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_EE_FF_FF_12_FF},
		{name: "7-byte misaligned addr, write 1 byte", addr: 0x00_00_FF_07, count: 1, writeLen: 1, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_EE_FF_FF_FF_12},

		{name: "Count of 0", addr: 0x00_00_FF_03, count: 0, writeLen: 0, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_EE_FF_FF_FF_FF},
		{name: "Count greater than 8", addr: 0x00_00_FF_00, count: 15, writeLen: 8, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0x12_34_56_78_98_76_54_32},
		{name: "Count greater than 8, unaligned", addr: 0x00_00_FF_01, count: 15, writeLen: 7, preimageOffset: 8, prestateMem: prestateMem, postateMem: 0xEE_12_34_56_78_98_76_54},
		{name: "Offset at last byte", addr: 0x00_00_FF_00, count: 8, writeLen: 1, preimageOffset: 15, prestateMem: prestateMem, postateMem: 0x32_EE_EE_EE_FF_FF_FF_FF},
		{name: "Offset just out of bounds", addr: 0x00_00_FF_00, count: 4, writeLen: 0, preimageOffset: 16, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_EE_FF_FF_FF_FF, shouldPanic: true},
		{name: "Offset out of bounds", addr: 0x00_00_FF_00, count: 4, writeLen: 0, preimageOffset: 17, prestateMem: prestateMem, postateMem: 0xEE_EE_EE_EE_FF_FF_FF_FF, shouldPanic: true},
	}

	preimageValue := make([]byte, 0, 8)
	preimageValue = binary.BigEndian.AppendUint32(preimageValue, 0x12_34_56_78)
	preimageValue = binary.BigEndian.AppendUint32(preimageValue, 0x98_76_54_32)
	initState := func(t require.TestingT, testCase testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.PreimageKey = testutil.Keccak256Preimage(preimageValue)
		state.PreimageOffset = testCase.preimageOffset
		state.GetRegistersRef()[2] = arch.SysRead
		state.GetRegistersRef()[4] = exec.FdPreimageRead
		state.GetRegistersRef()[5] = testCase.addr
		state.GetRegistersRef()[6] = testCase.count
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetMemory().SetWord(testutil.EffAddr(testCase.addr), testCase.prestateMem)
	}

	setExpectations := func(t require.TestingT, testCase testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = testCase.writeLen
		expected.ActiveThread().Registers[7] = 0 // no error
		expected.PreimageOffset += testCase.writeLen
		expected.ExpectMemoryWrite(testutil.EffAddr(testCase.addr), testCase.postateMem)

		if testCase.shouldPanic {
			preimageKey := testutil.Keccak256Preimage(preimageValue)
			return ExpectPreimageOraclePanic(preimageKey, preimageValue, testCase.preimageOffset, "Preimage offset out-of-bounds")
		} else {
			return ExpectNormalExecution()
		}
	}

	po := func() mipsevm.PreimageOracle { return testutil.StaticOracle(t, preimageValue) }
	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases, WithPreimageOracle(po))
}

func TestEVM_MT_SysReadWrite_WithEventFd(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name       string
		syscallNum Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "SysRead", syscallNum: arch.SysRead},
		{name: "SysWrite", syscallNum: arch.SysWrite},
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		addr := Word(0x00_00_FF_00)
		state.GetRegistersRef()[2] = tt.syscallNum
		state.GetRegistersRef()[4] = exec.FdEventFd
		state.GetRegistersRef()[5] = addr
		state.GetRegistersRef()[6] = 1
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		// Set a memory value to ensure that memory at the target address is not modified
		state.GetMemory().SetWord(addr, Word(0x12_EE_EE_EE_FF_FF_FF_FF))
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = exec.MipsEAGAIN
		expected.ActiveThread().Registers[7] = exec.SysErrorSignal
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_MT_StoreOpsClearMemReservation64(t *testing.T) {
	t.Parallel()

	type testCase struct {
		// name is the test name
		name string
		// opcode is the instruction opcode
		opcode uint32
		// offset is the immediate offset encoded in the instruction
		offset uint32
		// base is the base/rs register prestate
		base Word
		// effAddr is the address used to set the prestate preMem value. It is also used as the base LLAddress that can be adjusted reservation assertions
		effAddr Word
		// premem is the prestate value of the word located at effrAddr
		preMem Word
		// postMem is the expected post-state value of the word located at effAddr
		postMem Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "Store byte", opcode: 0b10_1000, base: 0xFF_00_00_00, offset: 0x10, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0x78_FF_FF_FF_FF_FF_FF_FF},
		{name: "Store byte lower", opcode: 0b10_1000, base: 0xFF_00_00_00, offset: 0x14, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0xFF_FF_FF_FF_78_FF_FF_FF},
		{name: "Store halfword", opcode: 0b10_1001, base: 0xFF_00_00_00, offset: 0x10, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0x56_78_FF_FF_FF_FF_FF_FF},
		{name: "Store halfword lower", opcode: 0b10_1001, base: 0xFF_00_00_00, offset: 0x14, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0xFF_FF_FF_FF_56_78_FF_FF},
		{name: "Store word left", opcode: 0b10_1010, base: 0xFF_00_00_00, offset: 0x10, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0x12_34_56_78_FF_FF_FF_FF},
		{name: "Store word left lower", opcode: 0b10_1010, base: 0xFF_00_00_00, offset: 0x14, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0xFF_FF_FF_FF_12_34_56_78},
		{name: "Store word", opcode: 0b10_1011, base: 0xFF_00_00_00, offset: 0x10, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0x12_34_56_78_FF_FF_FF_FF},
		{name: "Store word lower", opcode: 0b10_1011, base: 0xFF_00_00_00, offset: 0x14, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0xFF_FF_FF_FF_12_34_56_78},
		{name: "Store word right", opcode: 0b10_1110, base: 0xFF_00_00_00, offset: 0x10, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0x78_FF_FF_FF_FF_FF_FF_FF},
		{name: "Store word right lower", opcode: 0b10_1110, base: 0xFF_00_00_00, offset: 0x14, effAddr: 0xFF_00_00_10, preMem: ^Word(0), postMem: 0xFF_FF_FF_FF_78_FF_FF_FF},
	}

	pc := Word(0x08)
	rt := Word(0x12_34_56_78)
	//rt := Word(0x12_34_56_78_12_34_56_78)
	baseReg := uint32(5)
	rtReg := uint32(6)
	initState := func(t require.TestingT, testCase testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := uint32((testCase.opcode << 26) | (baseReg & 0x1F << 21) | (rtReg & 0x1F << 16) | (0xFFFF & testCase.offset))

		state.GetRegistersRef()[rtReg] = rt
		state.GetRegistersRef()[baseReg] = testCase.base
		storeInsnWithCache(state, goVm, pc, insn)
		state.GetMemory().SetWord(testCase.effAddr, testCase.preMem)
	}

	setExpectations := func(t require.TestingT, testCase testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ExpectMemoryWrite(testCase.effAddr, testCase.postMem)
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithPCAndNextPC(pc)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

var NoopSyscalls64 = map[string]uint32{
	"SysMunmap":        5011,
	"SysMprotect":      5010,
	"SysGetAffinity":   5196,
	"SysMadvise":       5027,
	"SysRtSigprocmask": 5014,
	"SysSigaltstack":   5129,
	"SysRtSigaction":   5013,
	"SysPrlimit64":     5297,
	"SysClose":         5003,
	"SysPread64":       5016,
	"SysStat":          5004,
	"SysFstat":         5005,
	//"SysFstat64":      UndefinedSysNr,
	"SysOpenAt":       5247,
	"SysReadlink":     5087,
	"SysReadlinkAt":   5257,
	"SysIoctl":        5015,
	"SysEpollCreate1": 5285,
	"SysPipe2":        5287,
	"SysEpollCtl":     5208,
	"SysEpollPwait":   5272,
	"SysGetRandom":    5313,
	"SysUname":        5061,
	//"SysStat64":       UndefinedSysNr,
	"SysGetuid": 5100,
	"SysGetgid": 5102,
	//"SysLlseek":       UndefinedSysNr,
	"SysMinCore":      5026,
	"SysTgkill":       5225,
	"SysGetRLimit":    5095,
	"SysLseek":        5008,
	"SysSetITimer":    5036,
	"SysTimerCreate":  5216,
	"SysTimerSetTime": 5217,
	"SysTimerDelete":  5220,
}

func getNoopSyscalls64(vmVersion versions.StateVersion) map[string]uint32 {
	noOpCalls := maps.Clone(NoopSyscalls64)
	features := versions.FeaturesForVersion(vmVersion)
	if !features.SupportNoopMprotect {
		delete(noOpCalls, "SysMprotect")
	}
	if features.SupportWorkingSysGetRandom {
		delete(noOpCalls, "SysGetRandom")
	}
	return noOpCalls
}

func getSupportedSyscalls(vmVersion versions.StateVersion) []uint32 {
	supportedSyscalls := []uint32{arch.SysMmap, arch.SysBrk, arch.SysClone, arch.SysExitGroup, arch.SysRead, arch.SysWrite, arch.SysFcntl, arch.SysExit, arch.SysSchedYield, arch.SysGetTID, arch.SysFutex, arch.SysOpen, arch.SysNanosleep, arch.SysClockGetTime, arch.SysGetpid, arch.SysEventFd2}

	features := versions.FeaturesForVersion(vmVersion)
	if features.SupportWorkingSysGetRandom {
		supportedSyscalls = append(supportedSyscalls, arch.SysGetRandom)
	}
	return supportedSyscalls
}

func TestEVM_NoopSyscall64(t *testing.T) {
	t.Parallel()
	for _, vmVersion := range GetMipsVersionTestCases(t) {
		noOpCalls := getNoopSyscalls64(vmVersion.Version)
		testNoopSyscall(t, vmVersion, noOpCalls)
	}
}

func TestEVM_UnsupportedSyscall64(t *testing.T) {
	t.Parallel()
	for _, vmVersion := range GetMipsVersionTestCases(t) {
		var noopSyscallNums = maps.Values(getNoopSyscalls64(vmVersion.Version))
		var SupportedSyscalls = getSupportedSyscalls(vmVersion.Version)
		unsupportedSyscalls := make([]uint32, 0, 400)
		for i := 5000; i < 5400; i++ {
			candidate := uint32(i)
			if slices.Contains(SupportedSyscalls, candidate) || slices.Contains(noopSyscallNums, candidate) {
				continue
			}
			unsupportedSyscalls = append(unsupportedSyscalls, candidate)
		}

		unsupported := unsupportedSyscalls
		testUnsupportedSyscall(t, vmVersion, unsupported)
	}
}

// Asserts the undefined syscall handling on cannon64 triggers a VM panic
func TestEVM_UndefinedSyscall(t *testing.T) {
	type testCase struct {
		name       string
		syscallNum Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{"SysFstat64", arch.SysFstat64},
		{"SysStat64", arch.SysStat64},
		{"SysLlseek", arch.SysLlseek},
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = Word(tt.syscallNum)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		goPanic := fmt.Sprintf("unrecognized syscall: %d", tt.syscallNum)
		evmErr := "unimplemented syscall"
		return ExpectVmPanic(goPanic, evmErr)
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}
