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
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type operatorTestCase struct {
	name      string
	isImm     bool
	rs        Word
	rt        Word
	imm       uint16
	funct     uint32
	opcode    uint32
	expectRes Word
}

func (c operatorTestCase) Name() string {
	return c.name
}

func testOperators(t *testing.T, testCases []operatorTestCase, mips32Insn bool) {
	var cases []operatorTestCase
	for _, tt := range testCases {
		if mips32Insn {
			tt.rs = randomizeUpperWord(signExtend64(tt.rs))
			tt.rt = randomizeUpperWord(signExtend64(tt.rt))
			tt.expectRes = signExtend64(tt.expectRes)
		}
		cases = append(cases, tt)
	}

	pc := arch.Word(0)
	rtReg := uint32(8)
	rdReg := uint32(18)

	initState := func(t require.TestingT, tt operatorTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		var insn uint32
		var baseReg uint32 = 17
		if tt.isImm {
			insn = tt.opcode<<26 | baseReg<<21 | rtReg<<16 | uint32(tt.imm)
			state.GetRegistersRef()[rtReg] = tt.rt
			state.GetRegistersRef()[baseReg] = tt.rs
		} else {
			insn = baseReg<<21 | rtReg<<16 | rdReg<<11 | tt.funct
			state.GetRegistersRef()[baseReg] = tt.rs
			state.GetRegistersRef()[rtReg] = tt.rt
		}
		storeInsnWithCache(state, goVm, pc, insn)
	}

	setExpectations := func(t require.TestingT, tt operatorTestCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		if tt.isImm {
			expected.ActiveThread().Registers[rtReg] = tt.expectRes
		} else {
			expected.ActiveThread().Registers[rdReg] = tt.expectRes
		}
		return ExpectNormalExecution()
	}

	NewDiffTester((operatorTestCase).Name).
		InitState(initState, mtutil.WithPCAndNextPC(pc)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

type mulDivTestCase struct {
	name      string
	rs        Word
	rt        Word
	funct     uint32
	opcode    uint32
	expectHi  Word
	expectLo  Word
	expectRes Word
	rdReg     uint32
	panicMsg  string
	revertMsg string
}

func (c mulDivTestCase) Name() string {
	return c.name
}

func testMulDiv(t *testing.T, templateCases []mulDivTestCase, mips32Insn bool) {
	// Set up cases
	var cases []mulDivTestCase
	for _, tt := range templateCases {
		if mips32Insn {
			tt.rs = randomizeUpperWord(signExtend64(tt.rs))
			tt.rt = randomizeUpperWord(signExtend64(tt.rt))
			tt.expectHi = signExtend64(tt.expectHi)
			tt.expectLo = signExtend64(tt.expectLo)
			tt.expectRes = signExtend64(tt.expectRes)
		}
		cases = append(cases, tt)
	}

	baseReg := uint32(0x9)
	rtReg := uint32(0xa)
	pc := arch.Word(0)

	initState := func(t require.TestingT, tt mulDivTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := tt.opcode<<26 | baseReg<<21 | rtReg<<16 | tt.rdReg<<11 | tt.funct
		state.GetRegistersRef()[rtReg] = tt.rt
		state.GetRegistersRef()[baseReg] = tt.rs
		storeInsnWithCache(state, goVm, pc, insn)
	}

	setExpectations := func(t require.TestingT, tt mulDivTestCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		if tt.panicMsg != "" {
			return ExpectVmPanic(tt.panicMsg, tt.revertMsg)
		} else {
			expected.ExpectStep()
			if tt.expectRes != 0 {
				expected.ActiveThread().Registers[tt.rdReg] = tt.expectRes
			} else {
				expected.ActiveThread().HI = tt.expectHi
				expected.ActiveThread().LO = tt.expectLo
			}
			return ExpectNormalExecution()
		}
	}

	NewDiffTester((mulDivTestCase).Name).
		InitState(initState, mtutil.WithPCAndNextPC(pc)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

type loadStoreTestCase struct {
	name         string
	rt           Word
	base         Word
	imm          uint32
	opcode       uint32
	memVal       Word
	expectMemVal Word
	expectRes    Word
}

func (t loadStoreTestCase) effAddr() arch.Word {
	addr := t.base + Word(t.imm)
	return arch.AddressMask & addr
}

func (t loadStoreTestCase) Name() string {
	return t.name
}

func testLoadStore(t *testing.T, cases []loadStoreTestCase) {
	baseReg := uint32(9)
	rtReg := uint32(8)
	pc := arch.Word(0)

	initState := func(t require.TestingT, tt loadStoreTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := tt.opcode<<26 | baseReg<<21 | rtReg<<16 | tt.imm

		storeInsnWithCache(state, goVm, pc, insn)
		state.GetMemory().SetWord(tt.effAddr(), tt.memVal)
		state.GetRegistersRef()[rtReg] = tt.rt
		state.GetRegistersRef()[baseReg] = tt.base
	}

	setExpectations := func(t require.TestingT, tt loadStoreTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expect.ExpectStep()
		if tt.expectMemVal != 0 {
			expect.ExpectMemoryWrite(tt.effAddr(), tt.expectMemVal)
		} else {
			expect.ActiveThread().Registers[rtReg] = tt.expectRes
		}
		return ExpectNormalExecution()
	}

	NewDiffTester((loadStoreTestCase).Name).
		InitState(initState, mtutil.WithPCAndNextPC(pc)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

type branchTestCase struct {
	name         string
	pc           Word
	expectNextPC Word
	opcode       uint32
	regimm       uint32
	expectLink   bool
	rs           arch.SignedInteger
	offset       uint16
}

func (t branchTestCase) Name() string {
	return t.name
}

func testBranch(t *testing.T, cases []branchTestCase) {
	initState := func(t require.TestingT, tt branchTestCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		const rsReg = 8 // t0
		insn := tt.opcode<<26 | rsReg<<21 | tt.regimm<<16 | uint32(tt.offset)

		state.GetCurrentThread().Cpu.PC = tt.pc
		state.GetCurrentThread().Cpu.NextPC = tt.pc + 4
		storeInsnWithCache(state, goVm, tt.pc, insn)
		state.GetRegistersRef()[rsReg] = Word(tt.rs)
	}

	setExpectations := func(t require.TestingT, tt branchTestCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().NextPC = tt.expectNextPC
		if tt.expectLink {
			expected.ActiveThread().Registers[31] = tt.pc + 8
		}

		return ExpectNormalExecution()
	}

	NewDiffTester((branchTestCase).Name).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func testNoopSyscall(t *testing.T, vm VersionedVMTestCase, syscalls map[string]uint32) {
	type testCase struct {
		name      string
		sycallNum arch.Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	var cases []testCase
	for name, syscallNum := range syscalls {
		cases = append(cases, testCase{name: name, sycallNum: arch.Word(syscallNum)})
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = tt.sycallNum // Set syscall number
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = 0
		expected.ActiveThread().Registers[7] = 0

		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases, WithVm(vm))
}

func testUnsupportedSyscall(t *testing.T, vm VersionedVMTestCase, unsupportedSyscalls []uint32) {
	type testCase struct {
		name      string
		sycallNum arch.Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	var cases []testCase
	for _, syscallNum := range unsupportedSyscalls {
		name := fmt.Sprintf("Syscall %d", syscallNum)
		cases = append(cases, testCase{name: name, sycallNum: arch.Word(syscallNum)})
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = tt.sycallNum // Set syscall number
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		goErr := fmt.Sprintf("unrecognized syscall: %v", tt.sycallNum)
		return ExpectVmPanic(goErr, "unimplemented syscall")
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases, WithVm(vm))
}

// signExtend64 is used to sign-extend 32-bit words for 64-bit compatibility
func signExtend64(w Word) Word {
	if arch.IsMips32 {
		return w
	} else {
		return exec.SignExtend(w, 32)
	}
}

const seed = 0xdead

var rand = testutil.NewRandHelper(seed)

// randomizeUpperWord is used to assert that 32-bit operations use the lower word only
func randomizeUpperWord(w Word) Word {
	if arch.IsMips32 {
		return w
	} else {
		if w>>32 == 0x0 { // nolint:staticcheck
			rnd := rand.Uint32()
			upper := uint64(rnd) << 32
			return Word(upper | uint64(uint32(w)))
		} else {
			return w
		}
	}
}
