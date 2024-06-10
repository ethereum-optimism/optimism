package mipsevm

import (
	"bytes"
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-chain-ops/foundry"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/stretchr/testify/require"
)

func testContractsSetup(t require.TestingT) (*Artifacts, *Addresses) {
	artifacts, err := LoadArtifacts()
	require.NoError(t, err)

	addrs := &Addresses{
		MIPS:         common.Address{0: 0xff, 19: 1},
		Oracle:       common.Address{0: 0xff, 19: 2},
		Sender:       common.Address{0x13, 0x37},
		FeeRecipient: common.Address{0xaa},
	}

	return artifacts, addrs
}

func MarkdownTracer() vm.EVMLogger {
	return logger.NewMarkdownLogger(&logger.Config{}, os.Stdout)
}

type MIPSEVM struct {
	env         *vm.EVM
	evmState    *state.StateDB
	addrs       *Addresses
	localOracle PreimageOracle
	artifacts   *Artifacts
}

func NewMIPSEVM(artifacts *Artifacts, addrs *Addresses) *MIPSEVM {
	env, evmState := NewEVMEnv(artifacts, addrs)
	return &MIPSEVM{env, evmState, addrs, nil, artifacts}
}

func (m *MIPSEVM) SetTracer(tracer vm.EVMLogger) {
	m.env.Config.Tracer = tracer
}

func (m *MIPSEVM) SetLocalOracle(oracle PreimageOracle) {
	m.localOracle = oracle
}

// Step is a pure function that computes the poststate from the VM state encoded in the StepWitness.
func (m *MIPSEVM) Step(t *testing.T, stepWitness *StepWitness) []byte {
	sender := common.Address{0x13, 0x37}
	startingGas := uint64(30_000_000)

	// we take a snapshot so we can clean up the state, and isolate the logs of this instruction run.
	snap := m.env.StateDB.Snapshot()

	if stepWitness.HasPreimage() {
		t.Logf("reading preimage key %x at offset %d", stepWitness.PreimageKey, stepWitness.PreimageOffset)
		poInput, err := encodePreimageOracleInput(t, stepWitness, LocalContext{}, m.localOracle, m.artifacts.Oracle)
		require.NoError(t, err, "encode preimage oracle input")
		_, leftOverGas, err := m.env.Call(vm.AccountRef(sender), m.addrs.Oracle, poInput, startingGas, common.U2560)
		require.NoErrorf(t, err, "evm should not fail, took %d gas", startingGas-leftOverGas)
	}

	input := encodeStepInput(t, stepWitness, LocalContext{}, m.artifacts.MIPS)
	ret, leftOverGas, err := m.env.Call(vm.AccountRef(sender), m.addrs.MIPS, input, startingGas, common.U2560)
	require.NoError(t, err, "evm should not fail")
	require.Len(t, ret, 32, "expecting 32-byte state hash")
	// remember state hash, to check it against state
	postHash := common.Hash(*(*[32]byte)(ret))
	logs := m.evmState.Logs()
	require.Equal(t, 1, len(logs), "expecting a log with post-state")
	evmPost := logs[0].Data

	stateHash, err := StateWitness(evmPost).StateHash()
	require.NoError(t, err, "state hash could not be computed")
	require.Equal(t, stateHash, postHash, "logged state must be accurate")

	m.env.StateDB.RevertToSnapshot(snap)
	t.Logf("EVM step took %d gas, and returned stateHash %s", startingGas-leftOverGas, postHash)
	return evmPost
}

func encodeStepInput(t *testing.T, wit *StepWitness, localContext LocalContext, mips *foundry.Artifact) []byte {
	input, err := mips.ABI.Pack("step", wit.State, wit.MemProof, localContext)
	require.NoError(t, err)
	return input
}

