package tests

import (
	"bytes"
	"math"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	mtutil "github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded/testutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
)

const syscallInsn = uint32(0x00_00_00_0c)

func FuzzStateSyscallBrk(f *testing.F) {
	vms := GetMipsVersionTestCases(f)

	initState := func(t require.TestingT, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.GetRegistersRef()[2] = arch.SysBrk
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
	}

	setExpectations := func(t require.TestingT, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = program.PROGRAM_BREAK // Return fixed BRK value
		expected.ActiveThread().Registers[7] = 0                     // No error
		return ExpectNormalExecution()
	}

	diffTester := NewSimpleDiffTester().
		InitState(initState).
		SetExpectations(setExpectations)

	f.Fuzz(func(t *testing.T, seed int64) {
		diffTester.Run(t, fuzzTestOptions(vms, seed)...)
	})
}

func FuzzStateSyscallMmap(f *testing.F) {
	// Add special cases for large memory allocation
	f.Add(Word(0), Word(0x1000), Word(program.HEAP_END), int64(1))
	f.Add(Word(0), Word(1<<31), Word(program.HEAP_START), int64(2))
	// Check edge case - just within bounds
	f.Add(Word(0), Word(0x1000), Word(program.HEAP_END-4096), int64(3))

	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		addr Word
		siz  Word
		heap Word
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.Heap = c.heap
		state.GetRegistersRef()[2] = arch.SysMmap
		state.GetRegistersRef()[4] = c.addr
		state.GetRegistersRef()[5] = c.siz
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		if c.addr == 0 {
			sizAlign := c.siz
			if sizAlign&memory.PageAddrMask != 0 { // adjust size to align with page size
				sizAlign = c.siz + memory.PageSize - (c.siz & memory.PageAddrMask)
			}
			newHeap := c.heap + sizAlign
			if newHeap > program.HEAP_END || newHeap < c.heap || sizAlign < c.siz {
				expected.ActiveThread().Registers[2] = exec.MipsEINVAL
				expected.ActiveThread().Registers[7] = exec.SysErrorSignal
			} else {
				expected.Heap = c.heap + sizAlign
				expected.ActiveThread().Registers[2] = c.heap
				expected.ActiveThread().Registers[7] = 0 // no error
			}
		} else {
			expected.ActiveThread().Registers[2] = c.addr
			expected.ActiveThread().Registers[7] = 0 // no error
		}
		return ExpectNormalExecution()
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState).
		SetExpectations(setExpectations)

	f.Fuzz(func(t *testing.T, addr Word, siz Word, heap Word, seed int64) {
		tests := []testCase{{addr, siz, heap}}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed)...)
	})
}

func FuzzStateSyscallExitGroup(f *testing.F) {
	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		exitCode uint8
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.GetRegistersRef()[2] = arch.SysExitGroup
		state.GetRegistersRef()[4] = Word(c.exitCode)
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.Step += 1
		expected.ExpectNoContextSwitch()
		expected.Exited = true
		expected.ExitCode = c.exitCode
		return ExpectNormalExecution()
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState).
		SetExpectations(setExpectations)

	f.Fuzz(func(t *testing.T, exitCode uint8, seed int64) {
		tests := []testCase{{exitCode}}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed)...)
	})
}

