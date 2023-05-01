package mipsevm

import (
	"bytes"
	"debug/elf"
	"io"
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func testContractsSetup(t *testing.T) (*Contracts, *Addresses, *SourceMapTracer) {
	contracts, err := LoadContracts()
	require.NoError(t, err)

	mipsSrcMap, err := contracts.MIPS.SourceMap([]string{"../contracts/src/MIPS.sol"})
	require.NoError(t, err)
	oracleSrcMap, err := contracts.Oracle.SourceMap([]string{"../contracts/src/Oracle.sol"})
	require.NoError(t, err)

	addrs := &Addresses{
		MIPS:         common.Address{0: 0xff, 19: 1},
		Oracle:       common.Address{0: 0xff, 19: 2},
		Sender:       common.Address{0x13, 0x37},
		FeeRecipient: common.Address{0xaa},
	}
	tracer := NewSourceMapTracer(map[common.Address]*SourceMap{addrs.MIPS: mipsSrcMap, addrs.Oracle: oracleSrcMap}, os.Stdout)
	return contracts, addrs, tracer
}

func TestEVM(t *testing.T) {
	testFiles, err := os.ReadDir("open_mips_tests/test/bin")
	require.NoError(t, err)

	contracts, addrs, tracer := testContractsSetup(t)
	sender := common.Address{0x13, 0x37}
	//tracer = logger.NewMarkdownLogger(&logger.Config{}, os.Stdout)

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			if f.Name() == "oracle.bin" {
				t.Skip("oracle test needs to be updated to use syscall pre-image oracle")
			}

			env, evmState := NewEVMEnv(contracts, addrs)
			env.Config.Debug = false
			env.Config.Tracer = tracer

			fn := path.Join("open_mips_tests/test/bin", f.Name())
			programMem, err := os.ReadFile(fn)
			require.NoError(t, err)
			state := &State{PC: 0, NextPC: 4, Memory: NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			mu, err := NewUnicorn()
			require.NoError(t, err, "load unicorn")
			defer mu.Close()

			require.NoError(t, mu.MemMap(baseAddrStart, ((baseAddrEnd-baseAddrStart)&^pageAddrMask)+pageSize))
			require.NoError(t, mu.MemMap(endAddr&^pageAddrMask, pageSize))

			err = LoadUnicorn(state, mu)
			require.NoError(t, err, "load state into unicorn")

			us, err := NewUnicornState(mu, state, nil, os.Stdout, os.Stderr)
			require.NoError(t, err, "hook unicorn to state")

			for i := 0; i < 1000; i++ {
				if us.state.PC == endAddr {
					break
				}
				insn := state.Memory.GetMemory(state.PC)
				t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.PC, insn)

				stepWitness := us.Step(true)
				input := stepWitness.EncodeStepInput()
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
				evmPost := logs[0].Data
				require.Equal(t, crypto.Keccak256Hash(evmPost), postHash, "logged state must be accurate")
				env.StateDB.RevertToSnapshot(snap)

				t.Logf("EVM step took %d gas, and returned stateHash %s", startingGas-leftOverGas, postHash)

				// verify the post-state matches.
				// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
				uniPost := us.state.EncodeWitness()
				require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
					"unicorn produced different state than EVM")
			}
			require.Equal(t, uint32(endAddr), state.PC, "must reach end")
			// inspect test result
			done, result := state.Memory.GetMemory(baseAddrEnd+4), state.Memory.GetMemory(baseAddrEnd+8)
			require.Equal(t, done, uint32(1), "must be done")
			require.Equal(t, result, uint32(1), "must have success result")
		})
	}
}

