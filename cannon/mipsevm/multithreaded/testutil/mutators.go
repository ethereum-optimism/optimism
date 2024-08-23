package testutil

import (
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/multithreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type StateMutatorMultiThreaded struct {
	state *multithreaded.State
}

var _ testutil.StateMutator = (*StateMutatorMultiThreaded)(nil)

func NewStateMutatorMultiThreaded(state *multithreaded.State) testutil.StateMutator {
	return &StateMutatorMultiThreaded{state: state}
}

func (m *StateMutatorMultiThreaded) Randomize(randSeed int64) {
	r := rand.New(rand.NewSource(randSeed))

	step := testutil.RandStep(r)

	m.state.PreimageKey = testutil.RandHash(r)
	m.state.PreimageOffset = r.Uint32()
	m.state.Heap = r.Uint32()
	m.state.Step = step
	m.state.LastHint = testutil.RandHint(r)
	m.state.StepsSinceLastContextSwitch = uint64(r.Intn(exec.SchedQuantum))

	// Randomize threads
	activeStackThreads := r.Intn(2) + 1
	inactiveStackThreads := r.Intn(3)
	traverseRight := r.Intn(2) == 1
	SetupThreads(randSeed+1, m.state, traverseRight, activeStackThreads, inactiveStackThreads)
}

func (m *StateMutatorMultiThreaded) SetHI(val uint32) {
	m.state.GetCurrentThread().Cpu.HI = val
}

func (m *StateMutatorMultiThreaded) SetLO(val uint32) {
	m.state.GetCurrentThread().Cpu.LO = val
}

func (m *StateMutatorMultiThreaded) SetExitCode(val uint8) {
	m.state.ExitCode = val
}

func (m *StateMutatorMultiThreaded) SetExited(val bool) {
	m.state.Exited = val
}

func (m *StateMutatorMultiThreaded) SetPC(val uint32) {
	thread := m.state.GetCurrentThread()
	thread.Cpu.PC = val
}

func (m *StateMutatorMultiThreaded) SetHeap(val uint32) {
	m.state.Heap = val
}

func (m *StateMutatorMultiThreaded) SetNextPC(val uint32) {
	thread := m.state.GetCurrentThread()
	thread.Cpu.NextPC = val
}

func (m *StateMutatorMultiThreaded) SetLastHint(val hexutil.Bytes) {
	m.state.LastHint = val
}

func (m *StateMutatorMultiThreaded) SetPreimageKey(val common.Hash) {
	m.state.PreimageKey = val
}

func (m *StateMutatorMultiThreaded) SetPreimageOffset(val uint32) {
	m.state.PreimageOffset = val
}

func (m *StateMutatorMultiThreaded) SetStep(val uint64) {
	m.state.Step = val
}