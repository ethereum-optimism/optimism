package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	uc "github.com/unicorn-engine/unicorn/bindings/go/unicorn"
)

func TestEVM(t *testing.T) {

	testFiles, err := os.ReadDir("test/bin")
	require.NoError(t, err)

	contracts, err := LoadContracts()
	require.NoError(t, err)

	// the first unlisted source seems to be the ABIDecoderV2 code that the compiler inserts
	mipsSrcMap, err := contracts.MIPS.SourceMap([]string{"../contracts/src/MIPS.sol", "~compiler?", "../contracts/src/MIPS.sol"})
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

			env, evmState := NewEVMEnv(contracts, addrs)
			env.Config.Debug = false
			//env.Config.Tracer = logger.NewMarkdownLogger(&logger.Config{}, os.Stdout)
			env.Config.Tracer = mipsSrcMap.Tracer(os.Stdout)

			fn := path.Join("test/bin", f.Name())
			programMem, err := os.ReadFile(fn)
			state := &State{PC: 0, NextPC: 4, Memory: make(map[uint32]*Page)}
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

			so := NewStateCache()
			var stateData []byte
			var insn uint32
			var pc uint32
			var post []byte
			preCode := func() {
				insn = state.GetMemory(state.PC)
				pc = state.PC
				fmt.Printf("PRE - pc: %08x insn: %08x\n", pc, insn)
				// remember the pre-state, to repeat it in the EVM during the post processing step
				stateData = state.EncodeWitness(so)
				if post != nil {
					require.Equal(t, hexutil.Bytes(stateData).String(), hexutil.Bytes(post).String(),
						"unicorn produced different state than EVM")
				}

				al.Reset() // reset access list
			}
			postCode := func() {
				fmt.Printf("POST - pc: %08x insn: %08x\n", pc, insn)

				var proofData []byte
				proofData = binary.BigEndian.AppendUint32(proofData, insn)
				if len(al.memReads) > 0 {
					proofData = binary.BigEndian.AppendUint32(proofData, al.memReads[0].PreValue)
				} else if len(al.memWrites) > 0 {
					proofData = binary.BigEndian.AppendUint32(proofData, al.memWrites[0].PreValue)
				} else {
					proofData = append(proofData, make([]byte, 4)...)
				}
				proofData = append(proofData, make([]byte, 32-4-4)...)

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

				// we take a snapshot so we can clean up the state, and isolate the logs of this instruction run.
				snap := env.StateDB.Snapshot()
				ret, leftOverGas, err := env.Call(vm.AccountRef(sender), addrs.MIPS, input, startingGas, big.NewInt(0))
				require.NoError(t, err, "evm should not fail")
				require.Len(t, ret, 32, "expecting 32-byte state hash")
				// remember state hash, to check it against state
				postHash := common.Hash(*(*[32]byte)(ret))
				logs := evmState.Logs()
				require.Equal(t, 1, len(logs), "expecting a log with post-state")
				post = logs[0].Data
				require.Equal(t, crypto.Keccak256Hash(post), postHash, "logged state must be accurate")
				env.StateDB.RevertToSnapshot(snap)

				t.Logf("EVM step took %d gas, and returned stateHash %s", startingGas-leftOverGas, postHash)
			}

			firstStep := true
			_, err = mu.HookAdd(uc.HOOK_CODE, func(mu uc.Unicorn, addr uint64, size uint32) {
				if state.PC == endAddr {
					require.NoError(t, mu.Stop(), "stop test when returned")
				}
				if !firstStep {
					postCode()
				}
				preCode()
				firstStep = false
			}, 0, ^uint64(0))
			require.NoError(t, err, "hook code")

			err = RunUnicorn(mu, state.PC, 1000)
			require.NoError(t, err, "must run steps without error")

			// inspect test result
			done, result := state.GetMemory(baseAddrEnd+4), state.GetMemory(baseAddrEnd+8)
			require.Equal(t, done, uint32(1), "must be done")
			require.Equal(t, result, uint32(1), "must have success result")
		})
	}
}

func uint32ToBytes32(v uint32) []byte {
	var out [32]byte
	binary.BigEndian.PutUint32(out[32-4:], v)
	return out[:]
}
