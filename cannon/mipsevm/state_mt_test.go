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
func TestMTState_EncodeWitness(t *testing.T) {
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

	heap := uint32(12)
	preimageKey := crypto.Keccak256Hash([]byte{1, 2, 3, 4})
	preimageOffset := uint32(24)
	step := uint64(33)
	stepsSinceContextSwitch := uint64(123)
	for _, c := range cases {
		state := CreateEmptyMTState()
		state.Exited = c.exited
		state.ExitCode = c.exitCode
		state.PreimageKey = preimageKey
		state.PreimageOffset = preimageOffset
		state.Heap = heap
		state.Step = step
		state.StepsSinceLastContextSwitch = stepsSinceContextSwitch

		memRoot := state.Memory.MerkleRoot()
		leftStackRoot := state.calculateThreadStackRoot(state.LeftThreadStack)
		rightStackRoot := EmptyThreadsRoot

		// Set up expected witness
		expectedWitness := make(MTStateWitness, MT_STATE_WITNESS_SIZE)
		setWitnessField(expectedWitness, MEMROOT_MT_WITNESS_OFFSET, memRoot[:])
		setWitnessField(expectedWitness, PREIMAGE_KEY_MT_WITNESS_OFFSET, preimageKey[:])
		setWitnessField(expectedWitness, PREIMAGE_OFFSET_MT_WITNESS_OFFSET, []byte{0, 0, 0, byte(preimageOffset)})
		setWitnessField(expectedWitness, HEAP_MT_WITNESS_OFFSET, []byte{0, 0, 0, byte(heap)})
		setWitnessField(expectedWitness, EXITCODE_MT_WITNESS_OFFSET, []byte{c.exitCode})
		if c.exited {
			setWitnessField(expectedWitness, EXITED_MT_WITNESS_OFFSET, []byte{1})
		}
		setWitnessField(expectedWitness, STEP_MT_WITNESS_OFFSET, []byte{0, 0, 0, 0, 0, 0, 0, byte(step)})
		setWitnessField(expectedWitness, STEPS_SINCE_CONTEXT_SWITCH_MT_WITNESS_OFFSET, []byte{0, 0, 0, 0, 0, 0, 0, byte(stepsSinceContextSwitch)})
		setWitnessField(expectedWitness, WAKEUP_MT_WITNESS_OFFSET, []byte{0xFF, 0xFF, 0xFF, 0xFF})
		setWitnessField(expectedWitness, TRAVERSE_RIGHT_MT_WITNESS_OFFSET, []byte{0})
		setWitnessField(expectedWitness, LEFT_THREADS_ROOT_MT_WITNESS_OFFSET, leftStackRoot[:])
		setWitnessField(expectedWitness, RIGHT_THREADS_ROOT_MT_WITNESS_OFFSET, rightStackRoot[:])
		setWitnessField(expectedWitness, THREAD_ID_MT_WITNESS_OFFSET, []byte{0, 0, 0, 1})

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

func TestMTState_JSONCodec(t *testing.T) {
	elfProgram, err := elf.Open("../example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")
	state, err := LoadELF(elfProgram, CreateInitialMTState)
	require.NoError(t, err, "load ELF into state")
	// Set a few additional fields
	state.PreimageKey = crypto.Keccak256Hash([]byte{1, 2, 3, 4})
	state.PreimageOffset = 4
	state.Heap = 555
	state.Step = 99_999
	state.StepsSinceLastContextSwitch = 123
	state.Exited = true
	state.ExitCode = 2
	state.LastHint = []byte{11, 12, 13}

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
	require.Equal(t, state.StepsSinceLastContextSwitch, newState.StepsSinceLastContextSwitch)
	require.Equal(t, state.Wakeup, newState.Wakeup)
	require.Equal(t, state.TraverseRight, newState.TraverseRight)
	require.Equal(t, state.LeftThreadStack, newState.LeftThreadStack)
	require.Equal(t, state.RightThreadStack, newState.RightThreadStack)
	require.Equal(t, state.NextThreadId, newState.NextThreadId)
	require.Equal(t, state.LastHint, newState.LastHint)
}

func TestMTState_EmptyThreadsRoot(t *testing.T) {
	data := [64]byte{}
	expectedEmptyRoot := crypto.Keccak256Hash(data[:])

	require.Equal(t, expectedEmptyRoot, EmptyThreadsRoot)
}