func FuzzStateSyscallFcntl(f *testing.F) {
	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		fd  Word
		cmd Word
	}

	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.GetRegistersRef()[2] = arch.SysFcntl
		state.GetRegistersRef()[4] = c.fd
		state.GetRegistersRef()[5] = c.cmd
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		if c.cmd == 1 {
			switch c.fd {
			case exec.FdStdin, exec.FdStdout, exec.FdStderr,
				exec.FdPreimageRead, exec.FdHintRead, exec.FdPreimageWrite, exec.FdHintWrite:
				expected.ActiveThread().Registers[2] = 0
				expected.ActiveThread().Registers[7] = 0
			default:
				expected.ActiveThread().Registers[2] = exec.MipsEBADF
				expected.ActiveThread().Registers[7] = exec.SysErrorSignal
			}
		} else if c.cmd == 3 {
			switch c.fd {
			case exec.FdStdin, exec.FdPreimageRead, exec.FdHintRead:
				expected.ActiveThread().Registers[2] = 0
				expected.ActiveThread().Registers[7] = 0
			case exec.FdStdout, exec.FdStderr, exec.FdPreimageWrite, exec.FdHintWrite:
				expected.ActiveThread().Registers[2] = 1
				expected.ActiveThread().Registers[7] = 0
			default:
				expected.ActiveThread().Registers[2] = exec.MipsEBADF
				expected.ActiveThread().Registers[7] = exec.SysErrorSignal
			}
		} else {
			expected.ActiveThread().Registers[2] = exec.MipsEINVAL
			expected.ActiveThread().Registers[7] = exec.SysErrorSignal
		}
		return ExpectNormalExecution()
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState).
		SetExpectations(setExpectations)

	f.Fuzz(func(t *testing.T, fd Word, cmd Word, seed int64) {
		tests := []testCase{{fd, cmd}}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed)...)
	})
}

func FuzzStateHintRead(f *testing.F) {
	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		addr  Word
		count Word
	}

	preimageData := []byte("hello world")
	preimageKey := preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey()
	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.PreimageKey = preimageKey
		state.GetRegistersRef()[2] = arch.SysRead
		state.GetRegistersRef()[4] = exec.FdHintRead
		state.GetRegistersRef()[5] = c.addr
		state.GetRegistersRef()[6] = c.count
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = c.count
		expected.ActiveThread().Registers[7] = 0 // no error
		return ExpectNormalExecution()
	}

	postCheck := func(t require.TestingT, c testCase, vm VersionedVMTestCase, deps *TestDependencies, stepWitness *mipsevm.StepWitness) {
		require.False(t, stepWitness.HasPreimage())
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState).
		SetExpectations(setExpectations).
		PostCheck(postCheck)

	f.Fuzz(func(t *testing.T, addr Word, count Word, seed int64) {
		tests := []testCase{{addr, count}}
		po := func() mipsevm.PreimageOracle {
			return testutil.StaticOracle(t, preimageData)
		}

		diffTester.Run(t, tests, fuzzTestOptions(vms, seed, WithPreimageOracle(po))...)
	})
}

func FuzzStatePreimageRead(f *testing.F) {
	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		addr           arch.Word
		pc             arch.Word
		count          arch.Word
		preimageOffset arch.Word
	}

	preexistingMemoryVal := ^arch.Word(0)
	preimageValue := []byte("hello world")
	preimageData := mtutil.AddPreimageLengthPrefix(preimageValue)
	preimageKey := preimage.Keccak256Key(crypto.Keccak256Hash(preimageValue)).PreimageKey()
	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.PreimageKey = preimageKey
		state.PreimageOffset = c.preimageOffset
		state.GetCurrentThread().Cpu.PC = c.pc
		state.GetCurrentThread().Cpu.NextPC = c.pc + 4
		state.GetRegistersRef()[2] = arch.SysRead
		state.GetRegistersRef()[4] = exec.FdPreimageRead
		state.GetRegistersRef()[5] = c.addr
		state.GetRegistersRef()[6] = c.count
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetMemory().SetWord(testutil.EffAddr(c.addr), preexistingMemoryVal)
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		alignment := c.addr & arch.ExtMask
		writeLen := arch.WordSizeBytes - alignment
		if c.count < writeLen {
			writeLen = c.count
		}
		// Cap write length to remaining bytes of the preimage
		preimageDataLen := Word(len(preimageData))
		if c.preimageOffset+writeLen > preimageDataLen {
			writeLen = preimageDataLen - c.preimageOffset
		}

		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = writeLen
		expected.ActiveThread().Registers[7] = 0 // no error
		expected.PreimageOffset += writeLen
		if writeLen > 0 {
			// Expect a memory write
			var expectedMemory []byte
			expectedMemory = arch.ByteOrderWord.AppendWord(expectedMemory, preexistingMemoryVal)
			copy(expectedMemory[alignment:], preimageData[c.preimageOffset:c.preimageOffset+writeLen])
			expected.ExpectMemoryWrite(testutil.EffAddr(c.addr), arch.ByteOrderWord.Word(expectedMemory[:]))
		}
		return ExpectNormalExecution()
	}

	postCheck := func(t require.TestingT, c testCase, vm VersionedVMTestCase, deps *TestDependencies, stepWitness *mipsevm.StepWitness) {
		require.True(t, stepWitness.HasPreimage())
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState).
		SetExpectations(setExpectations).
		PostCheck(postCheck)

	f.Fuzz(func(t *testing.T, addr arch.Word, pc arch.Word, count arch.Word, preimageOffset arch.Word, seed int64) {
		pc = testutil.EffAddr(pc)
		if preimageOffset >= Word(len(preimageData)) || pc == testutil.EffAddr(addr) {
			t.SkipNow()
		}
		po := func() mipsevm.PreimageOracle {
			return testutil.StaticOracle(t, preimageValue)
		}

		tests := []testCase{{addr, pc, count, preimageOffset}}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed, WithPreimageOracle(po))...)
	})
}

