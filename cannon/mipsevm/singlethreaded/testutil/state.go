package testutil

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/testutil"
)

type StateMutatorSingleThreaded struct {
	state *singlethreaded.State
}

var _ testutil.StateMutator = (*StateMutatorSingleThreaded)(nil)

func NewStateMutatorSingleThreaded(state *singlethreaded.State) testutil.StateMutator {
	return &StateMutatorSingleThreaded{state: state}
}

func (m *StateMutatorSingleThreaded) SetPC(val uint32) {
	m.state.Cpu.PC = val
}

func (m *StateMutatorSingleThreaded) SetNextPC(val uint32) {
	m.state.Cpu.NextPC = val
}

func (m *StateMutatorSingleThreaded) SetHI(val uint32) {
	m.state.Cpu.HI = val
}

func (m *StateMutatorSingleThreaded) SetLO(val uint32) {
	m.state.Cpu.LO = val
}

func (m *StateMutatorSingleThreaded) SetHeap(val uint32) {
	m.state.Heap = val
}

func (m *StateMutatorSingleThreaded) SetExitCode(val uint8) {
	m.state.ExitCode = val
}

func (m *StateMutatorSingleThreaded) SetExited(val bool) {
	m.state.Exited = val
}

func (m *StateMutatorSingleThreaded) SetLastHint(val hexutil.Bytes) {
	m.state.LastHint = val
}

func (m *StateMutatorSingleThreaded) SetPreimageKey(val common.Hash) {
	m.state.PreimageKey = val
}

func (m *StateMutatorSingleThreaded) SetPreimageOffset(val uint32) {
	m.state.PreimageOffset = val
}

func (m *StateMutatorSingleThreaded) SetStep(val uint64) {
	m.state.Step = val
}

func (m *StateMutatorSingleThreaded) GetRegistersRef() *[32]uint32 {
	return m.state.GetRegistersRef()
}
