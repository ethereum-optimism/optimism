package multithreaded

import (
	"debug/elf"
	"encoding/json"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
)

func setWitnessField(witness StateWitness, fieldOffset int, fieldData []byte) {
	start := fieldOffset
	end := fieldOffset + len(fieldData)
	copy(witness[start:end], fieldData)
}

// Run through all permutations of `exited` / `exitCode` and ensure that the
// correct witness, state hash, and VM Status is produced.
func TestState_EncodeWitness(t *testing.T) {
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
		state := CreateEmptyState()
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
		expectedWitness := make(StateWitness, STATE_WITNESS_SIZE)
		setWitnessField(expectedWitness, MEMROOT_WITNESS_OFFSET, memRoot[:])
		setWitnessField(expectedWitness, PREIMAGE_KEY_WITNESS_OFFSET, preimageKey[:])
		setWitnessField(expectedWitness, PREIMAGE_OFFSET_WITNESS_OFFSET, []byte{0, 0, 0, byte(preimageOffset)})
		setWitnessField(expectedWitness, HEAP_WITNESS_OFFSET, []byte{0, 0, 0, byte(heap)})
		setWitnessField(expectedWitness, EXITCODE_WITNESS_OFFSET, []byte{c.exitCode})
		if c.exited {
			setWitnessField(expectedWitness, EXITED_WITNESS_OFFSET, []byte{1})
		}
		setWitnessField(expectedWitness, STEP_WITNESS_OFFSET, []byte{0, 0, 0, 0, 0, 0, 0, byte(step)})
		setWitnessField(expectedWitness, STEPS_SINCE_CONTEXT_SWITCH_WITNESS_OFFSET, []byte{0, 0, 0, 0, 0, 0, 0, byte(stepsSinceContextSwitch)})
		setWitnessField(expectedWitness, WAKEUP_WITNESS_OFFSET, []byte{0xFF, 0xFF, 0xFF, 0xFF})
		setWitnessField(expectedWitness, TRAVERSE_RIGHT_WITNESS_OFFSET, []byte{0})
		setWitnessField(expectedWitness, LEFT_THREADS_ROOT_WITNESS_OFFSET, leftStackRoot[:])
		setWitnessField(expectedWitness, RIGHT_THREADS_ROOT_WITNESS_OFFSET, rightStackRoot[:])
		setWitnessField(expectedWitness, THREAD_ID_WITNESS_OFFSET, []byte{0, 0, 0, 1})

		// Validate witness
		actualWitness, actualStateHash := state.EncodeWitness()
		require.Equal(t, len(actualWitness), STATE_WITNESS_SIZE, "Incorrect witness size")
		require.EqualValues(t, expectedWitness[:], actualWitness[:], "Incorrect witness")
		// Validate witness hash
		expectedStateHash := crypto.Keccak256Hash(actualWitness)
		expectedStateHash[0] = mipsevm.VmStatus(c.exited, c.exitCode)
		require.Equal(t, expectedStateHash, actualStateHash, "Incorrect state hash")
	}
}

func TestState_JSONCodec(t *testing.T) {
	elfProgram, err := elf.Open("../../testdata/example/bin/hello.elf")
	require.NoError(t, err, "open ELF file")
	state, err := program.LoadELF(elfProgram, CreateInitialState)
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

	var newState *State
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

func TestState_EmptyThreadsRoot(t *testing.T) {
	data := [64]byte{}
	expectedEmptyRoot := crypto.Keccak256Hash(data[:])

	require.Equal(t, expectedEmptyRoot, EmptyThreadsRoot)
}

func TestState_EncodeThreadProof_SingleThread(t *testing.T) {
	state := CreateEmptyState()
	// Set some fields on the active thread
	activeThread := state.GetCurrentThread()
	activeThread.Cpu.PC = 4
	activeThread.Cpu.NextPC = 8
	activeThread.Cpu.HI = 11
	activeThread.Cpu.LO = 22
	for i := 0; i < 32; i++ {
		activeThread.Registers[i] = uint32(i)
	}

	expectedProof := append([]byte{}, activeThread.serializeThread()[:]...)
	expectedProof = append(expectedProof, EmptyThreadsRoot[:]...)

	actualProof := state.EncodeThreadProof()
	require.Equal(t, THREAD_WITNESS_SIZE, len(actualProof))
	require.Equal(t, expectedProof, actualProof)
}

func TestState_EncodeThreadProof_MultipleThreads(t *testing.T) {
	state := CreateEmptyState()
	// Add some more threads
	require.Equal(t, state.TraverseRight, false, "sanity check")
	state.LeftThreadStack = append(state.LeftThreadStack, CreateEmptyThread())
	state.LeftThreadStack = append(state.LeftThreadStack, CreateEmptyThread())
	require.Equal(t, 3, len(state.LeftThreadStack), "sanity check")

	// Set some fields on our threads
	for i := 0; i < 3; i++ {
		curThread := state.LeftThreadStack[i]
		curThread.Cpu.PC = uint32(4 * i)
		curThread.Cpu.NextPC = curThread.Cpu.PC + 4
		curThread.Cpu.HI = uint32(11 + i)
		curThread.Cpu.LO = uint32(22 + i)
		for j := 0; j < 32; j++ {
			curThread.Registers[j] = uint32(j + i)
		}
	}

	expectedRoot := EmptyThreadsRoot
	for i := 0; i < 2; i++ {
		curThread := state.LeftThreadStack[i]
		hashedThread := crypto.Keccak256Hash(curThread.serializeThread())

		// root = prevRoot ++ hash(curRoot)
		hashData := append([]byte{}, expectedRoot[:]...)
		hashData = append(hashData, hashedThread[:]...)
		expectedRoot = crypto.Keccak256Hash(hashData)
	}

	expectedProof := append([]byte{}, state.GetCurrentThread().serializeThread()[:]...)
	expectedProof = append(expectedProof, expectedRoot[:]...)

	actualProof := state.EncodeThreadProof()
	require.Equal(t, THREAD_WITNESS_SIZE, len(actualProof))
	require.Equal(t, expectedProof, actualProof)
}

func TestState_EncodeThreadProof_EmptyThreadStackPanic(t *testing.T) {
	cases := []struct {
		name          string
		wakeupAddr    uint32
		traverseRight bool
	}{
		{"traverse left during wakeup traversal", uint32(99), false},
		{"traverse left during normal traversal", exec.FutexEmptyAddr, false},
		{"traverse right during wakeup traversal", uint32(99), true},
		{"traverse right during normal traversal", exec.FutexEmptyAddr, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Set up invalid state where the active stack is empty
			state := CreateEmptyState()
			state.Wakeup = c.wakeupAddr
			state.TraverseRight = c.traverseRight
			if c.traverseRight {
				state.LeftThreadStack = []*ThreadState{CreateEmptyThread()}
				state.RightThreadStack = []*ThreadState{}
			} else {
				state.LeftThreadStack = []*ThreadState{}
				state.RightThreadStack = []*ThreadState{CreateEmptyThread()}
			}

			assert.PanicsWithValue(t, "Invalid empty thread stack", func() { state.EncodeThreadProof() })
		})
	}
}
