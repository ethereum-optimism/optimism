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

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

const syscallInsn = uint32(0x00_00_00_0c)

func FuzzStateSyscallBrk(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, pc uint32, step uint64, preimageOffset uint32) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				pc = pc & 0xFF_FF_FF_FC // align PC
				nextPC := pc + 4

				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithPC(pc), WithNextPC(nextPC), WithStep(step), WithPreimageOffset(preimageOffset))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysBrk
				state.GetMemory().SetMemory(pc, syscallInsn)

				preStateRoot := state.GetMemory().MerkleRoot()
				expectedRegisters := testutil.CopyRegisters(state)
				expectedRegisters[2] = program.PROGRAM_BREAK

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.False(t, stepWitness.HasPreimage())

				require.Equal(t, pc+4, state.GetPC())
				require.Equal(t, nextPC+4, state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, uint32(0), state.GetHeap())
				require.Equal(t, uint8(0), state.GetExitCode())
				require.Equal(t, false, state.GetExited())
				require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
				require.Equal(t, expectedRegisters, state.GetRegistersRef())
				require.Equal(t, step+1, state.GetStep())
				require.Equal(t, common.Hash{}, state.GetPreimageKey())
				require.Equal(t, preimageOffset, state.GetPreimageOffset())

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
	})
}

func FuzzStateSyscallMmap(f *testing.F) {
	// Add special cases for large memory allocation
	f.Add(uint32(0), uint32(0x1000), uint32(program.HEAP_END), int64(1))
	f.Add(uint32(0), uint32(1<<31), uint32(program.HEAP_START), int64(2))
	// Check edge case - just within bounds
	f.Add(uint32(0), uint32(0x1000), uint32(program.HEAP_END-4096), int64(3))

	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, addr uint32, siz uint32, heap uint32, seed int64) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				step := uint64(0)
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithStep(step), WithHeap(heap))
				state := goVm.GetState()
				*state.GetRegistersRef() = testutil.RandomRegisters(seed)
				state.GetRegistersRef()[2] = exec.SysMmap
				state.GetRegistersRef()[4] = addr
				state.GetRegistersRef()[5] = siz
				state.GetMemory().SetMemory(0, syscallInsn)

				preStateRoot := state.GetMemory().MerkleRoot()
				preStateRegisters := testutil.CopyRegisters(state)

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.False(t, stepWitness.HasPreimage())

				var expectedHeap uint32
				expectedRegisters := preStateRegisters
				if addr == 0 {
					sizAlign := siz
					if sizAlign&memory.PageAddrMask != 0 { // adjust size to align with page size
						sizAlign = siz + memory.PageSize - (siz & memory.PageAddrMask)
					}
					newHeap := heap + sizAlign
					if newHeap > program.HEAP_END || newHeap < heap || sizAlign < siz {
						expectedHeap = heap
						expectedRegisters[2] = exec.SysErrorSignal
						expectedRegisters[7] = exec.MipsEINVAL
					} else {
						expectedRegisters[2] = heap
						expectedRegisters[7] = 0 // no error
						expectedHeap = heap + sizAlign
					}
				} else {
					expectedRegisters[2] = addr
					expectedRegisters[7] = 0 // no error
					expectedHeap = heap
				}

				require.Equal(t, uint32(4), state.GetCpu().PC)
				require.Equal(t, uint32(8), state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
				require.Equal(t, uint64(1), state.GetStep())
				require.Equal(t, common.Hash{}, state.GetPreimageKey())
				require.Equal(t, uint32(0), state.GetPreimageOffset())
				require.Equal(t, expectedHeap, state.GetHeap())
				require.Equal(t, uint8(0), state.GetExitCode())
				require.Equal(t, false, state.GetExited())
				require.Equal(t, expectedRegisters, state.GetRegistersRef())

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
	})
}

