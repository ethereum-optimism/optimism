package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func TestEVM(t *testing.T) {
	testFiles, err := os.ReadDir("open_mips_tests/test/bin")
	require.NoError(t, err)

	var tracer *tracing.Hooks // no-tracer by default, but test_util.MarkdownTracer

	cases := GetMipsVersionTestCases(t)
	skippedTests := map[string][]string{
		"multi-threaded":  []string{"clone.bin"},
		"single-threaded": []string{},
	}

	for _, c := range cases {
		skipped, exists := skippedTests[c.Name]
		require.True(t, exists)
		for _, f := range testFiles {
			testName := fmt.Sprintf("%v (%v)", f.Name(), c.Name)
			t.Run(testName, func(t *testing.T) {
				for _, skipped := range skipped {
					if f.Name() == skipped {
						t.Skipf("Skipping explicitly excluded open_mips testcase: %v", f.Name())
					}
				}

				oracle := testutil.SelectOracleFixture(t, f.Name())
				// Short-circuit early for exit_group.bin
				exitGroup := f.Name() == "exit_group.bin"
				expectPanic := strings.HasSuffix(f.Name(), "panic.bin")

				evm := testutil.NewMIPSEVM(c.Contracts)
				evm.SetTracer(tracer)
				evm.SetLocalOracle(oracle)
				testutil.LogStepFailureAtCleanup(t, evm)

				fn := path.Join("open_mips_tests/test/bin", f.Name())
				programMem, err := os.ReadFile(fn)
				require.NoError(t, err)

				goVm := c.VMFactory(oracle, os.Stdout, os.Stderr, testutil.CreateLogger())
				state := goVm.GetState()
				err = state.GetMemory().SetMemoryRange(0, bytes.NewReader(programMem))
				require.NoError(t, err, "load program into state")

				// set the return address ($ra) to jump into when test completes
				state.GetRegistersRef()[31] = testutil.EndAddr

				// Catch panics and check if they are expected
				defer func() {
					if r := recover(); r != nil {
						if expectPanic {
							// Success
						} else {
							t.Errorf("unexpected panic: %v", r)
						}
					}
				}()

				for i := 0; i < 1000; i++ {
					curStep := goVm.GetState().GetStep()
					if goVm.GetState().GetPC() == testutil.EndAddr {
						break
					}
					if exitGroup && goVm.GetState().GetExited() {
						break
					}
					insn := state.GetMemory().GetMemory(state.GetPC())
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)

					stepWitness, err := goVm.Step(true)
					require.NoError(t, err)
					evmPost := evm.Step(t, stepWitness, curStep, c.StateHashFn)
					// verify the post-state matches.
					// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
					goPost, _ := goVm.GetState().EncodeWitness()
					require.Equalf(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
						"mipsevm produced different state than EVM at step %d", state.GetStep())
				}
				if exitGroup {
					require.NotEqual(t, uint32(testutil.EndAddr), goVm.GetState().GetPC(), "must not reach end")
					require.True(t, goVm.GetState().GetExited(), "must set exited state")
					require.Equal(t, uint8(1), goVm.GetState().GetExitCode(), "must exit with 1")
				} else if expectPanic {
					require.NotEqual(t, uint32(testutil.EndAddr), state.GetPC(), "must not reach end")
				} else {
					require.Equal(t, uint32(testutil.EndAddr), state.GetPC(), "must reach end")
					// inspect test result
					done, result := state.GetMemory().GetMemory(testutil.BaseAddrEnd+4), state.GetMemory().GetMemory(testutil.BaseAddrEnd+8)
					require.Equal(t, done, uint32(1), "must be done")
					require.Equal(t, result, uint32(1), "must have success result")
				}
			})
		}
	}
}

