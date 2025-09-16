package tests

import (
	"bytes"
	"io"
	"math/big"
	"os"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/register"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/versions"
)

type insnCache interface {
	UpdateInstructionCache(pc arch.Word)
}

func storeInsnWithCache(state *multithreaded.State, goVm mipsevm.FPVM, pc arch.Word, insn uint32) {
	testutil.StoreInstruction(state.GetMemory(), pc, insn)
	if ic, ok := goVm.(insnCache); ok {
		ic.UpdateInstructionCache(pc)
	}
}

func TestEVM_SingleStep_Jump(t *testing.T) {
	type testCase struct {
		name         string
		pc           arch.Word
		nextPC       arch.Word
		insn         uint32
		expectNextPC arch.Word
		expectLink   bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "j MSB set target", pc: 0, nextPC: 4, insn: 0x0A_00_00_02, expectNextPC: 0x08_00_00_08},                                           // j 0x02_00_00_02
		{name: "j non-zero PC region", pc: 0x10000000, nextPC: 0x10000004, insn: 0x08_00_00_02, expectNextPC: 0x10_00_00_08},                     // j 0x2
		{name: "jal MSB set target", pc: 0, nextPC: 4, insn: 0x0E_00_00_02, expectNextPC: 0x08_00_00_08, expectLink: true},                       // jal 0x02_00_00_02
		{name: "jal non-zero PC region", pc: 0x10000000, nextPC: 0x10000004, insn: 0x0C_00_00_02, expectNextPC: 0x10_00_00_08, expectLink: true}, // jal 0x2
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.GetCurrentThread().Cpu.PC = tt.pc
		state.GetCurrentThread().Cpu.NextPC = tt.nextPC
		storeInsnWithCache(state, goVm, tt.pc, tt.insn)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().NextPC = tt.expectNextPC
		if tt.expectLink {
			expected.ActiveThread().Registers[31] = tt.pc + 8
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_Operators(t *testing.T) {
	cases := []operatorTestCase{
		{name: "add", funct: 0x20, isImm: false, rs: Word(12), rt: Word(20), expectRes: Word(32)},                                  // add t0, s1, s2
		{name: "add", funct: 0x20, isImm: false, rs: ^Word(0), rt: ^Word(0), expectRes: Word(0xFF_FF_FF_FE)},                       // add t0, s1, s2
		{name: "add", funct: 0x20, isImm: false, rs: Word(0x7F_FF_FF_FF), rt: Word(0x7F_FF_FF_FF), expectRes: Word(0xFF_FF_FF_FE)}, // add t0, s1, s2
		{name: "add", funct: 0x20, isImm: false, rs: ^Word(0), rt: Word(2), expectRes: Word(1)},                                    // add t0, s1, s2
		{name: "add", funct: 0x20, isImm: false, rs: Word(2), rt: ^Word(0), expectRes: Word(1)},                                    // add t0, s1, s2
		{name: "add", funct: 0x20, isImm: false, rs: Word(0x7F_FF_FF_FF), rt: Word(1), expectRes: Word(0x80_00_00_00)},             // add t0, s1, s2

		{name: "addu", funct: 0x21, isImm: false, rs: Word(12), rt: Word(20), expectRes: Word(32)},                                  // addu t0, s1, s2
		{name: "addu", funct: 0x21, isImm: false, rs: ^Word(0), rt: ^Word(0), expectRes: Word(0xFF_FF_FF_FE)},                       // addu t0, s1, s2
		{name: "addu", funct: 0x21, isImm: false, rs: Word(0x7F_FF_FF_FF), rt: Word(0x7F_FF_FF_FF), expectRes: Word(0xFF_FF_FF_FE)}, // addu t0, s1, s2
		{name: "addu", funct: 0x21, isImm: false, rs: ^Word(0), rt: Word(2), expectRes: Word(1)},                                    // addu t0, s1, s2
		{name: "addu", funct: 0x21, isImm: false, rs: Word(0x7F_FF_FF_FF), rt: Word(1), expectRes: Word(0x80_00_00_00)},             // addu t0, s1, s2

		{name: "addi", opcode: 0x8, isImm: true, rs: Word(4), rt: Word(1), imm: uint16(40), expectRes: Word(44)},                              // addi t0, s1, 40
		{name: "addi", opcode: 0x8, isImm: true, rs: ^Word(0), rt: Word(0xAA_BB_CC_DD), imm: uint16(1), expectRes: Word(0)},                   // addi t0, s1, 40
		{name: "addi", opcode: 0x8, isImm: true, rs: ^Word(0), rt: Word(0xAA_BB_CC_DD), imm: uint16(0xFF_FF), expectRes: Word(0xFF_FF_FF_FE)}, // addi t0, s1, 40
		{name: "addi sign", opcode: 0x8, isImm: true, rs: Word(2), rt: Word(1), imm: uint16(0xfffe), expectRes: Word(0)},                      // addi t0, s1, -2

		{name: "addiu", opcode: 0x9, isImm: true, rs: Word(4), rt: Word(1), imm: uint16(40), expectRes: Word(44)},                              // addiu t0, s1, 40
		{name: "addiu", opcode: 0x9, isImm: true, rs: ^Word(0), rt: Word(0xAA_BB_CC_DD), imm: uint16(1), expectRes: Word(0)},                   // addiu t0, s1, 40
		{name: "addiu", opcode: 0x9, isImm: true, rs: ^Word(0), rt: Word(0xAA_BB_CC_DD), imm: uint16(0xFF_FF), expectRes: Word(0xFF_FF_FF_FE)}, // addiu t0, s1, 40

		{name: "sub", funct: 0x22, isImm: false, rs: Word(20), rt: Word(12), expectRes: Word(8)},            // sub t0, s1, s2
		{name: "sub", funct: 0x22, isImm: false, rs: ^Word(0), rt: Word(1), expectRes: Word(0xFF_FF_FF_FE)}, // sub t0, s1, s2
		{name: "sub", funct: 0x22, isImm: false, rs: Word(1), rt: ^Word(0), expectRes: Word(0x2)},           // sub t0, s1, s2

		{name: "subu", funct: 0x23, isImm: false, rs: Word(20), rt: Word(12), expectRes: Word(8)},            // subu t0, s1, s2
		{name: "subu", funct: 0x23, isImm: false, rs: ^Word(0), rt: Word(1), expectRes: Word(0xFF_FF_FF_FE)}, // subu t0, s1, s2
		{name: "subu", funct: 0x23, isImm: false, rs: Word(1), rt: ^Word(0), expectRes: Word(0x2)},           // subu t0, s1, s2
	}
	testOperators(t, cases, true)
}

func TestEVM_SingleStep_Bitwise(t *testing.T) {
	// bitwise operations that use the full word size
	cases := []operatorTestCase{
		{name: "and", funct: 0x24, isImm: false, rs: Word(0b1010_1100), rt: Word(0b1100_0101), expectRes: Word(0b1000_0100)},                                   // and t0, s1, s2
		{name: "andi", opcode: 0xc, isImm: true, rs: Word(0b1010_1100), rt: Word(1), imm: uint16(0b1100_0101), expectRes: Word(0b1000_0100)},                   // andi t0, s1, imm
		{name: "or", funct: 0x25, isImm: false, rs: Word(0b1010_1100), rt: Word(0b1100_0101), expectRes: Word(0b1110_1101)},                                    // or t0, s1, s2
		{name: "ori", opcode: 0xd, isImm: true, rs: Word(0b1010_1100), rt: Word(0xFFFF_FFFF), imm: uint16(0b1100_0101), expectRes: Word(0b1110_1101)},          // ori t0, s1, imm
		{name: "xor", funct: 0x26, isImm: false, rs: Word(0b1010_1100), rt: Word(0b1100_0101), expectRes: Word(0b0110_1001)},                                   // xor t0, s1, s2
		{name: "xori", opcode: 0xe, isImm: true, rs: Word(0b1010_1100), rt: Word(1), imm: uint16(0b1100_0101), expectRes: Word(0b0110_1001)},                   // xori t0, s1, imm
		{name: "nor", funct: 0x27, isImm: false, rs: Word(0b1010_1100), rt: Word(0b1100_0101), expectRes: signExtend64(0b0001_0010 | 0xFFFF_FF00)},             // nor t0, s1, s2
		{name: "slt, success, positive vals", funct: 0x2a, isImm: false, rs: 1, rt: Word(5), expectRes: Word(1)},                                               // slt t0, s1, s2
		{name: "slt, success, mixed vals", funct: 0x2a, isImm: false, rs: signExtend64(0xFF_FF_FF_FE), rt: Word(5), expectRes: Word(1)},                        // slt t0, s1, s2
		{name: "slt, success, negative vals", funct: 0x2a, isImm: false, rs: signExtend64(0xFF_FF_FF_FD), rt: signExtend64(0xFF_FF_FF_FE), expectRes: Word(1)}, // slt t0, s1, s2
		{name: "slt, fail, negative values", funct: 0x2a, isImm: false, rs: signExtend64(0xFF_FF_FF_FE), rt: signExtend64(0xFF_FF_FF_FD), expectRes: Word(0)},  // slt t0, s1, s2
		{name: "slt, fail, positive values", funct: 0x2a, isImm: false, rs: 555, rt: 123, expectRes: Word(0)},                                                  // slt t0, s1, s2
		{name: "slt, fail, mixed values", funct: 0x2a, isImm: false, rs: 555, rt: signExtend64(0xFF_FF_FF_FD), expectRes: Word(0)},                             // slt t0, s1, s2
		{name: "slti, success, positive vals", opcode: 0xa, isImm: true, rs: 1, imm: 5, expectRes: Word(1)},
		{name: "slti, success, mixed vals", opcode: 0xa, isImm: true, rs: signExtend64(0xFF_FF_FF_FE), imm: 5, expectRes: Word(1)},
		{name: "slti, success, negative vals", opcode: 0xa, isImm: true, rs: signExtend64(0xFF_FF_FF_FD), imm: 0xFFFE, expectRes: Word(1)},
		{name: "slti, fail, negative values", opcode: 0xa, isImm: true, rs: signExtend64(0xFF_FF_FF_FE), imm: 0xFFFD, expectRes: Word(0)},
		{name: "slti, fail, positive values", opcode: 0xa, isImm: true, rs: 555, imm: 123, expectRes: Word(0)},
		{name: "slti, fail, mixed values", opcode: 0xa, isImm: true, rs: 555, imm: 0xFFFD, expectRes: Word(0)},
		{name: "sltu, success", funct: 0x2b, isImm: false, rs: Word(490), rt: Word(1200), expectRes: Word(1)},                                                  // sltu t0, s1, s2
		{name: "sltu, success, large values", funct: 0x2b, isImm: false, rs: signExtend64(0xFF_FF_FF_FD), rt: signExtend64(0xFF_FF_FF_FE), expectRes: Word(1)}, // sltu t0, s1, s2
		{name: "sltu, fail", funct: 0x2b, isImm: false, rs: Word(1200), rt: Word(490), expectRes: Word(0)},                                                     // sltu t0, s1, s2
		{name: "sltu, fail, large values", funct: 0x2b, isImm: false, rs: signExtend64(0xFF_FF_FF_FE), rt: signExtend64(0xFF_FF_FF_FD), expectRes: Word(0)},    // sltu t0, s1, s2
		{name: "sltiu, success", opcode: 0xb, isImm: true, rs: Word(490), imm: 1200, expectRes: Word(1)},
		{name: "sltiu, success, large values", opcode: 0xb, isImm: true, rs: signExtend64(0xFF_FF_FF_FD), imm: 0xFFFE, expectRes: Word(1)},
		{name: "sltiu, fail", opcode: 0xb, isImm: true, rs: Word(1200), imm: 490, expectRes: Word(0)},
		{name: "sltiu, fail, large values", opcode: 0xb, isImm: true, rs: signExtend64(0xFF_FF_FF_FE), imm: 0xFFFD, expectRes: Word(0)},
	}
	testOperators(t, cases, false)
}

func TestEVM_SingleStep_Lui(t *testing.T) {
	type testCase struct {
		name     string
		rtReg    uint32
		imm      uint32
		expectRt Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "lui unsigned", rtReg: 5, imm: 0x1234, expectRt: 0x1234_0000},
		{name: "lui signed", rtReg: 7, imm: 0x8765, expectRt: signExtend64(0x8765_0000)},
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := 0b1111<<26 | uint32(tt.rtReg)<<16 | (tt.imm & 0xFFFF)
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[tt.rtReg] = tt.expectRt
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_CloClz(t *testing.T) {
	type testCase struct {
		name           string
		rs             Word
		expectedResult Word
		funct          uint32
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "clo", rs: 0xFFFF_FFFE, expectedResult: 31, funct: 0b10_0001},
		{name: "clo", rs: 0xE000_0000, expectedResult: 3, funct: 0b10_0001},
		{name: "clo", rs: 0x8000_0000, expectedResult: 1, funct: 0b10_0001},
		{name: "clo, sign-extended", rs: signExtend64(0x8000_0000), expectedResult: 1, funct: 0b10_0001},
		{name: "clo, sign-extended", rs: signExtend64(0xF800_0000), expectedResult: 5, funct: 0b10_0001},
		{name: "clz", rs: 0x1, expectedResult: 31, funct: 0b10_0000},
		{name: "clz", rs: 0x1000_0000, expectedResult: 3, funct: 0b10_0000},
		{name: "clz", rs: 0x8000_0000, expectedResult: 0, funct: 0b10_0000},
		{name: "clz, sign-extended", rs: signExtend64(0x8000_0000), expectedResult: 0, funct: 0b10_0000},
	}

	rsReg := uint32(5)
	rdReg := uint32(6)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := 0b01_1100<<26 | rsReg<<21 | rdReg<<11 | tt.funct
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetRegistersRef()[rsReg] = tt.rs
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[rdReg] = tt.expectedResult
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_MovzMovn(t *testing.T) {
	type testCase struct {
		name          string
		funct         uint32
		testValue     Word
		shouldSucceed bool
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "movz, success", funct: uint32(0xa), testValue: 0, shouldSucceed: true},
		{name: "movz, failure, testVal=1", funct: uint32(0xa), testValue: 1, shouldSucceed: false},
		{name: "movz, failure, testVal=2", funct: uint32(0xa), testValue: 2, shouldSucceed: false},
		{name: "movn, success, testVal=1", funct: uint32(0xb), testValue: 1, shouldSucceed: true},
		{name: "movn, success, testVal=2", funct: uint32(0xb), testValue: 2, shouldSucceed: true},
		{name: "movn, failure", funct: uint32(0xb), testValue: 0, shouldSucceed: false},
	}

	pc := arch.Word(0)
	rsReg := uint32(9)
	rtReg := uint32(10)
	rdReg := uint32(8)
	val := Word(0xb)
	otherVal := Word(0xa)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := rsReg<<21 | rtReg<<16 | rdReg<<11 | tt.funct
		state.GetRegistersRef()[rtReg] = tt.testValue
		state.GetRegistersRef()[rsReg] = val
		state.GetRegistersRef()[rdReg] = otherVal
		storeInsnWithCache(state, goVm, pc, insn)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		if tt.shouldSucceed {
			expected.ActiveThread().Registers[rdReg] = val
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithPCAndNextPC(pc)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_MfhiMflo(t *testing.T) {
	type testCase struct {
		name   string
		funct  uint32
		hi     Word
		lo     Word
		result Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "mflo", funct: uint32(0x12), lo: Word(0xdeadbeef), hi: Word(0x0), result: Word(0xdeadbeef)},
		{name: "mfhi", funct: uint32(0x10), lo: Word(0x0), hi: Word(0xdeadbeef), result: Word(0xdeadbeef)},
	}

	rdReg := uint32(8)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := rdReg<<11 | tt.funct
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetCurrentThread().Cpu.HI = tt.hi
		state.GetCurrentThread().Cpu.LO = tt.lo
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[rdReg] = tt.result
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_MulDiv(t *testing.T) {
	flip := testutil.FlipSign
	cases := []mulDivTestCase{
		{name: "mul", funct: uint32(0x2), opcode: uint32(28), rs: Word(5), rt: Word(2), rdReg: uint32(0x8), expectRes: Word(10)},                                    // mul t0, t1, t2
		{name: "mul", funct: uint32(0x2), opcode: uint32(28), rs: Word(0x1), rt: ^Word(0), rdReg: uint32(0x8), expectRes: ^Word(0)},                                 // mul t1, t2
		{name: "mul", funct: uint32(0x2), opcode: uint32(28), rs: Word(0xFF_FF_FF_FF), rt: Word(0xFF_FF_FF_FF), rdReg: uint32(0x8), expectRes: Word(0x1)},           // mul t1, t2
		{name: "mul", funct: uint32(0x2), opcode: uint32(28), rs: Word(0xFF_FF_FF_D3), rt: Word(0xAA_BB_CC_DD), rdReg: uint32(0x8), expectRes: Word(0xFC_FC_FD_27)}, // mul t1, t2

		{name: "mult", funct: uint32(0x18), rs: Word(0x0F_FF_00_00), rt: Word(100), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0x6), expectLo: Word(0x3F_9C_00_00)},           // mult t1, t2
		{name: "mult", funct: uint32(0x18), rs: Word(0x1), rt: Word(0xFF_FF_FF_FF), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0xFF_FF_FF_FF), expectLo: Word(0xFF_FF_FF_FF)}, // mult t1, t2
		{name: "mult", funct: uint32(0x18), rs: Word(0xFF_FF_FF_FF), rt: Word(0xFF_FF_FF_FF), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0), expectLo: Word(0x1)},             // mult t1, t2
		{name: "mult", funct: uint32(0x18), rs: Word(0xFF_FF_FF_D3), rt: Word(0xAA_BB_CC_DD), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0xE), expectLo: Word(0xFC_FC_FD_27)}, // mult t1, t2

		{name: "multu", funct: uint32(0x19), rs: Word(0x0F_FF_00_00), rt: Word(100), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0x6), expectLo: Word(0x3F_9C_00_00)},                     // multu t1, t2
		{name: "multu", funct: uint32(0x19), rs: Word(0x1), rt: Word(0xFF_FF_FF_FF), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0x0), expectLo: Word(0xFF_FF_FF_FF)},                     // multu t1, t2
		{name: "multu", funct: uint32(0x19), rs: Word(0xFF_FF_FF_FF), rt: Word(0xFF_FF_FF_FF), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0xFF_FF_FF_FE), expectLo: Word(0x1)},           // multu t1, t2
		{name: "multu", funct: uint32(0x19), rs: Word(0xFF_FF_FF_D3), rt: Word(0xAA_BB_CC_DD), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0xAA_BB_CC_BE), expectLo: Word(0xFC_FC_FD_27)}, // multu t1, t2
		{name: "multu", funct: uint32(0x19), rs: Word(0xFF_FF_FF_D3), rt: Word(0xAA_BB_CC_BE), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(0xAA_BB_CC_9F), expectLo: Word(0xFC_FD_02_9A)}, // multu t1, t2

		{name: "div", funct: uint32(0x1a), rs: Word(5), rt: Word(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(1), expectLo: Word(2)},                                          // div t1, t2
		{name: "div w neg dividend", funct: uint32(0x1a), rs: flip(9), rt: Word(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: flip(1), expectLo: flip(4)},                           // div t1, t2
		{name: "div w neg divisor", funct: uint32(0x1a), rs: 9, rt: flip(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: 1, expectLo: flip(4)},                                        // div t1, t2
		{name: "div w neg operands", funct: uint32(0x1a), rs: flip(9), rt: flip(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: flip(1), expectLo: 4},                                 // div t1, t2
		{name: "div by zero", funct: uint32(0x1a), rs: Word(5), rt: Word(0), rdReg: uint32(0x0), opcode: uint32(0), panicMsg: "instruction divide by zero", revertMsg: "division by zero"}, // div t1, t2
		{name: "divu", funct: uint32(0x1b), rs: Word(5), rt: Word(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: Word(1), expectLo: Word(2)},
		{name: "divu w neg dividend", funct: uint32(0x1b), rs: flip(9), rt: Word(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: 1, expectLo: (flip(9) & exec.U32Mask) >> 1},           // div t1, t2
		{name: "divu w neg divisor", funct: uint32(0x1b), rs: 9, rt: flip(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: 9, expectLo: 0},                                              // div t1, t2
		{name: "divu w neg divisor #2", funct: uint32(0x1b), rs: 2, rt: flip(9), rdReg: uint32(0x0), opcode: uint32(0), expectHi: 2, expectLo: 0},                                           // div t1, t2
		{name: "divu w neg operands", funct: uint32(0x1b), rs: flip(9), rt: flip(2), rdReg: uint32(0x0), opcode: uint32(0), expectHi: flip(9), expectLo: 0},                                 // divu t1, t2
		{name: "divu w neg operands #2", funct: uint32(0x1b), rs: flip(2), rt: flip(9), rdReg: uint32(0x0), opcode: uint32(0), expectHi: 7, expectLo: 1},                                    // divu t1, t2
		{name: "divu by zero", funct: uint32(0x1b), rs: Word(5), rt: Word(0), rdReg: uint32(0x0), opcode: uint32(0), panicMsg: "instruction divide by zero", revertMsg: "division by zero"}, // divu t1, t2
	}

	testMulDiv(t, cases, true)
}

func TestEVM_SingleStep_MthiMtlo(t *testing.T) {
	type testCase struct {
		name  string
		funct uint32
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "mtlo", funct: uint32(0x13)},
		{name: "mthi", funct: uint32(0x11)},
	}

	val := Word(0xdeadbeef)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		rsReg := uint32(8)
		insn := rsReg<<21 | tt.funct
		storeInsnWithCache(state, goVm, state.GetPC(), insn)
		state.GetRegistersRef()[rsReg] = val
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		if tt.funct == 0x11 {
			expected.ActiveThread().HI = val
		} else {
			expected.ActiveThread().LO = val
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_BeqBne(t *testing.T) {
	type testCase struct {
		name           string
		imm            uint16
		opcode         uint32
		rs             Word
		rt             Word
		expectedNextPC Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "bne, success", opcode: uint32(0x5), imm: 10, rs: Word(0x123), rt: Word(0x456), expectedNextPC: 844},                                  // bne $t0, $t1, 16
		{name: "bne, success, signed-extended offset", opcode: uint32(0x5), imm: negative(3), rs: Word(0x123), rt: Word(0x456), expectedNextPC: 792}, // bne $t0, $t1, 16
		{name: "bne, fail", opcode: uint32(0x5), imm: 10, rs: Word(0x123), rt: Word(0x123), expectedNextPC: 808},                                     // bne $t0, $t1, 16
		{name: "beq, success", opcode: uint32(0x4), imm: 10, rs: Word(0x123), rt: Word(0x123), expectedNextPC: 844},                                  // beq $t0, $t1, 16
		{name: "beq, success, sign-extended offset", opcode: uint32(0x4), imm: negative(25), rs: Word(0x123), rt: Word(0x123), expectedNextPC: 704},  // beq $t0, $t1, 16
		{name: "beq, fail", opcode: uint32(0x4), imm: 10, rs: Word(0x123), rt: Word(0x456), expectedNextPC: 808},
	}

	pc := Word(800)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		rsReg := uint32(9)
		rtReg := uint32(8)
		insn := tt.opcode<<26 | rsReg<<21 | rtReg<<16 | uint32(tt.imm)
		state.GetRegistersRef()[rtReg] = tt.rt
		state.GetRegistersRef()[rsReg] = tt.rs
		storeInsnWithCache(state, goVm, pc, insn)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().NextPC = tt.expectedNextPC
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithPCAndNextPC(pc)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func negative(value Word) uint16 {
	flipped := testutil.FlipSign(value)
	return uint16(flipped)
}

func TestEVM_SingleStep_SlSr(t *testing.T) {
	type testCase struct {
		name      string
		rs        Word
		rt        Word
		rsReg     uint32
		funct     uint16
		expectVal Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "sll", funct: uint16(4) << 6, rt: Word(0x20), rsReg: uint32(0x0), expectVal: Word(0x200)}, // sll t0, t1, 3
		{name: "sll with overflow", funct: uint16(1) << 6, rt: Word(0x8000_0000), rsReg: uint32(0x0), expectVal: 0x0},
		{name: "sll with sign extension", funct: uint16(4) << 6, rt: Word(0x0800_0000), rsReg: uint32(0x0), expectVal: signExtend64(0x8000_0000)},
		{name: "sll with max shift, sign extension", funct: uint16(31) << 6, rt: Word(0x01), rsReg: uint32(0x0), expectVal: signExtend64(0x8000_0000)},
		{name: "sll with max shift, overflow", funct: uint16(31) << 6, rt: Word(0x02), rsReg: uint32(0x0), expectVal: 0x0},
		{name: "srl", funct: uint16(4)<<6 | 2, rt: Word(0x20), rsReg: uint32(0x0), expectVal: Word(0x2)},                                                // srl t0, t1, 3
		{name: "srl with sign extension", funct: uint16(0)<<6 | 2, rt: Word(0x8000_0000), rsReg: uint32(0x0), expectVal: signExtend64(0x8000_0000)},     // srl t0, t1, 3
		{name: "sra", funct: uint16(4)<<6 | 3, rt: Word(0x70_00_00_20), rsReg: uint32(0x0), expectVal: signExtend64(0x07_00_00_02)},                     // sra t0, t1, 3
		{name: "sra with sign extension", funct: uint16(4)<<6 | 3, rt: Word(0x80_00_00_20), rsReg: uint32(0x0), expectVal: signExtend64(0xF8_00_00_02)}, // sra t0, t1, 3
		{name: "sllv", funct: uint16(4), rt: Word(0x20), rs: Word(4), rsReg: uint32(0xa), expectVal: Word(0x200)},                                       // sllv t0, t1, t2
		{name: "sllv with overflow", funct: uint16(4), rt: Word(0x8000_0000), rs: Word(1), rsReg: uint32(0xa), expectVal: 0x0},
		{name: "sllv with sign extension", funct: uint16(4), rt: Word(0x0800_0000), rs: Word(4), rsReg: uint32(0xa), expectVal: signExtend64(0x8000_0000)},
		{name: "sllv with max shift, sign extension", funct: uint16(4), rt: Word(0x01), rs: Word(31), rsReg: uint32(0xa), expectVal: signExtend64(0x8000_0000)},
		{name: "sllv with max shift, overflow", funct: uint16(4), rt: Word(0x02), rs: Word(31), rsReg: uint32(0xa), expectVal: 0x0},
		{name: "srlv", funct: uint16(6), rt: Word(0x20_00), rs: Word(4), rsReg: uint32(0xa), expectVal: Word(0x02_00)},                                          // srlv t0, t1, t2
		{name: "srlv with sign extension", funct: uint16(6), rt: Word(0x8000_0000), rs: Word(0), rsReg: uint32(0xa), expectVal: signExtend64(0x8000_0000)},      // srlv t0, t1, t2
		{name: "srlv with zero extension", funct: uint16(6), rt: Word(0x8000_0000), rs: Word(1), rsReg: uint32(0xa), expectVal: 0x4000_0000},                    // srlv t0, t1, t2
		{name: "srav", funct: uint16(7), rt: Word(0x1deafbee), rs: Word(12), rsReg: uint32(0xa), expectVal: signExtend64(Word(0x0001deaf))},                     // srav t0, t1, t2
		{name: "srav with sign extension", funct: uint16(7), rt: Word(0xdeafbeef), rs: Word(12), rsReg: uint32(0xa), expectVal: signExtend64(Word(0xfffdeafb))}, // srav t0, t1, t2
	}

	pc := Word(0)
	rdReg := uint32(0x8)
	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		rtReg := uint32(0x9)
		insn := tt.rsReg<<21 | rtReg<<16 | rdReg<<11 | uint32(tt.funct)
		state.GetRegistersRef()[rtReg] = tt.rt
		state.GetRegistersRef()[tt.rsReg] = tt.rs
		storeInsnWithCache(state, goVm, pc, insn)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[rdReg] = tt.expectVal
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithPCAndNextPC(pc)).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_JrJalr(t *testing.T) {
	type testCase struct {
		name       string
		funct      uint16
		rsReg      uint32
		jumpTo     Word
		rdReg      uint32
		pc         Word
		nextPC     Word
		expectLink bool
		errorMsg   string
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "jr", funct: uint16(0x8), rsReg: 8, jumpTo: 0x34, pc: 0, nextPC: 4},                                                                                       // jr t0
		{name: "jr, delay slot", funct: uint16(0x8), rsReg: 8, jumpTo: 0x34, pc: 0, nextPC: 8, errorMsg: "jump in delay slot"},                                           // jr t0
		{name: "jalr", funct: uint16(0x9), rsReg: 8, jumpTo: 0x34, rdReg: uint32(0x9), expectLink: true, pc: 0, nextPC: 4},                                               // jalr t1, t0
		{name: "jalr, delay slot", funct: uint16(0x9), rsReg: 8, jumpTo: 0x34, rdReg: uint32(0x9), expectLink: true, pc: 0, nextPC: 100, errorMsg: "jump in delay slot"}, // jalr t1, t0
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := tt.rsReg<<21 | tt.rdReg<<11 | uint32(tt.funct)
		state.GetRegistersRef()[tt.rsReg] = tt.jumpTo
		state.GetCurrentThread().Cpu.PC = tt.pc
		state.GetCurrentThread().Cpu.NextPC = tt.nextPC
		storeInsnWithCache(state, goVm, tt.pc, insn)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		if tt.errorMsg != "" {
			return ExpectVmPanic(tt.errorMsg, tt.errorMsg)
		} else {
			expected.ExpectStep()
			expected.ActiveThread().NextPC = tt.jumpTo
			if tt.expectLink {
				expected.ActiveThread().Registers[tt.rdReg] = tt.pc + 8
			}
			return ExpectNormalExecution()
		}
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SingleStep_Sync(t *testing.T) {
	type testCase struct {
		name string
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "simple"},
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		syncInsn := uint32(0x0000_000F)
		storeInsnWithCache(state, goVm, state.GetPC(), syncInsn)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_MMap(t *testing.T) {
	type testCase struct {
		name         string
		heap         arch.Word
		address      arch.Word
		size         arch.Word
		shouldFail   bool
		expectedHeap arch.Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "Increment heap by max value", heap: program.HEAP_START, address: 0, size: ^arch.Word(0), shouldFail: true},
		{name: "Increment heap to 0", heap: program.HEAP_START, address: 0, size: ^arch.Word(0) - program.HEAP_START + 1, shouldFail: true},
		{name: "Increment heap to previous page", heap: program.HEAP_START, address: 0, size: ^arch.Word(0) - program.HEAP_START - memory.PageSize + 1, shouldFail: true},
		{name: "Increment max page size", heap: program.HEAP_START, address: 0, size: ^arch.Word(0) & ^arch.Word(memory.PageAddrMask), shouldFail: true},
		{name: "Increment max page size from 0", heap: 0, address: 0, size: ^arch.Word(0) & ^arch.Word(memory.PageAddrMask), shouldFail: true},
		{name: "Increment heap at limit", heap: program.HEAP_END, address: 0, size: 1, shouldFail: true},
		{name: "Increment heap to limit", heap: program.HEAP_END - memory.PageSize, address: 0, size: 1, shouldFail: false, expectedHeap: program.HEAP_END},
		{name: "Increment heap within limit", heap: program.HEAP_END - 2*memory.PageSize, address: 0, size: 1, shouldFail: false, expectedHeap: program.HEAP_END - memory.PageSize},
		{name: "Request specific address", heap: program.HEAP_START, address: 0x50_00_00_00, size: 0, shouldFail: false, expectedHeap: program.HEAP_START},
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetRegistersRef()[2] = arch.SysMmap
		state.GetRegistersRef()[4] = c.address
		state.GetRegistersRef()[5] = c.size
		state.Heap = c.heap
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		if c.shouldFail {
			expected.ActiveThread().Registers[2] = exec.MipsEINVAL
			expected.ActiveThread().Registers[7] = exec.SysErrorSignal
		} else {
			expected.Heap = c.expectedHeap
			if c.address == 0 {
				expected.ActiveThread().Registers[2] = c.heap
				expected.ActiveThread().Registers[7] = 0
			} else {
				expected.ActiveThread().Registers[2] = c.address
				expected.ActiveThread().Registers[7] = 0
			}
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_SysGetRandom_isImplemented(t *testing.T) {
	t.Parallel()
	// Assert we have at least one vm with the working getrandom syscall
	foundVmWithSyscallEnabled := false
	for _, vers := range GetMipsVersionTestCases(t) {
		features := versions.FeaturesForVersion(vers.Version)
		foundVmWithSyscallEnabled = foundVmWithSyscallEnabled || features.SupportWorkingSysGetRandom
	}
	require.True(t, foundVmWithSyscallEnabled)

	// Assert that latest version has a working getrandom ssycall
	latestFeatures := versions.FeaturesForVersion(versions.GetExperimentalVersion())
	require.True(t, latestFeatures.SupportWorkingSysGetRandom)
}

func TestEVM_SysGetRandom(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name                 string
		bufAddrOffset        arch.Word
		bufLen               arch.Word
		expectedRandDataMask arch.Word
		expectedReturnValue  arch.Word
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		// Test word-aligned buffer address
		{name: "Word-aligned buffer, zero bytes requested", bufAddrOffset: 0, bufLen: 0, expectedRandDataMask: 0x0000_0000_0000_0000, expectedReturnValue: 0},
		{name: "Word-aligned buffer, 1 byte requested", bufAddrOffset: 0, bufLen: 1, expectedRandDataMask: 0xFF00_0000_0000_0000, expectedReturnValue: 1},
		{name: "Word-aligned buffer, 2 byte requested", bufAddrOffset: 0, bufLen: 2, expectedRandDataMask: 0xFFFF_0000_0000_0000, expectedReturnValue: 2},
		{name: "Word-aligned buffer, 3 byte requested", bufAddrOffset: 0, bufLen: 3, expectedRandDataMask: 0xFFFF_FF00_0000_0000, expectedReturnValue: 3},
		{name: "Word-aligned buffer, 7 byte requested", bufAddrOffset: 0, bufLen: 7, expectedRandDataMask: 0xFFFF_FFFF_FFFF_FF00, expectedReturnValue: 7},
		{name: "Word-aligned buffer, 8 byte requested", bufAddrOffset: 0, bufLen: 8, expectedRandDataMask: 0xFFFF_FFFF_FFFF_FFFF, expectedReturnValue: 8},
		// Test buffer address offset by 1
		{name: "Buffer offset by 1, zero bytes requested", bufAddrOffset: 1, bufLen: 0, expectedRandDataMask: 0x0000_0000_0000_0000, expectedReturnValue: 0},
		{name: "Buffer offset by 1, 1 byte requested", bufAddrOffset: 1, bufLen: 1, expectedRandDataMask: 0x00FF_0000_0000_0000, expectedReturnValue: 1},
		{name: "Buffer offset by 1, 2 byte requested", bufAddrOffset: 1, bufLen: 2, expectedRandDataMask: 0x00FF_FF00_0000_0000, expectedReturnValue: 2},
		{name: "Buffer offset by 1, 3 byte requested", bufAddrOffset: 1, bufLen: 6, expectedRandDataMask: 0x00FF_FFFF_FFFF_FF00, expectedReturnValue: 6},
		{name: "Buffer offset by 1, 7 byte requested", bufAddrOffset: 1, bufLen: 7, expectedRandDataMask: 0x00FF_FFFF_FFFF_FFFF, expectedReturnValue: 7},
		{name: "Buffer offset by 1, 8 byte requested", bufAddrOffset: 1, bufLen: 8, expectedRandDataMask: 0x00FF_FFFF_FFFF_FFFF, expectedReturnValue: 7},
		// Test buffer address offset by 6
		{name: "Buffer offset by 6, zero bytes requested", bufAddrOffset: 6, bufLen: 0, expectedRandDataMask: 0x0000_0000_0000_0000, expectedReturnValue: 0},
		{name: "Buffer offset by 6, 1 byte requested", bufAddrOffset: 6, bufLen: 1, expectedRandDataMask: 0x0000_0000_0000_FF00, expectedReturnValue: 1},
		{name: "Buffer offset by 6, 2 byte requested", bufAddrOffset: 6, bufLen: 2, expectedRandDataMask: 0x0000_0000_0000_FFFF, expectedReturnValue: 2},
		{name: "Buffer offset by 6, 3 byte requested", bufAddrOffset: 6, bufLen: 6, expectedRandDataMask: 0x0000_0000_0000_FFFF, expectedReturnValue: 2},
		{name: "Buffer offset by 6, 7 byte requested", bufAddrOffset: 6, bufLen: 7, expectedRandDataMask: 0x0000_0000_0000_FFFF, expectedReturnValue: 2},
		{name: "Buffer offset by 6, 8 byte requested", bufAddrOffset: 6, bufLen: 8, expectedRandDataMask: 0x0000_0000_0000_FFFF, expectedReturnValue: 2},
	}

	startingMemory := arch.Word(0x1234_5678_8765_4321)
	effAddr := arch.Word(0x1000_0000)
	// Random data is generated using the incremented step as the random seed
	// For validation of this random data see instrumented_test.go TestSplitmix64 unit tests
	step := uint64(0x1a2b3c4d5e6f7531) - 1
	randomData := arch.Word(0x4141302768c9e9d0)

	initState := func(t require.TestingT, testCase testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetMemory().SetWord(effAddr, startingMemory)
		state.GetRegistersRef()[register.RegV0] = arch.SysGetRandom
		state.GetRegistersRef()[register.RegA0] = effAddr + testCase.bufAddrOffset
		state.GetRegistersRef()[register.RegA1] = testCase.bufLen
	}

	setExpectations := func(t require.TestingT, testCase testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		isNoop := !versions.FeaturesForVersion(vm.Version).SupportWorkingSysGetRandom
		expectedMemory := testCase.expectedRandDataMask&randomData | ^testCase.expectedRandDataMask&startingMemory

		expected.ExpectStep()
		if isNoop {
			expected.ActiveThread().Registers[register.RegSyscallRet1] = 0
			expected.ActiveThread().Registers[register.RegSyscallErrno] = 0
		} else {
			expected.ActiveThread().Registers[register.RegSyscallRet1] = testCase.expectedReturnValue
			expected.ActiveThread().Registers[register.RegSyscallErrno] = 0
			expected.ExpectMemoryWrite(effAddr, expectedMemory)
		}
		return ExpectNormalExecution()
	}

	NewDiffTester(testNamer).
		InitState(initState, mtutil.WithStep(step)).
		SetExpectations(setExpectations).
		Run(t, cases, SkipAutomaticMemoryReservationTests())
	//Was getting failure from the “automatic memory reservation” modifier that the DiffTester adds.
	//I think the mod executes extra setup on a different thread before the syscall, which I think bumps the step counter.
	//Since sys_getrandom seeds splitmix64 with the incremented step, I think those extra steps shift the seed.
}

func TestEVM_SysWriteHint(t *testing.T) {
	type testCase struct {
		name             string
		memOffset        int      // Where the hint data is stored in memory
		hintData         []byte   // Hint data stored in memory at memOffset
		bytesToWrite     int      // How many bytes of hintData to write
		lastHint         []byte   // The buffer that stores lastHint in the state
		expectedHints    [][]byte // The hints we expect to be processed
		expectedLastHint []byte   // The lastHint we should expect for the post-state
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{
			name:      "write 1 full hint at beginning of page",
			memOffset: 4096,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 10,
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write 1 full hint across page boundary",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 8, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 12,
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write 2 full hints",
			memOffset: 5012,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
				0, 0, 0, 8, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 22,
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write a single partial hint",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite:     8,
			lastHint:         []byte{},
			expectedHints:    nil,
			expectedLastHint: []byte{0, 0, 0, 6, 0xAA, 0xAA, 0xAA, 0xAA},
		},
		{
			name:      "write 1 full, 1 partial hint",
			memOffset: 5012,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
				0, 0, 0, 8, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 16,
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{0, 0, 0, 8, 0xAA, 0xAA},
		},
		{
			name:      "write a single partial hint to large capacity lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite:     8,
			lastHint:         make([]byte, 0, 4096),
			expectedHints:    nil,
			expectedLastHint: []byte{0, 0, 0, 6, 0xAA, 0xAA, 0xAA, 0xAA},
		},
		{
			name:      "write full hint to large capacity lastHint buffer",
			memOffset: 5012,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 10,
			lastHint:     make([]byte, 0, 4096),
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write multiple hints to large capacity lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 8, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC, // Hint data
				0, 0, 0, 8, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 24,
			lastHint:     make([]byte, 0, 4096),
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC},
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write remaining hint data to non-empty lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC, // Hint data
			},
			bytesToWrite: 8,
			lastHint:     []byte{0, 0, 0, 8},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write partial hint data to non-empty lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC, // Hint data
			},
			bytesToWrite:     4,
			lastHint:         []byte{0, 0, 0, 8},
			expectedHints:    nil,
			expectedLastHint: []byte{0, 0, 0, 8, 0xAA, 0xAA, 0xAA, 0xAA},
		},
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.LastHint = tt.lastHint
		state.GetRegistersRef()[2] = arch.SysWrite
		state.GetRegistersRef()[4] = exec.FdHintWrite
		state.GetRegistersRef()[5] = arch.Word(tt.memOffset)
		state.GetRegistersRef()[6] = arch.Word(tt.bytesToWrite)
		// Set up memory
		err := state.GetMemory().SetMemoryRange(arch.Word(tt.memOffset), bytes.NewReader(tt.hintData))
		require.NoError(t, err)
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.LastHint = tt.expectedLastHint
		expected.ActiveThread().Registers[2] = arch.Word(tt.bytesToWrite) // Return count of bytes written
		expected.ActiveThread().Registers[7] = 0                          // no Error
		return ExpectNormalExecution()
	}

	postCheck := func(t require.TestingT, tt testCase, vm VersionedVMTestCase, deps *TestDependencies, wit *mipsevm.StepWitness) {
		trackingOracle, ok := deps.po.(*testutil.HintTrackingOracle)
		require.True(t, ok)
		require.Equal(t, tt.expectedHints, trackingOracle.Hints())
	}

	po := func() mipsevm.PreimageOracle { return &testutil.HintTrackingOracle{} }
	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		PostCheck(postCheck).
		Run(t, cases, WithPreimageOracle(po))
}

func TestEVM_Fault(t *testing.T) {
	type testCase struct {
		name         string
		pc           arch.Word
		nextPC       arch.Word
		insn         uint32
		goPanicValue interface{}
		evmErrStr    string
		evmErrSig    string
	}

	testNamer := func(tc testCase) string {
		return tc.name
	}

	cases := []testCase{
		{name: "illegal instruction", nextPC: 0, insn: 0b111110 << 26, evmErrStr: "invalid instruction", goPanicValue: "invalid instruction: f8000000"},
		{name: "branch in delay-slot", nextPC: 8, insn: 0x11_02_00_03, evmErrStr: "branch in delay slot", goPanicValue: "branch in delay slot"},
		{name: "jump in delay-slot", nextPC: 8, insn: 0x0c_00_00_0c, evmErrStr: "jump in delay slot", goPanicValue: "jump in delay slot"},

		{name: "misaligned instruction", pc: 1, nextPC: 4, insn: 0b110111_00001_00001 << 16, evmErrSig: "InvalidPC()", goPanicValue: "unaligned instruction fetch: PC = 0x1"},
		{name: "misaligned instruction", pc: 2, nextPC: 4, insn: 0b110111_00001_00001 << 16, evmErrSig: "InvalidPC()", goPanicValue: "unaligned instruction fetch: PC = 0x2"},
		{name: "misaligned instruction", pc: 3, nextPC: 4, insn: 0b110111_00001_00001 << 16, evmErrSig: "InvalidPC()", goPanicValue: "unaligned instruction fetch: PC = 0x3"},
		{name: "misaligned instruction", pc: 5, nextPC: 4, insn: 0b110111_00001_00001 << 16, evmErrSig: "InvalidPC()", goPanicValue: "unaligned instruction fetch: PC = 0x5"},
	}

	initState := func(t require.TestingT, tt testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		storeInsnWithCache(state, goVm, 0, tt.insn)
		state.GetCurrentThread().Cpu.PC = tt.pc
		state.GetCurrentThread().Cpu.NextPC = tt.nextPC
		// set the return address ($ra) to jump into when test completes
		state.GetRegistersRef()[31] = testutil.EndAddr
	}

	setExpectations := func(t require.TestingT, tt testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		// Memory is accessed when processing illegal instructions, so we need to make sure to append a memory proof
		// See: https://github.com/ethereum-optimism/optimism/blob/a08b5b343a0005c6308566cd8afa810dd67e0e8f/cannon/mipsevm/exec/mips_instructions.go#L102-L105
		rsReg := (tt.insn >> 21) & 0x1F
		rs := expected.ActiveThread().Registers[rsReg]
		memAddr := testutil.EffAddr(rs + exec.SignExtendImmediate(tt.insn))

		if tt.evmErrSig != "" {
			return ExpectVmPanicWithCustomErr(tt.goPanicValue, tt.evmErrSig, WithMemoryProofAddr(memAddr))
		} else {
			return ExpectVmPanic(tt.goPanicValue, tt.evmErrStr, WithMemoryProofAddr(memAddr))
		}
	}

	NewDiffTester(testNamer).
		InitState(initState).
		SetExpectations(setExpectations).
		Run(t, cases)
}

func TestEVM_RandomProgram(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "true" {
		t.Skip("Skipping slow test because SKIP_SLOW_TESTS is enabled")
	}

	t.Parallel()
	versionCases := GetMipsVersionTestCases(t)

	for _, v := range versionCases {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()

			if !versions.FeaturesForVersion(v.Version).SupportWorkingSysGetRandom {
				t.Skip("Skipping vm version that does not support working sys_getrandom")
			}

			validator := testutil.NewEvmValidator(t, v.StateHashFn, v.Contracts)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := testutil.ProgramPath("random", testutil.Go1_24)
			goVm := v.ElfVMFactory(t, elfFile, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			start := time.Now()
			for i := 0; i < 500_000; i++ {
				step := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}
				insn := testutil.GetInstruction(state.GetMemory(), state.GetPC())
				if i%100_000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				validator.ValidateEVM(t, stepWitness, step, goVm)
			}
			end := time.Now()
			delta := end.Sub(start)
			t.Logf("test took %s, %d instructions, %s per instruction", delta, state.GetStep(), delta/time.Duration(state.GetStep()))

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

			// Check output
			// Define the regex pattern we expect to match against stdOut
			pattern := `Random (hex data|int): (\w+)\s*`
			re, err := regexp.Compile(pattern)
			require.NoError(t, err)

			// Check that stdOut matches the expected regex
			expectedMatches := 3
			output := stdOutBuf.String()
			matches := re.FindAllStringSubmatch(output, -1)
			require.Equal(t, expectedMatches, len(matches))

			// Check each match and validate the random values that are printed to stdOut
			for i := 0; i < expectedMatches; i++ {
				match := matches[i]
				require.Contains(t, match[0], "Random")

				// Check that the generated random number is not zero
				dataType := match[1]
				dataValue := match[2]
				switch dataType {
				case "hex data":
					randVal, success := new(big.Int).SetString(dataValue, 16)
					require.True(t, success, "should successfully set hex value")
					require.NotEqual(t, 0, randVal.Sign(), "random data should be non-zero")
				case "int":
					randVal, err := strconv.ParseUint(dataValue, 10, 64)
					require.NoError(t, err)
					require.NotEqual(t, uint64(0), randVal, "random int should be non-zero")
				}
			}
		})
	}
}

func TestEVM_SyscallEventFdProgram(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "true" {
		t.Skip("Skipping slow test because SKIP_SLOW_TESTS is enabled")
	}

	t.Parallel()
	versionCases := GetMipsVersionTestCases(t)

	for _, v := range versionCases {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()

			validator := testutil.NewEvmValidator(t, v.StateHashFn, v.Contracts)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := testutil.ProgramPath("syscall-eventfd", v.GoTarget)
			goVm := v.ElfVMFactory(t, elfFile, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			start := time.Now()
			for i := 0; i < 550_000; i++ {
				step := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}
				insn := testutil.GetInstruction(state.GetMemory(), state.GetPC())
				if i%100_000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				validator.ValidateEVM(t, stepWitness, step, goVm)
			}
			end := time.Now()
			delta := end.Sub(start)
			t.Logf("test took %s, %d instructions, %s per instruction", delta, state.GetStep(), delta/time.Duration(state.GetStep()))

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

			// Check output
			output := stdOutBuf.String()
			require.Contains(t, output, "call eventfd with valid flags: '0x80080'")
			require.Contains(t, output, "call eventfd with valid flags: '0xFFFFFFFFFFFFFFFF'")
			require.Contains(t, output, "call eventfd with valid flags: '0x80'")
			require.Contains(t, output, "call eventfd with invalid flags: '0x0'")
			require.Contains(t, output, "call eventfd with invalid flags: '0xFFFFFFFFFFFFFF7F'")
			require.Contains(t, output, "call eventfd with invalid flags: '0x80000'")
			require.Contains(t, output, "write to eventfd object")
			require.Contains(t, output, "read from eventfd object")
			require.Contains(t, output, "done")

			// Check fd value
			pattern := `eventfd2 fd = '(.+)'`
			re, err := regexp.Compile(pattern)
			require.NoError(t, err)
			matches := re.FindAllStringSubmatch(output, -1)

			expectedMatches := 3
			require.Equal(t, expectedMatches, len(matches))
			for i := 0; i < expectedMatches; i++ {
				require.Equal(t, "100", matches[i][1])
			}
		})
	}
}

func TestEVM_HelloProgram(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "true" {
		t.Skip("Skipping slow test because SKIP_SLOW_TESTS is enabled")
	}

	t.Parallel()
	versions := GetMipsVersionTestCases(t)

	for _, v := range versions {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			validator := testutil.NewEvmValidator(t, v.StateHashFn, v.Contracts)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := testutil.ProgramPath("hello", v.GoTarget)
			goVm := v.ElfVMFactory(t, elfFile, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			start := time.Now()
			for i := 0; i < 500_000; i++ {
				step := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}
				insn := testutil.GetInstruction(state.GetMemory(), state.GetPC())
				if i%100_000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				validator.ValidateEVM(t, stepWitness, step, goVm)
			}
			end := time.Now()
			delta := end.Sub(start)
			t.Logf("test took %s, %d instructions, %s per instruction", delta, state.GetStep(), delta/time.Duration(state.GetStep()))

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

			require.Equal(t, "hello world!\n", stdOutBuf.String(), "stdout says hello")
			require.Equal(t, "", stdErrBuf.String(), "stderr silent")
		})
	}
}