func FuzzStateHintWrite(f *testing.F) {
	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		// Fuzz inputs
		addr  Word
		count Word
		hint1 []byte
		hint2 []byte
		hint3 []byte
		// Cached calculations
		hintData         []byte
		lastHint         []byte
		expectedHints    [][]byte
		expectedLastHint []byte
	}

	cacheHintCalculations := func(t require.TestingT, c *testCase) {
		if c.hintData != nil {
			// Already cached
			return
		}

		// Set up hint data
		r := testutil.NewRandHelper(seed)
		hints := [][]byte{c.hint1, c.hint2, c.hint3}
		c.hintData = make([]byte, 0)
		for _, hint := range hints {
			prefixedHint := mtutil.AddHintLengthPrefix(hint)
			c.hintData = append(c.hintData, prefixedHint...)
		}
		lastHintLen := math.Round(r.Fraction() * float64(len(c.hintData)))
		c.lastHint = c.hintData[:int(lastHintLen)]
		expectedBytesToProcess := int(c.count) + int(lastHintLen)
		if expectedBytesToProcess > len(c.hintData) {
			// Add an extra hint to span the rest of the hint data
			randomHint := r.RandomBytes(t, expectedBytesToProcess)
			prefixedHint := mtutil.AddHintLengthPrefix(randomHint)
			c.hintData = append(c.hintData, prefixedHint...)
			hints = append(hints, randomHint)
		}

		// Figure out hint expectations
		c.expectedLastHint = make([]byte, 0)
		byteIndex := 0
		for _, hint := range hints {
			hintDataLength := len(hint) + 4 // Hint data + prefix
			hintLastByteIndex := hintDataLength + byteIndex - 1
			if hintLastByteIndex < expectedBytesToProcess {
				c.expectedHints = append(c.expectedHints, hint)
			} else {
				c.expectedLastHint = c.hintData[byteIndex:expectedBytesToProcess]
				break
			}
			byteIndex += hintDataLength
		}
	}

	initState := func(t require.TestingT, c *testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		cacheHintCalculations(t, c)
		state.LastHint = c.lastHint
		state.GetRegistersRef()[2] = arch.SysWrite
		state.GetRegistersRef()[4] = exec.FdHintWrite
		state.GetRegistersRef()[5] = c.addr
		state.GetRegistersRef()[6] = c.count
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		err := state.GetMemory().SetMemoryRange(c.addr, bytes.NewReader(c.hintData[int(len(c.lastHint)):]))
		require.NoError(t, err)
	}

	setExpectations := func(t require.TestingT, c *testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		cacheHintCalculations(t, c)
		expected.ExpectStep()
		expected.ActiveThread().Registers[2] = c.count
		expected.ActiveThread().Registers[7] = 0 // no error
		expected.LastHint = c.expectedLastHint
		return ExpectNormalExecution()
	}

	postCheck := func(t require.TestingT, c *testCase, vm VersionedVMTestCase, deps *TestDependencies, stepWitness *mipsevm.StepWitness) {
		oracle, ok := deps.po.(*testutil.HintTrackingOracle)
		require.True(t, ok)
		require.Equal(t, c.expectedHints, oracle.Hints())
	}

	diffTester := NewDiffTester(NoopTestNamer[*testCase]).
		InitState(initState, mtutil.WithPCAndNextPC(0)).
		SetExpectations(setExpectations).
		PostCheck(postCheck)

	f.Fuzz(func(t *testing.T, addr Word, count Word, hint1, hint2, hint3 []byte, seed int64) {
		// Make sure pc does not overlap with hint data in memory
		if addr <= 8 {
			addr += 8
		}

		po := func() mipsevm.PreimageOracle {
			return &testutil.HintTrackingOracle{}
		}

		tests := []*testCase{
			{
				addr:  addr,
				count: count,
				hint1: hint1,
				hint2: hint2,
				hint3: hint3,
			},
		}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed, WithPreimageOracle(po))...)
	})
}