func encodePreimageOracleInput(t *testing.T, wit *StepWitness, localContext LocalContext, localOracle PreimageOracle, oracle *foundry.Artifact) ([]byte, error) {
	if wit.PreimageKey == ([32]byte{}) {
		return nil, errors.New("cannot encode pre-image oracle input, witness has no pre-image to proof")
	}

	switch preimage.KeyType(wit.PreimageKey[0]) {
	case preimage.LocalKeyType:
		if len(wit.PreimageValue) > 32+8 {
			return nil, fmt.Errorf("local pre-image exceeds maximum size of 32 bytes with key 0x%x", wit.PreimageKey)
		}
		preimagePart := wit.PreimageValue[8:]
		var tmp [32]byte
		copy(tmp[:], preimagePart)
		input, err := oracle.ABI.Pack("loadLocalData",
			new(big.Int).SetBytes(wit.PreimageKey[1:]),
			localContext,
			tmp,
			new(big.Int).SetUint64(uint64(len(preimagePart))),
			new(big.Int).SetUint64(uint64(wit.PreimageOffset)),
		)
		require.NoError(t, err)
		return input, nil
	case preimage.Keccak256KeyType:
		input, err := oracle.ABI.Pack(
			"loadKeccak256PreimagePart",
			new(big.Int).SetUint64(uint64(wit.PreimageOffset)),
			wit.PreimageValue[8:])
		require.NoError(t, err)
		return input, nil
	case preimage.PrecompileKeyType:
		if localOracle == nil {
			return nil, fmt.Errorf("local oracle is required for precompile preimages")
		}
		preimage := localOracle.GetPreimage(preimage.Keccak256Key(wit.PreimageKey).PreimageKey())
		precompile := common.BytesToAddress(preimage[:20])
		callInput := preimage[20:]
		input, err := oracle.ABI.Pack(
			"loadPrecompilePreimagePart",
			new(big.Int).SetUint64(uint64(wit.PreimageOffset)),
			precompile,
			callInput,
		)
		require.NoError(t, err)
		return input, nil
	default:
		return nil, fmt.Errorf("unsupported pre-image type %d, cannot prepare preimage with key %x offset %d for oracle",
			wit.PreimageKey[0], wit.PreimageKey, wit.PreimageOffset)
	}
}

func TestEVM(t *testing.T) {
	testFiles, err := os.ReadDir("open_mips_tests/test/bin")
	require.NoError(t, err)

	contracts, addrs := testContractsSetup(t)
	var tracer vm.EVMLogger // no-tracer by default, but MarkdownTracer

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			oracle := selectOracleFixture(t, f.Name())
			// Short-circuit early for exit_group.bin
			exitGroup := f.Name() == "exit_group.bin"

			evm := NewMIPSEVM(contracts, addrs)
			evm.SetTracer(tracer)
			evm.SetLocalOracle(oracle)

			fn := path.Join("open_mips_tests/test/bin", f.Name())
			programMem, err := os.ReadFile(fn)
			require.NoError(t, err)
			state := &State{PC: 0, NextPC: 4, Memory: NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			goState := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)

			for i := 0; i < 1000; i++ {
				if goState.state.PC == endAddr {
					break
				}
				if exitGroup && goState.state.Exited {
					break
				}
				insn := state.Memory.GetMemory(state.PC)
				t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.PC, insn)

				stepWitness, err := goState.Step(true)
				require.NoError(t, err)
				evmPost := evm.Step(t, stepWitness)
				// verify the post-state matches.
				// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
				goPost := goState.state.EncodeWitness()
				require.Equalf(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM at step %d", state.Step)
			}
			if exitGroup {
				require.NotEqual(t, uint32(endAddr), goState.state.PC, "must not reach end")
				require.True(t, goState.state.Exited, "must set exited state")
				require.Equal(t, uint8(1), goState.state.ExitCode, "must exit with 1")
			} else {
				require.Equal(t, uint32(endAddr), state.PC, "must reach end")
				// inspect test result
				done, result := state.Memory.GetMemory(baseAddrEnd+4), state.Memory.GetMemory(baseAddrEnd+8)
				require.Equal(t, done, uint32(1), "must be done")
				require.Equal(t, result, uint32(1), "must have success result")
			}
		})
	}
}

