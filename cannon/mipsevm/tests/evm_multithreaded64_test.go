//go:build cannon64
// +build cannon64

// These tests target architectures that are 64-bit or larger
package tests

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mttestutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func TestEVM_MT64_LL(t *testing.T) {
	var tracer *tracing.Hooks

	memVal := Word(0x11223344_55667788)
	memValNeg := Word(0xF1223344_F5667788)
	cases := []struct {
		name   string
		base   Word
		offset int
		addr   Word
		memVal Word
		retReg int
		retVal Word
	}{
		{name: "8-byte-aligned addr", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "8-byte-aligned addr, neg value", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memValNeg, retVal: 0xFFFFFFFF_F1223344, retReg: 5},
		{name: "8-byte-aligned addr, extra bits", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "8-byte-aligned addr, signed extended", base: 0x01, offset: 0xFF37, addr: 0xFFFF_FFFF_FFFF_FF38, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "8-byte-aligned addr, signed extended w overflow", base: 0x1000_0001, offset: 0xFF07, addr: 0x0000_0000_0FFF_FF08, memVal: memVal, retVal: 0x11223344, retReg: 5},
		{name: "4-byte-aligned addr", base: 0x01, offset: 0x0103, addr: 0x0104, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "4-byte-aligned addr, neg value", base: 0x01, offset: 0x0104, addr: 0x0105, memVal: memValNeg, retVal: 0xFFFFFFFF_F5667788, retReg: 5},
		{name: "4-byte-aligned addr, extra bits", base: 0x01, offset: 0x0105, addr: 0x0106, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "4-byte-aligned addr, signed extended", base: 0x01, offset: 0xFF33, addr: 0xFFFF_FFFF_FFFF_FF34, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "4-byte-aligned addr, signed extended w overflow", base: 0x1000_0001, offset: 0xFF03, addr: 0x0000_0000_0FFF_FF04, memVal: memVal, retVal: 0x55667788, retReg: 5},
		{name: "Return register set to 0", base: 0x01, offset: 0x0107, addr: 0x0108, memVal: memVal, retVal: 0x11223344, retReg: 0},
	}
	for i, c := range cases {
		for _, withExistingReservation := range []bool{true, false} {
			tName := fmt.Sprintf("%v (withExistingReservation = %v)", c.name, withExistingReservation)
			t.Run(tName, func(t *testing.T) {
				effAddr := arch.AddressMask & c.addr

				retReg := c.retReg
				baseReg := 6
				pc := Word(0x44)
				insn := uint32((0b11_0000 << 26) | (baseReg & 0x1F << 21) | (retReg & 0x1F << 16) | (0xFFFF & c.offset))
				goVm, state, contracts := setup(t, i, nil)
				step := state.GetStep()

				// Set up state
				state.GetCurrentThread().Cpu.PC = pc
				state.GetCurrentThread().Cpu.NextPC = pc + 4
				state.GetMemory().SetUint32(pc, insn)
				state.GetMemory().SetWord(effAddr, c.memVal)
				state.GetRegistersRef()[baseReg] = c.base
				if withExistingReservation {
					state.LLReservationActive = true
					state.LLAddress = c.addr + 1
					state.LLOwnerThread = 123
				} else {
					state.LLReservationActive = false
					state.LLAddress = 0
					state.LLOwnerThread = 0
				}

				// Set up expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.ExpectStep()
				expected.LLReservationActive = true
				expected.LLAddress = c.addr
				expected.LLOwnerThread = state.GetCurrentThread().ThreadId
				if retReg != 0 {
					expected.ActiveThread().Registers[retReg] = c.retVal
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Check expectations
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
			})
		}
	}
}

func TestEVM_MT64_SC(t *testing.T) {
	var tracer *tracing.Hooks

	llVariations := []struct {
		name                string
		llReservationActive bool
		matchThreadId       bool
		matchAddr           bool
		shouldSucceed       bool
	}{
		{name: "should succeed", llReservationActive: true, matchThreadId: true, matchAddr: true, shouldSucceed: true},
		{name: "mismatch addr", llReservationActive: true, matchThreadId: false, matchAddr: true, shouldSucceed: false},
		{name: "mismatched thread", llReservationActive: true, matchThreadId: true, matchAddr: false, shouldSucceed: false},
		{name: "mismatched addr & thread", llReservationActive: true, matchThreadId: false, matchAddr: false, shouldSucceed: false},
		{name: "no active reservation", llReservationActive: false, matchThreadId: true, matchAddr: true, shouldSucceed: false},
	}

	cases := []struct {
		name           string
		base           Word
		offset         int
		addr           Word
		value          Word
		expectedMemVal Word
		rtReg          int
		threadId       Word
	}{
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
	for i, c := range cases {
		for _, v := range llVariations {
			tName := fmt.Sprintf("%v (%v)", c.name, v.name)
			t.Run(tName, func(t *testing.T) {
				effAddr := arch.AddressMask & c.addr

				// Setup
				rtReg := c.rtReg
				baseReg := 6
				pc := Word(0x44)
				insn := uint32((0b11_1000 << 26) | (baseReg & 0x1F << 21) | (rtReg & 0x1F << 16) | (0xFFFF & c.offset))
				goVm, state, contracts := setup(t, i, nil)
				mttestutil.InitializeSingleThread(i*23456, state, i%2 == 1)
				step := state.GetStep()

				// Define LL-related params
				var llAddress, llOwnerThread Word
				if v.matchAddr {
					llAddress = c.addr
				} else {
					llAddress = c.addr + 1
				}
				if v.matchThreadId {
					llOwnerThread = c.threadId
				} else {
					llOwnerThread = c.threadId + 1
				}

				// Setup state
				state.GetCurrentThread().ThreadId = c.threadId
				state.GetCurrentThread().Cpu.PC = pc
				state.GetCurrentThread().Cpu.NextPC = pc + 4
				state.GetMemory().SetUint32(pc, insn)
				state.GetRegistersRef()[baseReg] = c.base
				state.GetRegistersRef()[rtReg] = c.value
				state.LLReservationActive = v.llReservationActive
				state.LLAddress = llAddress
				state.LLOwnerThread = llOwnerThread

				// Setup expectations
				expected := mttestutil.NewExpectedMTState(state)
				expected.ExpectStep()
				var retVal Word
				if v.shouldSucceed {
					retVal = 1
					expected.ExpectMemoryWordWrite(effAddr, c.expectedMemVal)
					expected.LLReservationActive = false
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
				testutil.ValidateEVM(t, stepWitness, step, goVm, multithreaded.GetStateHashFn(), contracts, tracer)
			})
		}
	}
}
