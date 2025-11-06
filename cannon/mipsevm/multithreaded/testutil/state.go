package testutil

import (
	"encoding/binary"
	"math"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

func AddHintLengthPrefix(data []byte) []byte {
	dataLen := len(data)
	prefixed := make([]byte, 0, dataLen+4)
	prefixed = binary.BigEndian.AppendUint32(prefixed, uint32(dataLen))
	prefixed = append(prefixed, data...)

	return prefixed
}

func AddPreimageLengthPrefix(data []byte) []byte {
	dataLen := len(data)
	prefixed := make([]byte, 0, dataLen+8)
	prefixed = binary.BigEndian.AppendUint64(prefixed, uint64(dataLen))
	prefixed = append(prefixed, data...)

	return prefixed
}

type StateMutator struct {
	state *multithreaded.State
}

func NewStateMutator(state *multithreaded.State) *StateMutator {
	return &StateMutator{state: state}
}

func (m *StateMutator) Randomize(randSeed int64) {
	r := testutil.NewRandHelper(randSeed)

	step := r.RandStep()

	m.state.PreimageKey = r.RandHash()
	m.state.PreimageOffset = r.Word()
	m.state.Step = step
	m.state.LastHint = r.RandHint()
	m.state.StepsSinceLastContextSwitch = uint64(r.Intn(exec.SchedQuantum))

	// Randomize memory-related fields
	halfMemory := math.MaxUint32 / 2
	m.state.Heap = arch.Word(r.Intn(halfMemory) + halfMemory)
	m.state.LLReservationStatus = multithreaded.LLReservationStatus(r.Intn(3))
	if m.state.LLReservationStatus != multithreaded.LLStatusNone {
		m.state.LLAddress = arch.Word(r.Intn(halfMemory))
		m.state.LLOwnerThread = arch.Word(r.Intn(10))
	}

	// Randomize threads
	activeStackThreads := r.Intn(2) + 1
	inactiveStackThreads := r.Intn(3)
	traverseRight := r.Intn(2) == 1
	SetupThreads(randSeed+1, m.state, traverseRight, activeStackThreads, inactiveStackThreads)
}

func (m *StateMutator) SetPC(val arch.Word) {
	thread := m.state.GetCurrentThread()
	thread.Cpu.PC = val
}

func (m *StateMutator) SetNextPC(val arch.Word) {
	thread := m.state.GetCurrentThread()
	thread.Cpu.NextPC = val
}

func (m *StateMutator) SetPreimageKey(val common.Hash) {
	m.state.PreimageKey = val
}

func (m *StateMutator) SetPreimageOffset(val arch.Word) {
	m.state.PreimageOffset = val
}

func (m *StateMutator) SetStep(val uint64) {
	m.state.Step = val
}

type StateOption func(state *StateMutator)

func WithPCAndNextPC(pc arch.Word) StateOption {
	return func(state *StateMutator) {
		state.SetPC(pc)
		state.SetNextPC(pc + 4)
	}
}

func WithPreimageKey(key common.Hash) StateOption {
	return func(state *StateMutator) {
		state.SetPreimageKey(key)
	}
}

func WithPreimageOffset(offset arch.Word) StateOption {
	return func(state *StateMutator) {
		state.SetPreimageOffset(offset)
	}
}

func WithStep(step uint64) StateOption {
	return func(state *StateMutator) {
		state.SetStep(step)
	}
}

func WithRandomization(seed int64) StateOption {
	return func(mut *StateMutator) {
		mut.Randomize(seed)
	}
}

func GetMtState(t require.TestingT, vm mipsevm.FPVM) *multithreaded.State {
	return ToMTState(t, vm.GetState())
}

func ToMTState(t require.TestingT, state mipsevm.FPVMState) *multithreaded.State {
	mtState, ok := state.(*multithreaded.State)
	if !ok {
		require.Fail(t, "Failed to cast FPVMState to multithreaded State type")
	}
	return mtState
}

func RandomState(seed int) *multithreaded.State {
	state := multithreaded.CreateEmptyState()
	mut := StateMutator{state}
	mut.Randomize(int64(seed))
	return state
}
