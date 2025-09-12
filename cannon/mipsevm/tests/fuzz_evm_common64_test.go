package tests

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func FuzzMulOp(f *testing.F) {
	const opcode uint32 = 28
	const mulFunct uint32 = 0x2
	multiplier := func(rs, rt Word) uint64 {
		return uint64(int64(int32(rs)) * int64(int32(rt)))
	}
	mulOpCheck(f, multiplier, opcode, true, mulFunct)
}

func FuzzMultOp(f *testing.F) {
	const multFunct uint32 = 0x18
	multiplier := func(rs, rt Word) uint64 {
		return uint64(int64(int32(rs)) * int64(int32(rt)))
	}
	mulOpCheck(f, multiplier, 0, false, multFunct)
}

func FuzzMultuOp(f *testing.F) {
	const multuFunct uint32 = 0x19
	multiplier := func(rs, rt Word) uint64 {
		return uint64(uint32(rs)) * uint64(uint32(rt))
	}
	mulOpCheck(f, multiplier, 0, false, multuFunct)
}

type multiplierFn func(rs, rt Word) uint64

func mulOpCheck(f *testing.F, multiplier multiplierFn, opcode uint32, expectRdReg bool, funct uint32) {
	f.Add(int64(0x80_00_00_00), int64(0x80_00_00_00), int64(1))
	f.Add(
		testutil.ToSignedInteger(0xFF_FF_FF_FF_11_22_33_44),
		testutil.ToSignedInteger(0xFF_FF_FF_FF_11_22_33_44),
		int64(1),
	)
	f.Add(
		testutil.ToSignedInteger(0xFF_FF_FF_FF_80_00_00_00),
		testutil.ToSignedInteger(0xFF_FF_FF_FF_80_00_00_00),
		int64(1),
	)
	f.Add(
		testutil.ToSignedInteger(0xFF_FF_FF_FF_FF_FF_FF_FF),
		testutil.ToSignedInteger(0xFF_FF_FF_FF_FF_FF_FF_FF),
		int64(1),
	)

	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		rs Word
		rt Word
	}

	rsReg := uint32(17)
	rtReg := uint32(18)
	rdReg := uint32(0)
	if expectRdReg {
		rdReg = 19
	}
	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		insn := opcode<<26 | rsReg<<21 | rtReg<<16 | rdReg<<11 | funct
		storeInsnWithCache(state, goVm, 0, insn)
		state.GetRegistersRef()[rsReg] = c.rs
		state.GetRegistersRef()[rtReg] = c.rt
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		result := multiplier(c.rs, c.rt)
		if expectRdReg {
			expected.ActiveThread().Registers[rdReg] = exec.SignExtend(result, 32)
		} else {
			expected.ActiveThread().LO = exec.SignExtend(result, 32)
			expected.ActiveThread().HI = exec.SignExtend(result>>32, 32)
		}
		return ExpectNormalExecution()
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState, mtutil.WithPCAndNextPC(0)).
		SetExpectations(setExpectations)

	f.Fuzz(func(t *testing.T, rs, rt, seed int64) {
		tests := []testCase{{rs: Word(rs), rt: Word(rt)}}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed)...)
	})
}
