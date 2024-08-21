package testutil

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

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

func (m *StateMutatorMultiThreaded) GetRegistersRef() *[32]uint32 {
	return m.state.GetRegistersRef()
}
