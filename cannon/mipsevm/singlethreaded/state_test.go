package singlethreaded

import (
	"debug/elf"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/memory"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

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
			Memory:   memory.NewMemory(),
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
		expectedStateHash[0] = mipsevm.VmStatus(c.exited, c.exitCode)
		require.Equal(t, expectedStateHash, actualStateHash, "Incorrect state hash")
	}
}

func TestStateJSONCodec(t *testing.T) {
	elfProgram, err := elf.Open("../../testdata/example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")
	state, err := program.LoadELF(elfProgram, CreateInitialState)
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