func FuzzStatePreimageWrite(f *testing.F) {
	vms := GetMipsVersionTestCases(f)
	type testCase struct {
		addr  arch.Word
		count arch.Word
	}

	preexistingMemoryVal := [8]byte{0x12, 0x34, 0x56, 0x78, 0x87, 0x65, 0x43, 0x21}
	preimageData := []byte("hello world")
	preimageKey := preimage.Keccak256Key(crypto.Keccak256Hash(preimageData)).PreimageKey()
	initState := func(t require.TestingT, c testCase, state *multithreaded.State, vm VersionedVMTestCase, r *testutil.RandHelper, goVm mipsevm.FPVM) {
		state.GetRegistersRef()[2] = arch.SysWrite
		state.GetRegistersRef()[4] = exec.FdPreimageWrite
		state.GetRegistersRef()[5] = c.addr
		state.GetRegistersRef()[6] = c.count
		storeInsnWithCache(state, goVm, state.GetPC(), syscallInsn)
		state.GetMemory().SetWord(testutil.EffAddr(c.addr), arch.ByteOrderWord.Word(preexistingMemoryVal[:]))
	}

	setExpectations := func(t require.TestingT, c testCase, expected *mtutil.ExpectedState, vm VersionedVMTestCase) ExpectedExecResult {
		expectBytesWritten := c.count
		alignment := c.addr & arch.ExtMask
		sz := arch.WordSizeBytes - alignment
		if sz < expectBytesWritten {
			expectBytesWritten = sz
		}

		expected.ExpectStep()
		expected.PreimageOffset = 0
		expected.ActiveThread().Registers[2] = expectBytesWritten
		expected.ActiveThread().Registers[7] = 0 // No error
		expected.PreimageKey = preimageKey
		if expectBytesWritten > 0 {
			// Copy original preimage key, but shift it left by expectBytesWritten
			copy(expected.PreimageKey[:], preimageKey[expectBytesWritten:])
			// Copy memory data to rightmost expectedBytesWritten
			copy(expected.PreimageKey[32-expectBytesWritten:], preexistingMemoryVal[alignment:])
		}
		return ExpectNormalExecution()
	}

	postCheck := func(t require.TestingT, c testCase, vm VersionedVMTestCase, deps *TestDependencies, stepWitness *mipsevm.StepWitness) {
		require.False(t, stepWitness.HasPreimage())
	}

	diffTester := NewDiffTester(NoopTestNamer[testCase]).
		InitState(initState, mtutil.WithPCAndNextPC(0), mtutil.WithPreimageKey(preimageKey), mtutil.WithPreimageOffset(128)).
		SetExpectations(setExpectations).
		PostCheck(postCheck)

	f.Fuzz(func(t *testing.T, addr arch.Word, count arch.Word, seed int64) {
		if addr <= 8 {
			addr += 8
		}

		po := func() mipsevm.PreimageOracle {
			return testutil.StaticOracle(t, preimageData)
		}

		tests := []testCase{{addr, count}}
		diffTester.Run(t, tests, fuzzTestOptions(vms, seed, WithPreimageOracle(po))...)
	})
}

func fuzzTestOptions(vms []VersionedVMTestCase, seed int64, opts ...TestOption) []TestOption {
	testOpts := []TestOption{
		WithVms(vms),
		WithRandomSeed(seed),
		SkipAutomaticMemoryReservationTests(),
	}
	testOpts = append(testOpts, opts...)
	return testOpts
}