func TestEVMSingleStep(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer vm.EVMLogger

	cases := []struct {
		name   string
		pc     uint32
		nextPC uint32
		insn   uint32
	}{
		{"j MSB set target", 0, 4, 0x0A_00_00_02},                         // j 0x02_00_00_02
		{"j non-zero PC region", 0x10000000, 0x10000004, 0x08_00_00_02},   // j 0x2
		{"jal MSB set target", 0, 4, 0x0E_00_00_02},                       // jal 0x02_00_00_02
		{"jal non-zero PC region", 0x10000000, 0x10000004, 0x0C_00_00_02}, // jal 0x2
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			state := &State{PC: tt.pc, NextPC: tt.nextPC, Memory: NewMemory()}
			state.Memory.SetMemory(tt.pc, tt.insn)

			us := NewInstrumentedState(state, nil, os.Stdout, os.Stderr)
			stepWitness, err := us.Step(true)
			require.NoError(t, err)

			evm := NewMIPSEVM(contracts, addrs)
			evm.SetTracer(tracer)
			evmPost := evm.Step(t, stepWitness)
			goPost := us.state.EncodeWitness()
			require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
				"mipsevm produced different state than EVM")
		})
	}
}

func TestEVMSysWriteHint(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer vm.EVMLogger

	cases := []struct {
		name          string
		memOffset     int      // Where the hint data is stored in memory
		hintData      []byte   // Hint data stored in memory at memOffset
		bytesToWrite  int      // How many bytes of hintData to write
		lastHint      []byte   // The buffer that stores lastHint in the state
		expectedHints [][]byte // The hints we expect to be processed
	}{
		{
			name:      "write 1 full hint at beginning of page",
			memOffset: 4096,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 10,
			lastHint:     nil,
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
			},
		},
		{
			name:      "write 1 full hint across page boundary",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 8, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 12,
			lastHint:     nil,
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB},
			},
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
			lastHint:     nil,
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB},
			},
		},
		{
			name:      "write a single partial hint",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite:  8,
			lastHint:      nil,
			expectedHints: nil,
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
			lastHint:     nil,
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
			},
		},
		{
			name:      "write a single partial hint to large capacity lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite:  8,
			lastHint:      make([]byte, 0, 4096),
			expectedHints: nil,
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
		},
		{
			name:      "write partial hint data to non-empty lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC, // Hint data
			},
			bytesToWrite:  4,
			lastHint:      []byte{0, 0, 0, 8},
			expectedHints: nil,
		},
	}

	const (
		insn = uint32(0x00_00_00_0C) // syscall instruction
	)

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			oracle := hintTrackingOracle{}
			state := &State{PC: 0, NextPC: 4, Memory: NewMemory()}

			state.LastHint = tt.lastHint
			state.Registers[2] = sysWrite
			state.Registers[4] = fdHintWrite
			state.Registers[5] = uint32(tt.memOffset)
			state.Registers[6] = uint32(tt.bytesToWrite)

			err := state.Memory.SetMemoryRange(uint32(tt.memOffset), bytes.NewReader(tt.hintData))
			require.NoError(t, err)
			state.Memory.SetMemory(0, insn)

			us := NewInstrumentedState(state, &oracle, os.Stdout, os.Stderr)
			stepWitness, err := us.Step(true)
			require.NoError(t, err)
			require.Equal(t, tt.expectedHints, oracle.hints)

			evm := NewMIPSEVM(contracts, addrs)
			evm.SetTracer(tracer)
			evmPost := evm.Step(t, stepWitness)
			goPost := us.state.EncodeWitness()
			require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
				"mipsevm produced different state than EVM")
		})
	}
}