func TestEVMSingleStep_Jump(t *testing.T) {
	var tracer *tracing.Hooks

	versions := GetMipsVersionTestCases(t)
	cases := []struct {
		name         string
		pc           uint32
		nextPC       uint32
		insn         uint32
		expectNextPC uint32
		expectLink   bool
	}{
		{name: "j MSB set target", pc: 0, nextPC: 4, insn: 0x0A_00_00_02, expectNextPC: 0x08_00_00_08},                                           // j 0x02_00_00_02
		{name: "j non-zero PC region", pc: 0x10000000, nextPC: 0x10000004, insn: 0x08_00_00_02, expectNextPC: 0x10_00_00_08},                     // j 0x2
		{name: "jal MSB set target", pc: 0, nextPC: 4, insn: 0x0E_00_00_02, expectNextPC: 0x08_00_00_08, expectLink: true},                       // jal 0x02_00_00_02
		{name: "jal non-zero PC region", pc: 0x10000000, nextPC: 0x10000004, insn: 0x0C_00_00_02, expectNextPC: 0x10_00_00_08, expectLink: true}, // jal 0x2
	}

	for _, v := range versions {
		for i, tt := range cases {
			testName := fmt.Sprintf("%v (%v)", tt.name, v.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)), testutil.WithPC(tt.pc), testutil.WithNextPC(tt.nextPC))
				state := goVm.GetState()
				state.GetMemory().SetMemory(tt.pc, tt.insn)
				step := state.GetStep()

				// Setup expectations
				expected := testutil.NewExpectedState(state)
				expected.Step += 1
				expected.PC = state.GetCpu().NextPC
				expected.NextPC = tt.expectNextPC
				if tt.expectLink {
					expected.Registers[31] = state.GetPC() + 8
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Check expectations
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts, tracer)
			})
		}
	}
}

func TestEVMSingleStep_Add(t *testing.T) {
	var tracer *tracing.Hooks

	versions := GetMipsVersionTestCases(t)
	cases := []struct {
		name      string
		insn      uint32
		ifImm     bool
		rs        uint32
		rt        uint32
		imm       uint16
		expectRD  uint32
		expectImm uint32
	}{
		{name: "add", insn: 0x02_32_40_20, ifImm: false, rs: uint32(12), rt: uint32(20), expectRD: uint32(32)},                         // add t0, s1, s2
		{name: "addu", insn: 0x02_32_40_21, ifImm: false, rs: uint32(12), rt: uint32(20), expectRD: uint32(32)},                        // addu t0, s1, s2
		{name: "addi", insn: 0x22_28_00_28, ifImm: true, rs: uint32(4), rt: uint32(1), imm: uint16(40), expectImm: uint32(44)},         // addi t0, s1, 40
		{name: "addi sign", insn: 0x22_28_ff_fe, ifImm: true, rs: uint32(2), rt: uint32(1), imm: uint16(0xfffe), expectImm: uint32(0)}, // addi t0, s1, -2
		{name: "addiu", insn: 0x26_28_00_28, ifImm: true, rs: uint32(4), rt: uint32(1), imm: uint16(40), expectImm: uint32(44)},        // addiu t0, s1, 40
	}

	for _, v := range versions {
		for i, tt := range cases {
			testName := fmt.Sprintf("%v (%v)", tt.name, v.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)), testutil.WithPC(0), testutil.WithNextPC(4))
				state := goVm.GetState()
				if tt.ifImm {
					state.GetRegistersRef()[8] = tt.rt
					state.GetRegistersRef()[17] = tt.rs
				} else {
					state.GetRegistersRef()[17] = tt.rs
					state.GetRegistersRef()[18] = tt.rt
				}
				state.GetMemory().SetMemory(0, tt.insn)
				step := state.GetStep()

				// Setup expectations
				expected := testutil.NewExpectedState(state)
				expected.Step += 1
				expected.PC = 4
				expected.NextPC = 8

				if tt.ifImm {
					expected.Registers[8] = tt.expectImm
					expected.Registers[17] = tt.rs
				} else {
					expected.Registers[8] = tt.expectRD
					expected.Registers[17] = tt.rs
					expected.Registers[18] = tt.rt
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Check expectations
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts, tracer)
			})
		}
	}
}