func FuzzStateSyscallExitGroup(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, exitCode uint8, pc uint32, step uint64) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				pc = pc & 0xFF_FF_FF_FC // align PC
				nextPC := pc + 4
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithPC(pc), WithNextPC(nextPC), WithStep(step))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysExitGroup
				state.GetRegistersRef()[4] = uint32(exitCode)
				state.GetMemory().SetMemory(pc, syscallInsn)

				preStateRoot := state.GetMemory().MerkleRoot()
				preStateRegisters := testutil.CopyRegisters(state)

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.False(t, stepWitness.HasPreimage())

				require.Equal(t, pc, state.GetCpu().PC)
				require.Equal(t, nextPC, state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, uint32(0), state.GetHeap())
				require.Equal(t, uint8(exitCode), state.GetExitCode())
				require.Equal(t, true, state.GetExited())
				require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
				require.Equal(t, preStateRegisters, state.GetRegistersRef())
				require.Equal(t, step+1, state.GetStep())
				require.Equal(t, common.Hash{}, state.GetPreimageKey())
				require.Equal(t, uint32(0), state.GetPreimageOffset())

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
	})
}

func FuzzStateSyscallFcntl(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, fd uint32, cmd uint32) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				step := uint64(0)
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithStep(step))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysFcntl
				state.GetRegistersRef()[4] = fd
				state.GetRegistersRef()[5] = cmd
				state.GetMemory().SetMemory(0, syscallInsn)

				preStateRoot := state.GetMemory().MerkleRoot()
				preStateRegisters := testutil.CopyRegisters(state)

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.False(t, stepWitness.HasPreimage())

				require.Equal(t, uint32(4), state.GetCpu().PC)
				require.Equal(t, uint32(8), state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, uint32(0), state.GetHeap())
				require.Equal(t, uint8(0), state.GetExitCode())
				require.Equal(t, false, state.GetExited())
				require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
				require.Equal(t, uint64(1), state.GetStep())
				require.Equal(t, common.Hash{}, state.GetPreimageKey())
				require.Equal(t, uint32(0), state.GetPreimageOffset())
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
					require.Equal(t, expectedRegisters, state.GetRegistersRef())
				} else {
					expectedRegisters := preStateRegisters
					expectedRegisters[2] = 0xFF_FF_FF_FF
					expectedRegisters[7] = exec.MipsEINVAL
					require.Equal(t, expectedRegisters, state.GetRegistersRef())
				}

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
	})
}

func FuzzStateHintRead(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				step := uint64(0)
				preimageData := []byte("hello world")
				preimageKey := preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey()
				oracle := testutil.StaticOracle(t, preimageData) // only used for hinting

				goVm := v.VMFactory(oracle, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithStep(step), WithPreimageKey(preimageKey))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysRead
				state.GetRegistersRef()[4] = exec.FdHintRead
				state.GetRegistersRef()[5] = addr
				state.GetRegistersRef()[6] = count
				state.GetMemory().SetMemory(0, syscallInsn)

				preStatePreimageKey := state.GetPreimageKey()
				preStateRoot := state.GetMemory().MerkleRoot()
				expectedRegisters := testutil.CopyRegisters(state)
				expectedRegisters[2] = count

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.False(t, stepWitness.HasPreimage())

				require.Equal(t, uint32(4), state.GetCpu().PC)
				require.Equal(t, uint32(8), state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, uint32(0), state.GetHeap())
				require.Equal(t, uint8(0), state.GetExitCode())
				require.Equal(t, false, state.GetExited())
				require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
				require.Equal(t, uint64(1), state.GetStep())
				require.Equal(t, preStatePreimageKey, state.GetPreimageKey())
				require.Equal(t, expectedRegisters, state.GetRegistersRef())

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
	})
}

func FuzzStatePreimageRead(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32, preimageOffset uint32) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				step := uint64(0)
				preimageData := []byte("hello world")
				if preimageOffset >= uint32(len(preimageData)) {
					t.SkipNow()
				}
				preimageKey := preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey()
				oracle := testutil.StaticOracle(t, preimageData)

				goVm := v.VMFactory(oracle, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithStep(step), WithPreimageKey(preimageKey), WithPreimageOffset(preimageOffset))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysRead
				state.GetRegistersRef()[4] = exec.FdPreimageRead
				state.GetRegistersRef()[5] = addr
				state.GetRegistersRef()[6] = count
				state.GetMemory().SetMemory(0, syscallInsn)

				preStatePreimageKey := state.GetPreimageKey()
				preStateRoot := state.GetMemory().MerkleRoot()
				writeLen := count
				if writeLen > 4 {
					writeLen = 4
				}
				if preimageOffset+writeLen > uint32(8+len(preimageData)) {
					writeLen = uint32(8+len(preimageData)) - preimageOffset
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.True(t, stepWitness.HasPreimage())

				require.Equal(t, uint32(4), state.GetCpu().PC)
				require.Equal(t, uint32(8), state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, uint32(0), state.GetHeap())
				require.Equal(t, uint8(0), state.GetExitCode())
				require.Equal(t, false, state.GetExited())
				if writeLen > 0 {
					// Memory may be unchanged if we're writing the first zero-valued 7 bytes of the pre-image.
					//require.NotEqual(t, preStateRoot, state.GetMemory().MerkleRoot())
					require.Greater(t, state.GetPreimageOffset(), preimageOffset)
				} else {
					require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
					require.Equal(t, state.GetPreimageOffset(), preimageOffset)
				}
				require.Equal(t, uint64(1), state.GetStep())
				require.Equal(t, preStatePreimageKey, state.GetPreimageKey())

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
	})
}