func TestEVMFault(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer vm.EVMLogger // no-tracer by default, but see MarkdownTracer
	sender := common.Address{0x13, 0x37}

	env, evmState := NewEVMEnv(contracts, addrs)
	env.Config.Tracer = tracer

	cases := []struct {
		name   string
		nextPC uint32
		insn   uint32
	}{
		{"illegal instruction", 0, 0xFF_FF_FF_FF},
		{"branch in delay-slot", 8, 0x11_02_00_03},
		{"jump in delay-slot", 8, 0x0c_00_00_0c},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			state := &State{PC: 0, NextPC: tt.nextPC, Memory: NewMemory()}
			initialState := &State{PC: 0, NextPC: tt.nextPC, Memory: state.Memory}
			state.Memory.SetMemory(0, tt.insn)

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = endAddr

			us := NewInstrumentedState(state, nil, os.Stdout, os.Stderr)
			require.Panics(t, func() { _, _ = us.Step(true) })

			insnProof := initialState.Memory.MerkleProof(0)
			stepWitness := &StepWitness{
				State:    initialState.EncodeWitness(),
				MemProof: insnProof[:],
			}
			input := encodeStepInput(t, stepWitness, LocalContext{}, contracts.MIPS)
			startingGas := uint64(30_000_000)

			_, _, err := env.Call(vm.AccountRef(sender), addrs.MIPS, input, startingGas, common.U2560)
			require.EqualValues(t, err, vm.ErrExecutionReverted)
			logs := evmState.Logs()
			require.Equal(t, 0, len(logs))
		})
	}
}

func TestHelloEVM(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer vm.EVMLogger // no-tracer by default, but see MarkdownTracer

	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = PatchGo(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")
	require.NoError(t, PatchStack(state), "add initial stack")

	var stdOutBuf, stdErrBuf bytes.Buffer
	goState := NewInstrumentedState(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))

	start := time.Now()
	for i := 0; i < 400_000; i++ {
		if goState.state.Exited {
			break
		}
		insn := state.Memory.GetMemory(state.PC)
		if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
			t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.PC, insn)
		}

		evm := NewMIPSEVM(contracts, addrs)
		evm.SetTracer(tracer)

		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		evmPost := evm.Step(t, stepWitness)
		// verify the post-state matches.
		// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
		goPost := goState.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	}
	end := time.Now()
	delta := end.Sub(start)
	t.Logf("test took %s, %d instructions, %s per instruction", delta, state.Step, delta/time.Duration(state.Step))

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, "hello world!\n", stdOutBuf.String(), "stdout says hello")
	require.Equal(t, "", stdErrBuf.String(), "stderr silent")
}

func TestClaimEVM(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer vm.EVMLogger // no-tracer by default, but see MarkdownTracer

	elfProgram, err := elf.Open("../example/bin/claim.elf")
	require.NoError(t, err, "open ELF file")

	state, err := LoadELF(elfProgram)
	require.NoError(t, err, "load ELF into state")

	err = PatchGo(elfProgram, state)
	require.NoError(t, err, "apply Go runtime patches")
	require.NoError(t, PatchStack(state), "add initial stack")

	oracle, expectedStdOut, expectedStdErr := claimTestOracle(t)

	var stdOutBuf, stdErrBuf bytes.Buffer
	goState := NewInstrumentedState(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))

	for i := 0; i < 2000_000; i++ {
		if goState.state.Exited {
			break
		}

		insn := state.Memory.GetMemory(state.PC)
		if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
			t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.PC, insn)
		}

		stepWitness, err := goState.Step(true)
		require.NoError(t, err)

		evm := NewMIPSEVM(contracts, addrs)
		evm.SetTracer(tracer)
		evmPost := evm.Step(t, stepWitness)

		goPost := goState.state.EncodeWitness()
		require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			"mipsevm produced different state than EVM")
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
	require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
}

type hintTrackingOracle struct {
	hints [][]byte
}

func (t *hintTrackingOracle) Hint(v []byte) {
	t.hints = append(t.hints, v)
}

func (t *hintTrackingOracle) GetPreimage(k [32]byte) []byte {
	return nil
}
