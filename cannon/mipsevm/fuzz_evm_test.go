package mipsevm

import (
	"math/big"
	"os"
	"testing"

	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

const syscallInsn = uint32(0x00_00_00_0c)

func FuzzStateSyscallBrk(f *testing.F) {
	f.Fuzz(func(t *testing.T, pc uint32, step uint64, preimageOffset uint32) {
		pc = pc & 0xFF_FF_FF_FC // align PC
		nextPC := pc + 4
		state := &State{
			PC:             pc,
			NextPC:         nextPC,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysBrk},
			Step:           step,
			PreimageKey:    common.Hash{},
			PreimageOffset: preimageOffset,
		}
		state.Memory.SetMemory(pc, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = 0x4000_0000

		us := NewInstrumentedState(state, nil, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, pc+4, state.PC)
		require.Equal(t, nextPC+4, state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, expectedRegisters, state.Registers)
		require.Equal(t, step+1, state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, preimageOffset, state.PreimageOffset)

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallClone(f *testing.F) {
	f.Fuzz(func(t *testing.T, pc uint32, step uint64, preimageOffset uint32) {
		pc = pc & 0xFF_FF_FF_FC // align PC
		nextPC := pc + 4
		state := &State{
			PC:             pc,
			NextPC:         nextPC,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysClone},
			Step:           step,
			PreimageOffset: preimageOffset,
		}
		state.Memory.SetMemory(pc, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = 0x1

		us := NewInstrumentedState(state, nil, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, pc+4, state.PC)
		require.Equal(t, nextPC+4, state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, expectedRegisters, state.Registers)
		require.Equal(t, step+1, state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, preimageOffset, state.PreimageOffset)

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallMmap(f *testing.F) {
	f.Fuzz(func(t *testing.T, addr uint32, siz uint32, heap uint32) {
		state := &State{
			PC:             0,
			NextPC:         4,
			LO:             0,
			HI:             0,
			Heap:           heap,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysMmap, 4: addr, 5: siz},
			Step:           0,
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		preStateRegisters := state.Registers

		us := NewInstrumentedState(state, nil, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.PC)
		require.Equal(t, uint32(8), state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, uint32(0), state.PreimageOffset)
		if addr == 0 {
			expectedRegisters := preStateRegisters
			expectedRegisters[2] = heap
			require.Equal(t, expectedRegisters, state.Registers)
			sizAlign := siz
			if sizAlign&PageAddrMask != 0 { // adjust size to align with page size
				sizAlign = siz + PageSize - (siz & PageAddrMask)
			}
			require.Equal(t, uint32(heap+sizAlign), state.Heap)
		} else {
			expectedRegisters := preStateRegisters
			expectedRegisters[2] = addr
			require.Equal(t, expectedRegisters, state.Registers)
			require.Equal(t, uint32(heap), state.Heap)
		}

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallExitGroup(f *testing.F) {
	f.Fuzz(func(t *testing.T, exitCode uint8, pc uint32, step uint64) {
		pc = pc & 0xFF_FF_FF_FC // align PC
		nextPC := pc + 4
		state := &State{
			PC:             pc,
			NextPC:         nextPC,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysExitGroup, 4: uint32(exitCode)},
			Step:           step,
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(pc, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		preStateRegisters := state.Registers

		us := NewInstrumentedState(state, nil, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, pc, state.PC)
		require.Equal(t, nextPC, state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(exitCode), state.ExitCode)
		require.Equal(t, true, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, preStateRegisters, state.Registers)
		require.Equal(t, step+1, state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, uint32(0), state.PreimageOffset)

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallFnctl(f *testing.F) {
	f.Fuzz(func(t *testing.T, fd uint32, cmd uint32) {
		state := &State{
			PC:             0,
			NextPC:         4,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysFcntl, 4: fd, 5: cmd},
			Step:           0,
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		preStateRegisters := state.Registers

		us := NewInstrumentedState(state, nil, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.PC)
		require.Equal(t, uint32(8), state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, uint32(0), state.PreimageOffset)
		if cmd == 3 {
			expectedRegisters := preStateRegisters
			switch fd {
			case fdStdin, fdPreimageRead, fdHintRead:
				expectedRegisters[2] = 0
			case fdStdout, fdStderr, fdPreimageWrite, fdHintWrite:
				expectedRegisters[2] = 1
			default:
				expectedRegisters[2] = 0xFF_FF_FF_FF
				expectedRegisters[7] = MipsEBADF
			}
			require.Equal(t, expectedRegisters, state.Registers)
		} else {
			expectedRegisters := preStateRegisters
			expectedRegisters[2] = 0xFF_FF_FF_FF
			expectedRegisters[7] = MipsEINVAL
			require.Equal(t, expectedRegisters, state.Registers)
		}

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateHintRead(f *testing.F) {
	f.Fuzz(func(t *testing.T, addr uint32, count uint32) {
		preimageData := []byte("hello world")
		state := &State{
			PC:             0,
			NextPC:         4,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysRead, 4: fdHintRead, 5: addr, 6: count},
			Step:           0,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStatePreimageKey := state.PreimageKey
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = count

		oracle := staticOracle(t, preimageData) // only used for hinting
		us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.PC)
		require.Equal(t, uint32(8), state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, preStatePreimageKey, state.PreimageKey)

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStatePreimageRead(f *testing.F) {
	f.Fuzz(func(t *testing.T, addr uint32, count uint32, preimageOffset uint32) {
		preimageData := []byte("hello world")
		if preimageOffset >= uint32(len(preimageData)) {
			t.SkipNow()
		}
		state := &State{
			PC:             0,
			NextPC:         4,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysRead, 4: fdPreimageRead, 5: addr, 6: count},
			Step:           0,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: preimageOffset,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStatePreimageKey := state.PreimageKey
		preStateRoot := state.Memory.MerkleRoot()
		toWrite := count
		if toWrite > 4 {
			toWrite = 4
		}
		if preimageOffset+toWrite > uint32(8+len(preimageData)) {
			toWrite = uint32(8+len(preimageData)) - preimageOffset
		}
		oracle := staticOracle(t, preimageData)

		us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.True(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.PC)
		require.Equal(t, uint32(8), state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		if toWrite > 0 {
			require.NotEqual(t, preStateRoot, state.Memory.MerkleRoot())
			require.Greater(t, state.PreimageOffset, preimageOffset)
		} else {
			require.Equal(t, state.PreimageOffset, preimageOffset)
		}
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, preStatePreimageKey, state.PreimageKey)

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateHintWrite(f *testing.F) {
	f.Fuzz(func(t *testing.T, addr uint32, count uint32) {
		preimageData := []byte("hello world")
		state := &State{
			PC:             0,
			NextPC:         4,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysWrite, 4: fdHintWrite, 5: addr, 6: count},
			Step:           0,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: 0,

			// This is only used by mips.go. The reads a zeroed page-sized buffer when reading hint data from memory.
			// We pre-allocate a buffer for the read hint data to be copied into.
			LastHint: make(hexutil.Bytes, PageSize),
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStatePreimageKey := state.PreimageKey
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = count

		oracle := staticOracle(t, preimageData) // only used for hinting
		us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.PC)
		require.Equal(t, uint32(8), state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, preStatePreimageKey, state.PreimageKey)

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStatePreimageWrite(f *testing.F) {
	f.Fuzz(func(t *testing.T, addr uint32, count uint32) {
		preimageData := []byte("hello world")
		state := &State{
			PC:             0,
			NextPC:         4,
			LO:             0,
			HI:             0,
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         NewMemory(),
			Registers:      [32]uint32{2: sysWrite, 4: fdPreimageWrite, 5: addr, 6: count},
			Step:           0,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: 128,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		sz := 4 - (addr & 0x3)
		if sz < count {
			sz = count
		}
		expectedRegisters[2] = sz

		oracle := staticOracle(t, preimageData)
		us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)
		stepWitness, err := us.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.PC)
		require.Equal(t, uint32(8), state.NextPC)
		require.Equal(t, uint32(0), state.LO)
		require.Equal(t, uint32(0), state.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, uint32(0), state.PreimageOffset)

		evmPost := computeMIPSEVMPostState(t, stepWitness)
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func computeMIPSEVMPostState(t *testing.T, stepWitness *StepWitness) []byte {
	contracts, addrs := testContractsSetup(t)
	sender := common.Address{0x13, 0x37}
	env, evmState := NewEVMEnv(contracts, addrs)
	startingGas := uint64(30_000_000)

	if stepWitness.HasPreimage() {
		t.Logf("reading preimage key %x at offset %d", stepWitness.PreimageKey, stepWitness.PreimageOffset)
		poInput, err := stepWitness.EncodePreimageOracleInput()
		require.NoError(t, err, "encode preimage oracle input")
		_, leftOverGas, err := env.Call(vm.AccountRef(addrs.Sender), addrs.Oracle, poInput, startingGas, big.NewInt(0))
		require.NoErrorf(t, err, "evm should not fail, took %d gas", startingGas-leftOverGas)
	}
	input := stepWitness.EncodeStepInput()
	ret, _, err := env.Call(vm.AccountRef(sender), addrs.MIPS, input, startingGas, big.NewInt(0))
	require.NoError(t, err, "evm should not fail")
	require.Len(t, ret, 32, "expecting 32-byte state hash")
	// remember state hash, to check it against state
	postHash := common.Hash(*(*[32]byte)(ret))
	logs := evmState.Logs()
	require.Equal(t, 1, len(logs), "expecting a log with post-state")
	evmPost := logs[0].Data
	require.Equal(t, crypto.Keccak256Hash(evmPost), postHash, "logged state must be accurate")
	return evmPost
}
