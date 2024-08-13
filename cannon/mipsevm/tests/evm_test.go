package tests

import (
	"bytes"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func testContractsSetup(t require.TestingT) (*testutil.Artifacts, *testutil.Addresses) {
	artifacts, err := testutil.LoadArtifacts()
	require.NoError(t, err)

	addrs := &testutil.Addresses{
		MIPS:         common.Address{0: 0xff, 19: 1},
		Oracle:       common.Address{0: 0xff, 19: 2},
		Sender:       common.Address{0x13, 0x37},
		FeeRecipient: common.Address{0xaa},
	}

	return artifacts, addrs
}

func TestEVM(t *testing.T) {
	testFiles, err := os.ReadDir("open_mips_tests/test/bin")
	require.NoError(t, err)

	contracts, addrs := testContractsSetup(t)
	var tracer *tracing.Hooks // no-tracer by default, but test_util.MarkdownTracer

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			oracle := testutil.SelectOracleFixture(t, f.Name())
			// Short-circuit early for exit_group.bin
			exitGroup := f.Name() == "exit_group.bin"

			evm := testutil.NewMIPSEVM(contracts, addrs)
			evm.SetTracer(tracer)
			evm.SetLocalOracle(oracle)
			testutil.LogStepFailureAtCleanup(t, evm)

			fn := path.Join("open_mips_tests/test/bin", f.Name())
			programMem, err := os.ReadFile(fn)
			require.NoError(t, err)
			state := &singlethreaded.State{Cpu: mipsevm.CpuScalars{PC: 0, NextPC: 4}, Memory: memory.NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = testutil.EndAddr

			goState := singlethreaded.NewInstrumentedState(state, oracle, os.Stdout, os.Stderr, nil)

			for i := 0; i < 1000; i++ {
				curStep := goState.GetState().GetStep()
				if goState.GetState().GetPC() == testutil.EndAddr {
					break
				}
				if exitGroup && goState.GetState().GetExited() {
					break
				}
				insn := state.Memory.GetMemory(state.Cpu.PC)
				t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.Cpu.PC, insn)

				stepWitness, err := goState.Step(true)
				require.NoError(t, err)
				evmPost := evm.Step(t, stepWitness, curStep, singlethreaded.GetStateHashFn())
				// verify the post-state matches.
				// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
				goPost, _ := goState.GetState().EncodeWitness()
				require.Equalf(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM at step %d", state.Step)
			}
			if exitGroup {
				require.NotEqual(t, uint32(testutil.EndAddr), goState.GetState().GetPC(), "must not reach end")
				require.True(t, goState.GetState().GetExited(), "must set exited state")
				require.Equal(t, uint8(1), goState.GetState().GetExitCode(), "must exit with 1")
			} else {
				require.Equal(t, uint32(testutil.EndAddr), state.Cpu.PC, "must reach end")
				// inspect test result
				done, result := state.Memory.GetMemory(testutil.BaseAddrEnd+4), state.Memory.GetMemory(testutil.BaseAddrEnd+8)
				require.Equal(t, done, uint32(1), "must be done")
				require.Equal(t, result, uint32(1), "must have success result")
			}
		})
	}
}

func TestEVM_CloneFlags(t *testing.T) {
	//contracts, addrs := testContractsSetup(t)
	//var tracer *tracing.Hooks

	cases := []struct {
		name  string
		flags uint32
		valid bool
	}{
		{"the supported flags bitmask", exec.ValidCloneFlags, true},
		{"no flags", 0, false},
		{"all flags", ^uint32(0), false},
		{"all unsupported flags", ^uint32(exec.ValidCloneFlags), false},
		{"a few supported flags", exec.CloneFs | exec.CloneSysvsem, false},
		{"one supported flag", exec.CloneFs, false},
		{"mixed supported and unsupported flags", exec.CloneFs | exec.CloneParentSettid, false},
		{"a single unsupported flag", exec.CloneUntraced, false},
		{"multiple unsupported flags", exec.CloneUntraced | exec.CloneParentSettid, false},
	}

	const insn = uint32(0x00_00_00_0C) // syscall instruction
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			state := multithreaded.CreateEmptyState()
			state.Memory.SetMemory(state.GetPC(), insn)
			state.GetRegisters()[2] = exec.SysClone // Set syscall number
			state.GetRegisters()[4] = tt.flags      // Set first argument
			//curStep := state.Step

			us := multithreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			if !tt.valid {
				// The VM should exit
				_, err := us.Step(true)
				require.NoError(t, err)
				require.Equal(t, true, us.GetState().GetExited())
				require.Equal(t, uint8(mipsevm.VMStatusPanic), us.GetState().GetExitCode())
			} else {
				/*stepWitness*/ _, err := us.Step(true)
				require.NoError(t, err)
			}

			// TODO: Validate EVM execution once onchain implementation is ready
			//evm := testutil.NewMIPSEVM(contracts, addrs)
			//evm.SetTracer(tracer)
			//testutil.LogStepFailureAtCleanup(t, evm)
			//
			//evmPost := evm.Step(t, stepWitness, curStep, singlethreaded.GetStateHashFn())
			//goPost, _ := us.GetState().EncodeWitness()
			//require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
			//	"mipsevm produced different state than EVM")
		})
	}
}

