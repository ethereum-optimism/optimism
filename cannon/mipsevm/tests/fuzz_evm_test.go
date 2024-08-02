package tests

import (
	"bytes"
	"math/rand"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

const syscallInsn = uint32(0x00_00_00_0c)

func FuzzStateSyscallBrk(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	f.Fuzz(func(t *testing.T, pc uint32, step uint64, preimageOffset uint32) {
		pc = pc & 0xFF_FF_FF_FC // align PC
		nextPC := pc + 4
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     pc,
				NextPC: nextPC,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysBrk},
			Step:           step,
			PreimageKey:    common.Hash{},
			PreimageOffset: preimageOffset,
		}
		state.Memory.SetMemory(pc, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = 0x4000_0000

		goState := singlethreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, pc+4, state.Cpu.PC)
		require.Equal(t, nextPC+4, state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, expectedRegisters, state.Registers)
		require.Equal(t, step+1, state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, preimageOffset, state.PreimageOffset)

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallClone(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	f.Fuzz(func(t *testing.T, pc uint32, step uint64, preimageOffset uint32) {
		pc = pc & 0xFF_FF_FF_FC // align PC
		nextPC := pc + 4
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     pc,
				NextPC: nextPC,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysClone},
			Step:           step,
			PreimageOffset: preimageOffset,
		}
		state.Memory.SetMemory(pc, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = 0x1

		goState := singlethreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, pc+4, state.Cpu.PC)
		require.Equal(t, nextPC+4, state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, expectedRegisters, state.Registers)
		require.Equal(t, step+1, state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, preimageOffset, state.PreimageOffset)

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallMmap(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	step := uint64(0)
	f.Fuzz(func(t *testing.T, addr uint32, siz uint32, heap uint32) {
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     0,
				NextPC: 4,
				LO:     0,
				HI:     0,
			},
			Heap:           heap,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysMmap, 4: addr, 5: siz},
			Step:           step,
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		preStateRegisters := state.Registers

		goState := singlethreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.Cpu.PC)
		require.Equal(t, uint32(8), state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
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
			if sizAlign&memory.PageAddrMask != 0 { // adjust size to align with page size
				sizAlign = siz + memory.PageSize - (siz & memory.PageAddrMask)
			}
			require.Equal(t, uint32(heap+sizAlign), state.Heap)
		} else {
			expectedRegisters := preStateRegisters
			expectedRegisters[2] = addr
			require.Equal(t, expectedRegisters, state.Registers)
			require.Equal(t, uint32(heap), state.Heap)
		}

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallExitGroup(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	f.Fuzz(func(t *testing.T, exitCode uint8, pc uint32, step uint64) {
		pc = pc & 0xFF_FF_FF_FC // align PC
		nextPC := pc + 4
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     pc,
				NextPC: nextPC,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysExitGroup, 4: uint32(exitCode)},
			Step:           step,
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(pc, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		preStateRegisters := state.Registers

		goState := singlethreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, pc, state.Cpu.PC)
		require.Equal(t, nextPC, state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(exitCode), state.ExitCode)
		require.Equal(t, true, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, preStateRegisters, state.Registers)
		require.Equal(t, step+1, state.Step)
		require.Equal(t, common.Hash{}, state.PreimageKey)
		require.Equal(t, uint32(0), state.PreimageOffset)

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateSyscallFcntl(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	step := uint64(0)
	f.Fuzz(func(t *testing.T, fd uint32, cmd uint32) {
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     0,
				NextPC: 4,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysFcntl, 4: fd, 5: cmd},
			Step:           step,
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		preStateRegisters := state.Registers

		goState := singlethreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.Cpu.PC)
		require.Equal(t, uint32(8), state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
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
			case exec.FdStdin, exec.FdPreimageRead, exec.FdHintRead:
				expectedRegisters[2] = 0
			case exec.FdStdout, exec.FdStderr, exec.FdPreimageWrite, exec.FdHintWrite:
				expectedRegisters[2] = 1
			default:
				expectedRegisters[2] = 0xFF_FF_FF_FF
				expectedRegisters[7] = exec.MipsEBADF
			}
			require.Equal(t, expectedRegisters, state.Registers)
		} else {
			expectedRegisters := preStateRegisters
			expectedRegisters[2] = 0xFF_FF_FF_FF
			expectedRegisters[7] = exec.MipsEINVAL
			require.Equal(t, expectedRegisters, state.Registers)
		}

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateHintRead(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	step := uint64(0)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32) {
		preimageData := []byte("hello world")
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     0,
				NextPC: 4,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysRead, 4: exec.FdHintRead, 5: addr, 6: count},
			Step:           step,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: 0,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStatePreimageKey := state.PreimageKey
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = count

		oracle := testutil.StaticOracle(t, preimageData) // only used for hinting
		goState := singlethreaded.NewInstrumentedState(state, oracle, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.Cpu.PC)
		require.Equal(t, uint32(8), state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, preStatePreimageKey, state.PreimageKey)
		require.Equal(t, expectedRegisters, state.Registers)

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStatePreimageRead(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	step := uint64(0)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32, preimageOffset uint32) {
		preimageData := []byte("hello world")
		if preimageOffset >= uint32(len(preimageData)) {
			t.SkipNow()
		}
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     0,
				NextPC: 4,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysRead, 4: exec.FdPreimageRead, 5: addr, 6: count},
			Step:           step,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: preimageOffset,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStatePreimageKey := state.PreimageKey
		preStateRoot := state.Memory.MerkleRoot()
		writeLen := count
		if writeLen > 4 {
			writeLen = 4
		}
		if preimageOffset+writeLen > uint32(8+len(preimageData)) {
			writeLen = uint32(8+len(preimageData)) - preimageOffset
		}
		oracle := testutil.StaticOracle(t, preimageData)

		goState := singlethreaded.NewInstrumentedState(state, oracle, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.True(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.Cpu.PC)
		require.Equal(t, uint32(8), state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		if writeLen > 0 {
			// Memory may be unchanged if we're writing the first zero-valued 7 bytes of the pre-image.
			//require.NotEqual(t, preStateRoot, state.Memory.MerkleRoot())
			require.Greater(t, state.PreimageOffset, preimageOffset)
		} else {
			require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
			require.Equal(t, state.PreimageOffset, preimageOffset)
		}
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, preStatePreimageKey, state.PreimageKey)

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStateHintWrite(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	step := uint64(0)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32, randSeed int64) {
		preimageData := []byte("hello world")
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     0,
				NextPC: 4,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysWrite, 4: exec.FdHintWrite, 5: addr, 6: count},
			Step:           step,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: 0,
			LastHint:       nil,
		}
		// Set random data at the target memory range
		randBytes, err := randomBytes(randSeed, count)
		require.NoError(t, err)
		err = state.Memory.SetMemoryRange(addr, bytes.NewReader(randBytes))
		require.NoError(t, err)
		// Set syscall instruction
		state.Memory.SetMemory(0, syscallInsn)

		preStatePreimageKey := state.PreimageKey
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		expectedRegisters[2] = count

		oracle := testutil.StaticOracle(t, preimageData) // only used for hinting
		goState := singlethreaded.NewInstrumentedState(state, oracle, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.Cpu.PC)
		require.Equal(t, uint32(8), state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, preStatePreimageKey, state.PreimageKey)
		require.Equal(t, expectedRegisters, state.Registers)

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func FuzzStatePreimageWrite(f *testing.F) {
	contracts, addrs := testContractsSetup(f)
	step := uint64(0)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32) {
		preimageData := []byte("hello world")
		state := &singlethreaded.State{
			Cpu: mipsevm.CpuScalars{
				PC:     0,
				NextPC: 4,
				LO:     0,
				HI:     0,
			},
			Heap:           0,
			ExitCode:       0,
			Exited:         false,
			Memory:         memory.NewMemory(),
			Registers:      [32]uint32{2: exec.SysWrite, 4: exec.FdPreimageWrite, 5: addr, 6: count},
			Step:           0,
			PreimageKey:    preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey(),
			PreimageOffset: 128,
		}
		state.Memory.SetMemory(0, syscallInsn)
		preStateRoot := state.Memory.MerkleRoot()
		expectedRegisters := state.Registers
		sz := 4 - (addr & 0x3)
		if sz < count {
			count = sz
		}
		expectedRegisters[2] = count

		oracle := testutil.StaticOracle(t, preimageData)
		goState := singlethreaded.NewInstrumentedState(state, oracle, os.Stdout, os.Stderr, nil)
		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		require.False(t, stepWitness.HasPreimage())

		require.Equal(t, uint32(4), state.Cpu.PC)
		require.Equal(t, uint32(8), state.Cpu.NextPC)
		require.Equal(t, uint32(0), state.Cpu.LO)
		require.Equal(t, uint32(0), state.Cpu.HI)
		require.Equal(t, uint32(0), state.Heap)
		require.Equal(t, uint8(0), state.ExitCode)
		require.Equal(t, false, state.Exited)
		require.Equal(t, preStateRoot, state.Memory.MerkleRoot())
		require.Equal(t, uint64(1), state.Step)
		require.Equal(t, uint32(0), state.PreimageOffset)
		require.Equal(t, expectedRegisters, state.Registers)

		evm := testutil.NewMIPSEVM(contracts, addrs)
		evmPost := evm.Step(t, stepWitness, step, singlethreaded.GetStateHashFn())
		goPost, _ := goState.GetState().EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	})
}

func randomBytes(seed int64, length uint32) ([]byte, error) {
	r := rand.New(rand.NewSource(seed))
	randBytes := make([]byte, length)
	if _, err := r.Read(randBytes); err != nil {
		return nil, err
	}
	return randBytes, nil
}