func TestHelloEVM(t *testing.T) {
	contracts, addrs, tracer := testContractsSetup(t)
	sender := common.Address{0x13, 0x37}

	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = patchVM(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")

	mu, err := NewUnicorn()
	require.NoError(t, err, "load unicorn")
	defer mu.Close()
	err = LoadUnicorn(state, mu)
	require.NoError(t, err, "load state into unicorn")
	var stdOutBuf, stdErrBuf bytes.Buffer
	us, err := NewUnicornState(mu, state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))
	require.NoError(t, err, "hook unicorn to state")

	env, evmState := NewEVMEnv(contracts, addrs)
	env.Config.Debug = false
	env.Config.Tracer = tracer

	start := time.Now()
	for i := 0; i < 400_000; i++ {
		if us.state.Exited {
			break
		}
		insn := state.Memory.GetMemory(state.PC)
		if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
			t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.PC, insn)
		}

		stepWitness := us.Step(true)
		input := stepWitness.EncodeStepInput()
		startingGas := uint64(30_000_000)

		// we take a snapshot so we can clean up the state, and isolate the logs of this instruction run.
		snap := env.StateDB.Snapshot()
		ret, leftOverGas, err := env.Call(vm.AccountRef(sender), addrs.MIPS, input, startingGas, big.NewInt(0))
		require.NoErrorf(t, err, "evm should not fail, took %d gas", startingGas-leftOverGas)
		require.Len(t, ret, 32, "expecting 32-byte state hash")
		// remember state hash, to check it against state
		postHash := common.Hash(*(*[32]byte)(ret))
		logs := evmState.Logs()
		require.Equal(t, 1, len(logs), "expecting a log with post-state")
		evmPost := logs[0].Data
		require.Equal(t, crypto.Keccak256Hash(evmPost), postHash, "logged state must be accurate")
		env.StateDB.RevertToSnapshot(snap)

		//t.Logf("EVM step took %d gas, and returned stateHash %s", startingGas-leftOverGas, postHash)

		// verify the post-state matches.
		// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
		uniPost := us.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(uniPost).String(), hexutil.Bytes(evmPost).String(),
			"unicorn produced different state than EVM")
	}
	end := time.Now()
	delta := end.Sub(start)
	t.Logf("test took %s, %d instructions, %s per instruction", delta, state.Step, delta/time.Duration(state.Step))

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, "hello world!", stdOutBuf.String(), "stdout says hello")
	require.Equal(t, "", stdErrBuf.String(), "stderr silent")
}

func TestClaimEVM(t *testing.T) {
	contracts, addrs, tracer := testContractsSetup(t)

	elfProgram, err := elf.Open("../example/bin/claim.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = patchVM(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")

	mu, err := NewUnicorn()
	require.NoError(t, err, "load unicorn")
	defer mu.Close()
	err = LoadUnicorn(state, mu)
	require.NoError(t, err, "load state into unicorn")

	oracle, expectedStdOut, expectedStdErr := claimTestOracle(t)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us, err := NewUnicornState(mu, state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))
	require.NoError(t, err, "hook unicorn to state")

	env, evmState := NewEVMEnv(contracts, addrs)
	env.Config.Debug = false
	env.Config.Tracer = tracer

	for i := 0; i < 2000_000; i++ {
		if us.state.Exited {
			break
		}

		insn := state.Memory.GetMemory(state.PC)
		if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
			t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.PC, insn)
		}

		stepWitness := us.Step(true)
		input := stepWitness.EncodeStepInput()
		startingGas := uint64(30_000_000)

		// we take a snapshot so we can clean up the state, and isolate the logs of this instruction run.
		snap := env.StateDB.Snapshot()

		// prepare pre-image oracle data, if any
		if stepWitness.HasPreimage() {
			poInput, err := stepWitness.EncodePreimageOracleInput()
			require.NoError(t, err, "encode preimage oracle input")
			_, leftOverGas, err := env.Call(vm.AccountRef(addrs.Sender), addrs.Oracle, poInput, startingGas, big.NewInt(0))
			require.NoErrorf(t, err, "evm should not fail, took %d gas", startingGas-leftOverGas)
		}

		ret, leftOverGas, err := env.Call(vm.AccountRef(addrs.Sender), addrs.MIPS, input, startingGas, big.NewInt(0))
		require.NoErrorf(t, err, "evm should not fail, took %d gas", startingGas-leftOverGas)
		require.Len(t, ret, 32, "expecting 32-byte state hash")
		// remember state hash, to check it against state
		postHash := common.Hash(*(*[32]byte)(ret))
		logs := evmState.Logs()
		require.Equal(t, 1, len(logs), "expecting a log with post-state")
		evmPost := logs[0].Data
		require.Equal(t, crypto.Keccak256Hash(evmPost), postHash, "logged state must be accurate")
		env.StateDB.RevertToSnapshot(snap)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
	require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
}
