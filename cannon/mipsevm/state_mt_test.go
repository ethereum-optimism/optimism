package mipsevm

import (
	"debug/elf"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

func setWitnessField(witness MTStateWitness, fieldOffset int, fieldData []byte) {
	start := fieldOffset
	end := fieldOffset + len(fieldData)
	copy(witness[start:end], fieldData)
}

// Run through all permutations of `exited` / `exitCode` and ensure that the
// correct witness, state hash, and VM Status is produced.
func TestMTStateHash(t *testing.T) {
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

	for _, c := range cases {
		state := CreateEmptyMTState()
		state.Exited = c.exited
		state.ExitCode = c.exitCode

		memRoot := state.Memory.MerkleRoot()
		rightStackRoot := state.RightThreadStackRoots[0]
		leftStackRoot := EmptyThreadsRoot

		// Set up expected witness
		expectedWitness := make(MTStateWitness, MT_STATE_WITNESS_SIZE)
		setWitnessField(expectedWitness, MT_WITNESS_MEMROOT_OFFSET, memRoot[:])
		setWitnessField(expectedWitness, MT_WITNESS_EXITCODE_OFFSET, []byte{c.exitCode})
		if c.exited {
			setWitnessField(expectedWitness, MT_WITNESS_EXITED_OFFSET, []byte{1})
		}
		setWitnessField(expectedWitness, MT_WITNESS_WAKEUP_OFFSET, []byte{0xFF, 0xFF, 0xFF, 0xFF})
		setWitnessField(expectedWitness, MT_WITNESS_TRAVERSE_RIGHT_OFFSET, []byte{1})
		setWitnessField(expectedWitness, MT_WITNESS_LEFT_THREADS_ROOT_OFFSET, leftStackRoot[:])
		setWitnessField(expectedWitness, MT_WITNESS_RIGHT_THREADS_ROOT_OFFSET, rightStackRoot[:])
		setWitnessField(expectedWitness, MT_WITNESS_THREAD_ID_OFFSET, []byte{0, 0, 0, 1})

		// Validate witness
		actualWitness, actualStateHash := state.EncodeWitness()
		require.Equal(t, len(actualWitness), MT_STATE_WITNESS_SIZE, "Incorrect witness size")
		require.EqualValues(t, expectedWitness[:], actualWitness[:], "Incorrect witness")
		// Validate witness hash
		expectedStateHash := crypto.Keccak256Hash(actualWitness)
		expectedStateHash[0] = vmStatus(c.exited, c.exitCode)
		require.Equal(t, expectedStateHash, actualStateHash, "Incorrect state hash")
	}
}

func TestMTStateJSONCodec(t *testing.T) {
	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")
	state, err := LoadELF(elfProgram, CreateInitialMTState)
	require.NoError(t, err, "load ELF into state")

	stateJSON, err := json.Marshal(state)
	require.NoError(t, err)

	var newState *MTState
	err = json.Unmarshal(stateJSON, &newState)
	require.NoError(t, err)

	require.Equal(t, state.PreimageKey, newState.PreimageKey)
	require.Equal(t, state.PreimageOffset, newState.PreimageOffset)
	require.Equal(t, state.Heap, newState.Heap)
	require.Equal(t, state.ExitCode, newState.ExitCode)
	require.Equal(t, state.Exited, newState.Exited)
	require.Equal(t, state.Memory.MerkleRoot(), newState.Memory.MerkleRoot())
	require.Equal(t, state.Step, newState.Step)
	require.Equal(t, state.Wakeup, newState.Wakeup)
	require.Equal(t, state.TraverseRight, newState.TraverseRight)
	require.Equal(t, state.LeftThreadStack, newState.LeftThreadStack)
	require.Equal(t, state.RightThreadStack, newState.RightThreadStack)
	require.Equal(t, state.LeftThreadStackRoots, newState.LeftThreadStackRoots)
	require.Equal(t, state.RightThreadStackRoots, newState.RightThreadStackRoots)
	require.Equal(t, state.NextThreadId, newState.NextThreadId)
	require.Equal(t, state.LastHint, newState.LastHint)
}

func TestEmptyThreadsRoot(t *testing.T) {
	data := [64]byte{}
	expectedEmptyRoot := crypto.Keccak256Hash(data[:])

	require.Equal(t, expectedEmptyRoot, EmptyThreadsRoot)
}
