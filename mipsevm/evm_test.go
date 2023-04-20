package main

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/stretchr/testify/require"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func TestEVM(t *testing.T) {
	t.Skip("work in progress!")

	testFiles, err := os.ReadDir("test/bin")
	require.NoError(t, err)

	contracts, err := LoadContracts()
	require.NoError(t, err)

	addrs := &Addresses{
		MIPS:       common.Address{0: 0xff, 19: 1},
		MIPSMemory: common.Address{0: 0xff, 19: 2},
		Challenge:  common.Address{0: 0xff, 19: 3},
	}
	sender := common.Address{0x13, 0x37}

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			if f.Name() == "oracle.bin" {
				t.Skip("oracle test needs to be updated to use syscall pre-image oracle")
			}

			env := NewEVMEnv(contracts, addrs)
			env.Config.Debug = true
			env.Config.Tracer = logger.NewMarkdownLogger(&logger.Config{}, os.Stdout)

			fn := path.Join("test/bin", f.Name())
			programMem, err := os.ReadFile(fn)
			state := &State{PC: 0, Memory: make(map[uint32]*Page)}
			err = state.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			mu, err := NewUnicorn()
			require.NoError(t, err, "load unicorn")
			defer mu.Close()

			require.NoError(t, mu.MemMap(baseAddrStart, ((baseAddrEnd-baseAddrStart)&^pageAddrMask)+pageSize))
			require.NoError(t, mu.MemMap(endAddr&^pageAddrMask, pageSize))

			al := &AccessList{}

			err = LoadUnicorn(state, mu)
			require.NoError(t, err, "load state into unicorn")
			err = HookUnicorn(state, mu, os.Stdout, os.Stderr, al)
			require.NoError(t, err, "hook unicorn to state")

			// Add hook to stop unicorn once we reached the end of the test (i.e. "ate food")
			_, err = mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
				if state.PC == endAddr {
					require.NoError(t, mu.Stop(), "stop test when returned")
				}
			}, 0, ^uint64(0))
			require.NoError(t, err, "")

			so := NewStateCache()
			for i := 0; i < 1000; i++ {
				insn := state.GetMemory(state.PC)

				al.Reset() // reset
				require.NoError(t, RunUnicorn(mu, state.PC, 1))
				require.LessOrEqual(t, len(al.memReads)+len(al.memWrites), 1, "expecting at most a single mem read or write")

				proofData := make([]byte, 0, 32*2)
				proofData = append(proofData, uint32ToBytes32(32)...) // length in bytes
				var tmp [32]byte
				binary.BigEndian.PutUint32(tmp[0:4], insn) // instruction
				if len(al.memReads) > 0 {
					binary.BigEndian.PutUint32(tmp[4:8], state.GetMemory(al.memReads[0]))
				}
				if len(al.memWrites) > 0 {
					binary.BigEndian.PutUint32(tmp[4:8], state.GetMemory(al.memWrites[0]))
				}
				proofData = append(proofData, tmp[:]...)

				memRoot := state.MerkleizeMemory(so)

				stateData := make([]byte, 0, 44*32)
				stateData = append(stateData, memRoot[:]...)
				stateData = append(stateData, make([]byte, 32)...) // TODO preimageKey
				stateData = append(stateData, make([]byte, 32)...) // TODO preimageOffset
				for i := 0; i < 32; i++ {
					stateData = append(stateData, uint32ToBytes32(state.Registers[i])...)
				}
				stateData = append(stateData, uint32ToBytes32(state.PC)...)
				stateData = append(stateData, uint32ToBytes32(state.NextPC)...)
				stateData = append(stateData, uint32ToBytes32(state.LR)...)
				stateData = append(stateData, uint32ToBytes32(state.LO)...)
				stateData = append(stateData, uint32ToBytes32(state.HI)...)
				stateData = append(stateData, uint32ToBytes32(state.Heap)...)
				stateData = append(stateData, uint8ToBytes32(state.ExitCode)...)
				stateData = append(stateData, boolToBytes32(state.Exited)...)
				stateData = append(stateData, uint64ToBytes32(state.Step)...)

				stateHash := crypto.Keccak256Hash(stateData)
				var input []byte
				input = append(input, StepBytes4...)
				input = append(input, stateHash[:]...)
				input = append(input, uint32ToBytes32(32*3)...)                           // state data offset in bytes
				input = append(input, uint32ToBytes32(32*3+32+uint32(len(stateData)))...) // proof data offset in bytes

				input = append(input, uint32ToBytes32(uint32(len(stateData)))...) // state data length in bytes
				input = append(input, stateData[:]...)
				input = append(input, uint32ToBytes32(uint32(len(proofData)))...) // proof data length in bytes
				input = append(input, proofData[:]...)
				startingGas := uint64(30_000_000)
				ret, leftOverGas, err := env.Call(vm.AccountRef(sender), addrs.MIPS, input, startingGas, big.NewInt(0))
				require.NoError(t, err, "evm should not fail")
				t.Logf("step took %d gas", startingGas-leftOverGas)
				t.Logf("output (state hash): %x", ret)
				// TODO compare output against unicorn (need to reconstruct state and memory hash)
			}

			require.NoError(t, err, "must run steps without error")
			// inspect test result
			done, result := state.GetMemory(baseAddrEnd+4), state.GetMemory(baseAddrEnd+8)
			require.Equal(t, done, uint32(1), "must be done")
			require.Equal(t, result, uint32(1), "must have success result")
		})
	}
}

func uint64ToBytes32(v uint64) []byte {
	var out [32]byte
	binary.BigEndian.PutUint64(out[32-8:], v)
	return out[:]
}

func uint32ToBytes32(v uint32) []byte {
	var out [32]byte
	binary.BigEndian.PutUint32(out[32-4:], v)
	return out[:]
}

func uint8ToBytes32(v uint8) []byte {
	var out [32]byte
	out[31] = v
	return out[:]
}

func boolToBytes32(v bool) []byte {
	var out [32]byte
	if v {
		out[31] = 1
	}
	return out[:]
}