func TestEVMSingleStep(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer *tracing.Hooks

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
			state := &singlethreaded.State{Cpu: mipsevm.CpuScalars{PC: tt.pc, NextPC: tt.nextPC}, Memory: memory.NewMemory()}
			state.Memory.SetMemory(tt.pc, tt.insn)
			curStep := state.Step

			us := singlethreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			stepWitness, err := us.Step(true)
			require.NoError(t, err)

			evm := testutil.NewMIPSEVM(contracts, addrs)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			evmPost := evm.Step(t, stepWitness, curStep, singlethreaded.GetStateHashFn())
			goPost, _ := us.GetState().EncodeWitness()
			require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
				"mipsevm produced different state than EVM")
		})
	}
}

func TestEVMSysWriteHint(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer *tracing.Hooks

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
			state := &singlethreaded.State{Cpu: mipsevm.CpuScalars{PC: 0, NextPC: 4}, Memory: memory.NewMemory()}

			state.LastHint = tt.lastHint
			state.Registers[2] = exec.SysWrite
			state.Registers[4] = exec.FdHintWrite
			state.Registers[5] = uint32(tt.memOffset)
			state.Registers[6] = uint32(tt.bytesToWrite)

			err := state.Memory.SetMemoryRange(uint32(tt.memOffset), bytes.NewReader(tt.hintData))
			require.NoError(t, err)
			state.Memory.SetMemory(0, insn)
			curStep := state.Step

			us := singlethreaded.NewInstrumentedState(state, &oracle, os.Stdout, os.Stderr, nil)
			stepWitness, err := us.Step(true)
			require.NoError(t, err)
			require.Equal(t, tt.expectedHints, oracle.hints)

			evm := testutil.NewMIPSEVM(contracts, addrs)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			evmPost := evm.Step(t, stepWitness, curStep, singlethreaded.GetStateHashFn())
			goPost, _ := us.GetState().EncodeWitness()
			require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
				"mipsevm produced different state than EVM")
		})
	}
}

func TestEVMFault(t *testing.T) {
	contracts, addrs := testContractsSetup(t)
	var tracer *tracing.Hooks // no-tracer by default, but see test_util.MarkdownTracer
	sender := common.Address{0x13, 0x37}

	env, evmState := testutil.NewEVMEnv(contracts, addrs)
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
			state := &singlethreaded.State{Cpu: mipsevm.CpuScalars{PC: 0, NextPC: tt.nextPC}, Memory: memory.NewMemory()}
			initialState := &singlethreaded.State{Cpu: mipsevm.CpuScalars{PC: 0, NextPC: tt.nextPC}, Memory: state.Memory}
			state.Memory.SetMemory(0, tt.insn)

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = testutil.EndAddr

			us := singlethreaded.NewInstrumentedState(state, nil, os.Stdout, os.Stderr, nil)
			require.Panics(t, func() { _, _ = us.Step(true) })

			insnProof := initialState.Memory.MerkleProof(0)
			encodedWitness, _ := initialState.EncodeWitness()
			stepWitness := &mipsevm.StepWitness{
				State:     encodedWitness,
				ProofData: insnProof[:],
			}
			input := testutil.EncodeStepInput(t, stepWitness, mipsevm.LocalContext{}, contracts.MIPS)
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
	var tracer *tracing.Hooks // no-tracer by default, but see test_util.MarkdownTracer
	evm := testutil.NewMIPSEVM(contracts, addrs)
	evm.SetTracer(tracer)
	testutil.LogStepFailureAtCleanup(t, evm)

	state := testutil.LoadELFProgram(t, "../../testdata/example/bin/hello.elf", singlethreaded.CreateInitialState, true)
	var stdOutBuf, stdErrBuf bytes.Buffer
	goState := singlethreaded.NewInstrumentedState(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), nil)

	start := time.Now()
	for i := 0; i < 400_000; i++ {
		curStep := goState.GetState().GetStep()
		if goState.GetState().GetExited() {
			break
		}
		insn := state.Memory.GetMemory(state.Cpu.PC)
		if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
			t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.Cpu.PC, insn)
		}

		stepWitness, err := goState.Step(true)
		require.NoError(t, err)
		evmPost := evm.Step(t, stepWitness, curStep, singlethreaded.GetStateHashFn())
		// verify the post-state matches.
		// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
		goPost, _ := goState.GetState().EncodeWitness()
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
	var tracer *tracing.Hooks // no-tracer by default, but see test_util.MarkdownTracer
	evm := testutil.NewMIPSEVM(contracts, addrs)
	evm.SetTracer(tracer)
	testutil.LogStepFailureAtCleanup(t, evm)

	state := testutil.LoadELFProgram(t, "../../testdata/example/bin/claim.elf", singlethreaded.CreateInitialState, true)
	oracle, expectedStdOut, expectedStdErr := testutil.ClaimTestOracle(t)

	var stdOutBuf, stdErrBuf bytes.Buffer
	goState := singlethreaded.NewInstrumentedState(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), nil)

	for i := 0; i < 2000_000; i++ {
		curStep := goState.GetState().GetStep()
		if goState.GetState().GetExited() {
			break
		}

		insn := state.Memory.GetMemory(state.Cpu.PC)
		if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
			t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.Step, state.Cpu.PC, insn)
		}

		stepWitness, err := goState.Step(true)
		require.NoError(t, err)

		evmPost := evm.Step(t, stepWitness, curStep, singlethreaded.GetStateHashFn())

		goPost, _ := goState.GetState().EncodeWitness()
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