func TestEVM_MMap(t *testing.T) {
	var tracer *tracing.Hooks

	versions := GetMipsVersionTestCases(t)
	cases := []struct {
		name         string
		heap         uint32
		address      uint32
		size         uint32
		shouldFail   bool
		expectedHeap uint32
	}{
		{name: "Increment heap by max value", heap: program.HEAP_START, address: 0, size: ^uint32(0), shouldFail: true},
		{name: "Increment heap to 0", heap: program.HEAP_START, address: 0, size: ^uint32(0) - program.HEAP_START + 1, shouldFail: true},
		{name: "Increment heap to previous page", heap: program.HEAP_START, address: 0, size: ^uint32(0) - program.HEAP_START - memory.PageSize + 1, shouldFail: true},
		{name: "Increment max page size", heap: program.HEAP_START, address: 0, size: ^uint32(0) & ^uint32(memory.PageAddrMask), shouldFail: true},
		{name: "Increment max page size from 0", heap: 0, address: 0, size: ^uint32(0) & ^uint32(memory.PageAddrMask), shouldFail: true},
		{name: "Increment heap at limit", heap: program.HEAP_END, address: 0, size: 1, shouldFail: true},
		{name: "Increment heap to limit", heap: program.HEAP_END - memory.PageSize, address: 0, size: 1, shouldFail: false, expectedHeap: program.HEAP_END},
		{name: "Increment heap within limit", heap: program.HEAP_END - 2*memory.PageSize, address: 0, size: 1, shouldFail: false, expectedHeap: program.HEAP_END - memory.PageSize},
		{name: "Request specific address", heap: program.HEAP_START, address: 0x50_00_00_00, size: 0, shouldFail: false, expectedHeap: program.HEAP_START},
	}

	for _, v := range versions {
		for i, c := range cases {
			testName := fmt.Sprintf("%v (%v)", c.name, v.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)), testutil.WithHeap(c.heap))
				state := goVm.GetState()

				state.GetMemory().SetMemory(state.GetPC(), syscallInsn)
				state.GetRegistersRef()[2] = exec.SysMmap
				state.GetRegistersRef()[4] = c.address
				state.GetRegistersRef()[5] = c.size
				step := state.GetStep()

				expected := testutil.NewExpectedState(state)
				expected.Step += 1
				expected.PC = state.GetCpu().NextPC
				expected.NextPC = state.GetCpu().NextPC + 4
				if c.shouldFail {
					expected.Registers[2] = exec.SysErrorSignal
					expected.Registers[7] = exec.MipsEINVAL
				} else {
					expected.Heap = c.expectedHeap
					if c.address == 0 {
						expected.Registers[2] = state.GetHeap()
						expected.Registers[7] = 0
					} else {
						expected.Registers[2] = c.address
						expected.Registers[7] = 0
					}
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				// Check expectations
				expected.Validate(t, state)
				testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts, tracer)
			})
		}
	}
}

