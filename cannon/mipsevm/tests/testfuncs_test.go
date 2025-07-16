package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

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

func testOperators(t *testing.T, cases []operatorTestCase, mips32Insn bool) {
	versions := GetMipsVersionTestCases(t)
	for _, v := range versions {
		for i, tt := range cases {
			// sign extend inputs for 64-bit compatibility
			if mips32Insn {
				tt.rs = randomizeUpperWord(signExtend64(tt.rs))
				tt.rt = randomizeUpperWord(signExtend64(tt.rt))
				tt.expectRes = signExtend64(tt.expectRes)
			}

			testName := fmt.Sprintf("%v (%v)", tt.name, v.Name)
			t.Run(testName, func(t *testing.T) {
				validator := testutil.NewEvmValidator(t, v.StateHashFn, v.Contracts)
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), mtutil.WithRandomization(int64(i)), mtutil.WithPC(0), mtutil.WithNextPC(4))
				state := goVm.GetState()
				var insn uint32
				var baseReg uint32 = 17
				var rtReg uint32
				var rdReg uint32
				if tt.isImm {
					rtReg = 8
					insn = tt.opcode<<26 | baseReg<<21 | rtReg<<16 | uint32(tt.imm)
					state.GetRegistersRef()[rtReg] = tt.rt
					state.GetRegistersRef()[baseReg] = tt.rs
				} else {
					rtReg = 18
					rdReg = 8
					insn = baseReg<<21 | rtReg<<16 | rdReg<<11 | tt.funct
					state.GetRegistersRef()[baseReg] = tt.rs
					state.GetRegistersRef()[rtReg] = tt.rt
				}
				testutil.StoreInstruction(state.GetMemory(), 0, insn)
				step := state.GetStep()

				// Setup expectations
				expected := mtutil.NewExpectedState(t, state)
				expected.ExpectStep()
				if tt.isImm {
					expected.ActiveThread().Registers[rtReg] = tt.expectRes
				} else {
					expected.ActiveThread().Registers[rdReg] = tt.expectRes
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Check expectations
				expected.Validate(t, state)
				validator.ValidateEVM(t, stepWitness, step, goVm)
			})
		}
	}
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

	initState := func(tt mulDivTestCase, state *multithreaded.State, vm VersionedVMTestCase) {
		insn := tt.opcode<<26 | baseReg<<21 | rtReg<<16 | tt.rdReg<<11 | tt.funct
		state.GetRegistersRef()[rtReg] = tt.rt
		state.GetRegistersRef()[baseReg] = tt.rs
		testutil.StoreInstruction(state.GetMemory(), pc, insn)
	}

	setExpectations := func(tt mulDivTestCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
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

	initState := func(tt loadStoreTestCase, state *multithreaded.State, vm VersionedVMTestCase) {
		insn := tt.opcode<<26 | baseReg<<21 | rtReg<<16 | tt.imm

		testutil.StoreInstruction(state.GetMemory(), pc, insn)
		state.GetMemory().SetWord(tt.effAddr(), tt.memVal)
		state.GetRegistersRef()[rtReg] = tt.rt
		state.GetRegistersRef()[baseReg] = tt.base
	}

	setExpectations := func(tt loadStoreTestCase, expect *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
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

func testBranch(t *testing.T, cases []branchTestCase) {
	versions := GetMipsVersionTestCases(t)
	for _, v := range versions {
		for i, tt := range cases {
			testName := fmt.Sprintf("%v (%v)", tt.name, v.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), mtutil.WithRandomization(int64(i)), mtutil.WithPCAndNextPC(tt.pc))
				state := goVm.GetState()
				const rsReg = 8 // t0
				insn := tt.opcode<<26 | rsReg<<21 | tt.regimm<<16 | uint32(tt.offset)
				testutil.StoreInstruction(state.GetMemory(), tt.pc, insn)
				state.GetRegistersRef()[rsReg] = Word(tt.rs)
				step := state.GetStep()

				// Setup expectations
				expected := mtutil.NewExpectedState(t, state)
				expected.ExpectStep()
				expected.ActiveThread().NextPC = tt.expectNextPC
				if tt.expectLink {
					expected.ActiveThread().Registers[31] = state.GetPC() + 8
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Check expectations
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts)
			})
		}
	}
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

	initState := func(tt testCase, state *multithreaded.State, vm VersionedVMTestCase) {
		testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = tt.sycallNum // Set syscall number
	}

	setExpectations := func(tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
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

	initState := func(tt testCase, state *multithreaded.State, vm VersionedVMTestCase) {
		testutil.StoreInstruction(state.Memory, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = tt.sycallNum // Set syscall number
	}

	setExpectations := func(tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
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
