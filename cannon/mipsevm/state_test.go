package mipsevm

import (
	"bytes"
	"debug/elf"
	"io"
	"os"
	"path"
	"testing"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/core"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/patch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/test_util"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func TestState(t *testing.T) {
	testFiles, err := os.ReadDir("tests/open_mips_tests/test/bin")
	require.NoError(t, err)

	for _, f := range testFiles {
		t.Run(f.Name(), func(t *testing.T) {
			oracle := test_util.SelectOracleFixture(t, f.Name())
			// Short-circuit early for exit_group.bin
			exitGroup := f.Name() == "exit_group.bin"

			// TODO: currently tests are compiled as flat binary objects
			// We can use more standard tooling to compile them to ELF files and get remove maketests.py
			fn := path.Join("tests/open_mips_tests/test/bin", f.Name())
			//elfProgram, err := elf.Open()
			//require.NoError(t, err, "must load test ELF binary")
			//state, err := LoadELF(elfProgram)
			//require.NoError(t, err, "must load ELF into state")
			programMem, err := os.ReadFile(fn)
			require.NoError(t, err)
			state := &State{Cpu: core.CpuScalars{PC: 0, NextPC: 4}, Memory: core.NewMemory()}
			err = state.Memory.SetMemoryRange(0, bytes.NewReader(programMem))
			require.NoError(t, err, "load program into state")

			// set the return address ($ra) to jump into when test completes
			state.Registers[31] = test_util.EndAddr

			us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)

			for i := 0; i < 1000; i++ {
				if us.state.Cpu.PC == test_util.EndAddr {
					break
				}
				if exitGroup && us.state.Exited {
					break
				}
				_, err := us.Step(false)
				require.NoError(t, err)
			}

			if exitGroup {
				require.NotEqual(t, uint32(test_util.EndAddr), us.state.Cpu.PC, "must not reach end")
				require.True(t, us.state.Exited, "must set exited state")
				require.Equal(t, uint8(1), us.state.ExitCode, "must exit with 1")
			} else {
				require.Equal(t, uint32(test_util.EndAddr), us.state.Cpu.PC, "must reach end")
				done, result := state.Memory.GetMemory(test_util.BaseAddrEnd+4), state.Memory.GetMemory(test_util.BaseAddrEnd+8)
				// inspect test result
				require.Equal(t, done, uint32(1), "must be done")
				require.Equal(t, result, uint32(1), "must have success result")
			}
		})
	}
}

// Run through all permutations of `exited` / `exitCode` and ensure that the
// correct witness, state hash, and VM Status is produced.
func TestStateHash(t *testing.T) {
	cases := []struct {
		exited   bool
		exitCode uint8
	}{
		{exited: false, exitCode: 0},
		{exited: false, exitCode: 1},
		{exited: false, exitCode: 2},
		{exited: false, exitCode: 3},
		{exited: true, exitCode: 0},
		{exited: true, exitCode: 1},
		{exited: true, exitCode: 2},
		{exited: true, exitCode: 3},
	}

	exitedOffset := 32*2 + 4*6
	for _, c := range cases {
		state := &State{
			Memory:   core.NewMemory(),
			Exited:   c.exited,
			ExitCode: c.exitCode,
		}

		actualWitness, actualStateHash := state.EncodeWitness()
		require.Equal(t, len(actualWitness), STATE_WITNESS_SIZE, "Incorrect witness size")

		expectedWitness := make(StateWitness, 226)
		memRoot := state.Memory.MerkleRoot()
		copy(expectedWitness[:32], memRoot[:])
		expectedWitness[exitedOffset] = c.exitCode
		var exited uint8
		if c.exited {
			exited = 1
		}
		expectedWitness[exitedOffset+1] = uint8(exited)
		require.EqualValues(t, expectedWitness[:], actualWitness[:], "Incorrect witness")

		expectedStateHash := crypto.Keccak256Hash(actualWitness)
		expectedStateHash[0] = vmStatus(c.exited, c.exitCode)
		require.Equal(t, expectedStateHash, actualStateHash, "Incorrect state hash")
	}
}

func TestHello(t *testing.T) {
	state := test_util.LoadELFProgram(t, "../example/bin/hello.elf", CreateInitialState)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := NewInstrumentedState(state, nil, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))

	for i := 0; i < 400_000; i++ {
		if us.state.Exited {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, "hello world!\n", stdOutBuf.String(), "stdout says hello")
	require.Equal(t, "", stdErrBuf.String(), "stderr silent")
}

var _ core.PreimageOracle = (*test_util.TestOracle)(nil)

func TestClaim(t *testing.T) {
	state := test_util.LoadELFProgram(t, "../example/bin/claim.elf", CreateInitialState)

	oracle, expectedStdOut, expectedStdErr := test_util.ClaimTestOracle(t)

	var stdOutBuf, stdErrBuf bytes.Buffer
	us := NewInstrumentedState(state, oracle, io.MultiWriter(&stdOutBuf, os.Stdout), io.MultiWriter(&stdErrBuf, os.Stderr))

	for i := 0; i < 2000_000; i++ {
		if us.GetState().GetExited() {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
	}

	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")

	require.Equal(t, expectedStdOut, stdOutBuf.String(), "stdout")
	require.Equal(t, expectedStdErr, stdErrBuf.String(), "stderr")
}

func TestAlloc(t *testing.T) {
	t.Skip("TODO(client-pod#906): Currently fails on Single threaded Cannon. Re-enable for the MT FPVM")

	state := test_util.LoadELFProgram(t, "../example/bin/alloc.elf", CreateInitialState)
	const numAllocs = 100 // where each alloc is a 32 MiB chunk
	oracle := test_util.AllocOracle(t, numAllocs)

	// completes in ~870 M steps
	us := NewInstrumentedState(state, oracle, os.Stdout, os.Stderr)
	for i := 0; i < 20_000_000_000; i++ {
		if us.GetState().GetExited() {
			break
		}
		_, err := us.Step(false)
		require.NoError(t, err)
		if state.Step%10_000_000 == 0 {
			t.Logf("Completed %d steps", state.Step)
		}
	}
	t.Logf("Completed in %d steps", state.Step)
	require.True(t, state.Exited, "must complete program")
	require.Equal(t, uint8(0), state.ExitCode, "exit with 0")
	require.Less(t, state.Memory.PageCount()*core.PageSize, 1*1024*1024*1024, "must not allocate more than 1 GiB")
}

func TestStateJSONCodec(t *testing.T) {
	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")
	state, err := patch.LoadELF(elfProgram, CreateInitialState)
	require.NoError(t, err, "load ELF into state")

	stateJSON, err := state.MarshalJSON()
	require.NoError(t, err)

	newState := new(State)
	require.NoError(t, newState.UnmarshalJSON(stateJSON))

	require.Equal(t, state.PreimageKey, newState.PreimageKey)
	require.Equal(t, state.PreimageOffset, newState.PreimageOffset)
	require.Equal(t, state.Cpu, newState.Cpu)
	require.Equal(t, state.Heap, newState.Heap)
	require.Equal(t, state.ExitCode, newState.ExitCode)
	require.Equal(t, state.Exited, newState.Exited)
	require.Equal(t, state.Memory.MerkleRoot(), newState.Memory.MerkleRoot())
	require.Equal(t, state.Registers, newState.Registers)
	require.Equal(t, state.Step, newState.Step)
}