func TestEVMSysWriteHint(t *testing.T) {
	var tracer *tracing.Hooks

	versions := GetMipsVersionTestCases(t)
	cases := []struct {
		name             string
		memOffset        int      // Where the hint data is stored in memory
		hintData         []byte   // Hint data stored in memory at memOffset
		bytesToWrite     int      // How many bytes of hintData to write
		lastHint         []byte   // The buffer that stores lastHint in the state
		expectedHints    [][]byte // The hints we expect to be processed
		expectedLastHint []byte   // The lastHint we should expect for the post-state
	}{
		{
			name:      "write 1 full hint at beginning of page",
			memOffset: 4096,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 10,
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write 1 full hint across page boundary",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 8, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite: 12,
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
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
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{},
		},
		{
			name:      "write a single partial hint",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite:     8,
			lastHint:         []byte{},
			expectedHints:    nil,
			expectedLastHint: []byte{0, 0, 0, 6, 0xAA, 0xAA, 0xAA, 0xAA},
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
			lastHint:     []byte{},
			expectedHints: [][]byte{
				{0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB},
			},
			expectedLastHint: []byte{0, 0, 0, 8, 0xAA, 0xAA},
		},
		{
			name:      "write a single partial hint to large capacity lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0, 0, 0, 6, // Length prefix
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, // Hint data
			},
			bytesToWrite:     8,
			lastHint:         make([]byte, 0, 4096),
			expectedHints:    nil,
			expectedLastHint: []byte{0, 0, 0, 6, 0xAA, 0xAA, 0xAA, 0xAA},
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
			expectedLastHint: []byte{},
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
			expectedLastHint: []byte{},
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
			expectedLastHint: []byte{},
		},
		{
			name:      "write partial hint data to non-empty lastHint buffer",
			memOffset: 4092,
			hintData: []byte{
				0xAA, 0xAA, 0xAA, 0xAA, 0xBB, 0xBB, 0xCC, 0xCC, // Hint data
			},
			bytesToWrite:     4,
			lastHint:         []byte{0, 0, 0, 8},
			expectedHints:    nil,
			expectedLastHint: []byte{0, 0, 0, 8, 0xAA, 0xAA, 0xAA, 0xAA},
		},
	}

	const (
		insn = uint32(0x00_00_00_0C) // syscall instruction
	)

	for _, v := range versions {
		for i, tt := range cases {
			testName := fmt.Sprintf("%v (%v)", tt.name, v.Name)
			t.Run(testName, func(t *testing.T) {
				oracle := testutil.HintTrackingOracle{}
				goVm := v.VMFactory(&oracle, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithRandomization(int64(i)), testutil.WithLastHint(tt.lastHint))
				state := goVm.GetState()
				state.GetRegistersRef()[2] = exec.SysWrite
				state.GetRegistersRef()[4] = exec.FdHintWrite
				state.GetRegistersRef()[5] = uint32(tt.memOffset)
				state.GetRegistersRef()[6] = uint32(tt.bytesToWrite)

				err := state.GetMemory().SetMemoryRange(uint32(tt.memOffset), bytes.NewReader(tt.hintData))
				require.NoError(t, err)
				state.GetMemory().SetMemory(state.GetPC(), insn)
				step := state.GetStep()

				expected := testutil.NewExpectedState(state)
				expected.Step += 1
				expected.PC = state.GetCpu().NextPC
				expected.NextPC = state.GetCpu().NextPC + 4
				expected.LastHint = tt.expectedLastHint
				expected.Registers[2] = uint32(tt.bytesToWrite) // Return count of bytes written
				expected.Registers[7] = 0                       // no Error

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				expected.Validate(t, state)
				require.Equal(t, tt.expectedHints, oracle.Hints())
				testutil.ValidateEVM(t, stepWitness, step, goVm, v.StateHashFn, v.Contracts, tracer)
			})
		}
	}
}

func TestEVMFault(t *testing.T) {
	var tracer *tracing.Hooks // no-tracer by default, but see test_util.MarkdownTracer

	versions := GetMipsVersionTestCases(t)
	cases := []struct {
		name   string
		nextPC uint32
		insn   uint32
	}{
		{"illegal instruction", 0, 0xFF_FF_FF_FF},
		{"branch in delay-slot", 8, 0x11_02_00_03},
		{"jump in delay-slot", 8, 0x0c_00_00_0c},
	}

	for _, v := range versions {
		for _, tt := range cases {
			testName := fmt.Sprintf("%v (%v)", tt.name, v.Name)
			t.Run(testName, func(t *testing.T) {
				goVm := v.VMFactory(nil, os.Stdout, os.Stderr, testutil.CreateLogger(), testutil.WithNextPC(tt.nextPC))
				state := goVm.GetState()
				state.GetMemory().SetMemory(0, tt.insn)
				// set the return address ($ra) to jump into when test completes
				state.GetRegistersRef()[31] = testutil.EndAddr

				require.Panics(t, func() { _, _ = goVm.Step(true) })
				testutil.AssertEVMReverts(t, state, v.Contracts, tracer)
			})
		}
	}
}