func FuzzStateHintWrite(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32, randSeed int64) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				step := uint64(0)
				preimageData := []byte("hello world")
				preimageKey := preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey()
				oracle := testutil.StaticOracle(t, preimageData) // only used for hinting

				goVm := v.VMFactory(oracle, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithStep(step), WithPreimageKey(preimageKey))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysWrite
				state.GetRegistersRef()[4] = exec.FdHintWrite
				state.GetRegistersRef()[5] = addr
				state.GetRegistersRef()[6] = count

				// Set random data at the target memory range
				randBytes, err := randomBytes(randSeed, count)
				require.NoError(t, err)
				err = state.GetMemory().SetMemoryRange(addr, bytes.NewReader(randBytes))
				require.NoError(t, err)
				// Set syscall instruction
				state.GetMemory().SetMemory(0, syscallInsn)

				preStatePreimageKey := state.GetPreimageKey()
				preStateRoot := state.GetMemory().MerkleRoot()
				expectedRegisters := testutil.CopyRegisters(state)
				expectedRegisters[2] = count

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.False(t, stepWitness.HasPreimage())

				require.Equal(t, uint32(4), state.GetCpu().PC)
				require.Equal(t, uint32(8), state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, uint32(0), state.GetHeap())
				require.Equal(t, uint8(0), state.GetExitCode())
				require.Equal(t, false, state.GetExited())
				require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
				require.Equal(t, uint64(1), state.GetStep())
				require.Equal(t, preStatePreimageKey, state.GetPreimageKey())
				require.Equal(t, expectedRegisters, state.GetRegistersRef())

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
	})
}

func FuzzStatePreimageWrite(f *testing.F) {
	versions := GetMipsVersionTestCases(f)
	f.Fuzz(func(t *testing.T, addr uint32, count uint32) {
		for _, v := range versions {
			t.Run(v.Name, func(t *testing.T) {
				step := uint64(0)
				preimageData := []byte("hello world")
				preimageKey := preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey()
				oracle := testutil.StaticOracle(t, preimageData)

				goVm := v.VMFactory(oracle, os.Stdout, os.Stderr, testutil.CreateLogger(),
					WithStep(step), WithPreimageKey(preimageKey), WithPreimageOffset(128))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysWrite
				state.GetRegistersRef()[4] = exec.FdPreimageWrite
				state.GetRegistersRef()[5] = addr
				state.GetRegistersRef()[6] = count
				state.GetMemory().SetMemory(0, syscallInsn)

				preStateRoot := state.GetMemory().MerkleRoot()
				expectedRegisters := testutil.CopyRegisters(state)
				sz := 4 - (addr & 0x3)
				if sz < count {
					count = sz
				}
				expectedRegisters[2] = count

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				require.False(t, stepWitness.HasPreimage())

				require.Equal(t, uint32(4), state.GetCpu().PC)
				require.Equal(t, uint32(8), state.GetCpu().NextPC)
				require.Equal(t, uint32(0), state.GetCpu().LO)
				require.Equal(t, uint32(0), state.GetCpu().HI)
				require.Equal(t, uint32(0), state.GetHeap())
				require.Equal(t, uint8(0), state.GetExitCode())
				require.Equal(t, false, state.GetExited())
				require.Equal(t, preStateRoot, state.GetMemory().MerkleRoot())
				require.Equal(t, uint64(1), state.GetStep())
				require.Equal(t, uint32(0), state.GetPreimageOffset())
				require.Equal(t, expectedRegisters, state.GetRegistersRef())

				evm := testutil.NewMIPSEVM(v.Contracts)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			})
		}
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