func TestEVM_ClaimProgram(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "true" {
		t.Skip("Skipping slow test because SKIP_SLOW_TESTS is enabled")
	}

	t.Parallel()
	versions := GetMipsVersionTestCases(t)

	for _, v := range versions {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			validator := testutil.NewEvmValidator(t, v.StateHashFn, v.Contracts)
			oracle, expectedStdOut, expectedStdErr := testutil.ClaimTestOracle(t)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := testutil.ProgramPath("claim", v.GoTarget)
			goVm := v.ElfVMFactory(t, elfFile, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			for i := 0; i < 2000_000; i++ {
				curStep := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}

				insn := testutil.GetInstruction(state.GetMemory(), state.GetPC())
				if i%1_000_000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				validator.ValidateEVM(t, stepWitness, curStep, goVm)
			}
			t.Logf("Completed in %d steps", state.GetStep())

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

			require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
			require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
		})
	}
}

func TestEVM_EntryProgram(t *testing.T) {
	if os.Getenv("SKIP_SLOW_TESTS") == "true" {
		t.Skip("Skipping slow test because SKIP_SLOW_TESTS is enabled")
	}

	t.Parallel()
	versions := GetMipsVersionTestCases(t)

	for _, v := range versions {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			validator := testutil.NewEvmValidator(t, v.StateHashFn, v.Contracts)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := testutil.ProgramPath("entry", v.GoTarget)
			goVm := v.ElfVMFactory(t, elfFile, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			start := time.Now()
			for i := 0; i < 500_000; i++ {
				curStep := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}
				insn := testutil.GetInstruction(state.GetMemory(), state.GetPC())
				if i%10_000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				validator.ValidateEVM(t, stepWitness, curStep, goVm)
			}
			end := time.Now()
			delta := end.Sub(start)
			t.Logf("test took %s, %d instructions, %s per instruction", delta, state.GetStep(), delta/time.Duration(state.GetStep()))

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")
		})
	}
}