func TestHelloEVM(t *testing.T) {
	t.Parallel()
	var tracer *tracing.Hooks // no-tracer by default, but see test_util.MarkdownTracer
	versions := GetMipsVersionTestCases(t)

	for _, v := range versions {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			evm := testutil.NewMIPSEVM(v.Contracts)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := "../../testdata/example/bin/hello.elf"
			goVm := v.ElfVMFactory(t, elfFile, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			start := time.Now()
			for i := 0; i < 400_000; i++ {
				step := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}
				insn := state.GetMemory().GetMemory(state.GetPC())
				if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				evmPost := evm.Step(t, stepWitness, step, v.StateHashFn)
				// verify the post-state matches.
				// TODO: maybe more readable to decode the evmPost state, and do attribute-wise comparison.
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equalf(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM. insn: %x", insn)
			}
			end := time.Now()
			delta := end.Sub(start)
			t.Logf("test took %s, %d instructions, %s per instruction", delta, state.GetStep(), delta/time.Duration(state.GetStep()))

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

			require.Equal(t, "hello world!\n", stdOutBuf.String(), "stdout says hello")
			require.Equal(t, "", stdErrBuf.String(), "stderr silent")
		})
	}
}

func TestClaimEVM(t *testing.T) {
	t.Parallel()
	var tracer *tracing.Hooks // no-tracer by default, but see test_util.MarkdownTracer
	versions := GetMipsVersionTestCases(t)

	for _, v := range versions {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			evm := testutil.NewMIPSEVM(v.Contracts)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			oracle, expectedStdOut, expectedStdErr := testutil.ClaimTestOracle(t)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := "../../testdata/example/bin/claim.elf"
			goVm := v.ElfVMFactory(t, elfFile, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			for i := 0; i < 2000_000; i++ {
				curStep := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}

				insn := state.GetMemory().GetMemory(state.GetPC())
				if i%1000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)

				evmPost := evm.Step(t, stepWitness, curStep, v.StateHashFn)

				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			}

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")

			require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
			require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
		})
	}
}

func TestEntryEVM(t *testing.T) {
	t.Parallel()
	var tracer *tracing.Hooks // no-tracer by default, but see test_util.MarkdownTracer
	versions := GetMipsVersionTestCases(t)

	for _, v := range versions {
		v := v
		t.Run(v.Name, func(t *testing.T) {
			t.Parallel()
			evm := testutil.NewMIPSEVM(v.Contracts)
			evm.SetTracer(tracer)
			testutil.LogStepFailureAtCleanup(t, evm)

			var stdOutBuf, stdErrBuf bytes.Buffer
			elfFile := "../../testdata/example/bin/entry.elf"
			goVm := v.ElfVMFactory(t, elfFile, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr), testutil.CreateLogger())
			state := goVm.GetState()

			start := time.Now()
			for i := 0; i < 400_000; i++ {
				curStep := goVm.GetState().GetStep()
				if goVm.GetState().GetExited() {
					break
				}
				insn := state.GetMemory().GetMemory(state.GetPC())
				if i%10_000 == 0 { // avoid spamming test logs, we are executing many steps
					t.Logf("step: %4d pc: 0x%08x insn: 0x%08x", state.GetStep(), state.GetPC(), insn)
				}

				stepWitness, err := goVm.Step(true)
				require.NoError(t, err)
				evmPost := evm.Step(t, stepWitness, curStep, v.StateHashFn)
				// verify the post-state matches.
				goPost, _ := goVm.GetState().EncodeWitness()
				require.Equal(t, hexutil.Bytes(goPost).String(), hexutil.Bytes(evmPost).String(),
					"mipsevm produced different state than EVM")
			}
			end := time.Now()
			delta := end.Sub(start)
			t.Logf("test took %s, %d instructions, %s per instruction", delta, state.GetStep(), delta/time.Duration(state.GetStep()))

			require.True(t, state.GetExited(), "must complete program")
			require.Equal(t, uint8(0), state.GetExitCode(), "exit with 0")
		})
	}
}
