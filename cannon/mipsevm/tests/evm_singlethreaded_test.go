package tests

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func TestEVM_LL(t *testing.T) {
	var tracer *tracing.Hooks

	cases := []struct {
		name    string
		base    uint32
		offset  int
		value   uint32
		effAddr uint32
	}{
		{name: "Aligned effAddr", base: 0x00_00_00_01, offset: 0x0133, value: 0xABCD, effAddr: 0x00_00_01_34},
		{name: "Aligned effAddr, signed extended", base: 0x00_00_00_01, offset: 0xFF33, value: 0xABCD, effAddr: 0xFF_FF_FF_34},
		{name: "Unaligned effAddr", base: 0xFF_12_00_01, offset: 0x3401, value: 0xABCD, effAddr: 0xFF_12_34_00},
		{name: "Unaligned effAddr, sign extended w overflow", base: 0xFF_12_00_01, offset: 0x8401, value: 0xABCD, effAddr: 0xFF_11_84_00},
	}
	v := GetSingleThreadedTestCase(t)
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rtReg := 5
			baseReg := 6
			pc := uint32(0x44)
			insn := uint32((0b11_0000 << 26) | (baseReg & 0x1F << 21) | (rtReg & 0x1F << 16) | (0xFFFF & c.offset))
			goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)), testutil.WithPC(pc), testutil.WithNextPC(pc+4))
			state := goVm.GetState()
			state.GetMemory().SetMemory(pc, insn)
			state.GetMemory().SetMemory(c.effAddr, c.value)
			state.GetRegistersRef()[baseReg] = c.base
			step := state.GetStep()

			// Setup expectations
			expected := testutil.NewExpectedState(state)
			expected.Step += 1
			expected.PC = pc + 4
			expected.NextPC = pc + 8
			expected.Registers[rtReg] = c.value

			stepWitness, err := goVm.Step(true)
			require.NoError(t, err)

			// Check expectations
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts, tracer)
		})
	}
}

func TestEVM_SC(t *testing.T) {
	var tracer *tracing.Hooks

	cases := []struct {
		name    string
		base    uint32
		offset  int
		value   uint32
		effAddr uint32
	}{
		{name: "Aligned effAddr", base: 0x00_00_00_01, offset: 0x0133, value: 0xABCD, effAddr: 0x00_00_01_34},
		{name: "Aligned effAddr, signed extended", base: 0x00_00_00_01, offset: 0xFF33, value: 0xABCD, effAddr: 0xFF_FF_FF_34},
		{name: "Unaligned effAddr", base: 0xFF_12_00_01, offset: 0x3401, value: 0xABCD, effAddr: 0xFF_12_34_00},
		{name: "Unaligned effAddr, sign extended w overflow", base: 0xFF_12_00_01, offset: 0x8401, value: 0xABCD, effAddr: 0xFF_11_84_00},
	}
	v := GetSingleThreadedTestCase(t)
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rtReg := 5
			baseReg := 6
			pc := uint32(0x44)
			insn := uint32((0b11_1000 << 26) | (baseReg & 0x1F << 21) | (rtReg & 0x1F << 16) | (0xFFFF & c.offset))
			goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)), testutil.WithPC(pc), testutil.WithNextPC(pc+4))
			state := goVm.GetState()
			state.GetMemory().SetMemory(pc, insn)
			state.GetRegistersRef()[baseReg] = c.base
			state.GetRegistersRef()[rtReg] = c.value
			step := state.GetStep()

			// Setup expectations
			expected := testutil.NewExpectedState(state)
			expected.Step += 1
			expected.PC = pc + 4
			expected.NextPC = pc + 8
			expected.Registers[rtReg] = 1 // 1 for success
			expectedMemory := memory.NewMemory()
			expectedMemory.SetMemory(pc, insn)
			expectedMemory.SetMemory(c.effAddr, c.value)
			expected.MemoryRoot = expectedMemory.MerkleRoot()

			stepWitness, err := goVm.Step(true)
			require.NoError(t, err)

			// Check expectations
			expected.Validate(t, state)
			testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts, tracer)
		})
	}
}
