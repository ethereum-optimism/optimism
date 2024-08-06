package singlethreaded

import (
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type InstrumentedState struct {
	meta       *program.Metadata
	sleepCheck program.SymbolMatcher

	state *State

	stdOut io.Writer
	stdErr io.Writer

	memoryTracker *exec.MemoryTrackerImpl
	stackTracker  exec.TraceableStackTracker

	preimageOracle *exec.TrackingPreimageOracleReader
}

var _ mipsevm.FPVM = (*InstrumentedState)(nil)

func NewInstrumentedState(state *State, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, meta *program.Metadata) *InstrumentedState {
	var sleepCheck program.SymbolMatcher
	if meta == nil {
		sleepCheck = func(addr uint32) bool { return false }
	} else {
		sleepCheck = meta.CreateSymbolMatcher("runtime.notesleep")
	}

	return &InstrumentedState{
		meta:           meta,
		sleepCheck:     sleepCheck,
		state:          state,
		stdOut:         stdOut,
		stdErr:         stdErr,
		memoryTracker:  exec.NewMemoryTracker(state.Memory),
		stackTracker:   &exec.NoopStackTracker{},
		preimageOracle: exec.NewTrackingPreimageOracleReader(po),
	}
}

func NewInstrumentedStateFromFile(stateFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, meta *program.Metadata) (*InstrumentedState, error) {
	state, err := jsonutil.LoadJSON[State](stateFile)
	if err != nil {
		return nil, err
	}
	return NewInstrumentedState(state, po, stdOut, stdErr, meta), nil
}

func (m *InstrumentedState) InitDebug() error {
	stackTracker, err := exec.NewStackTracker(m.state, m.meta)
	if err != nil {
		return err
	}
	m.stackTracker = stackTracker
	return nil
}

func (m *InstrumentedState) Step(proof bool) (wit *mipsevm.StepWitness, err error) {
	m.preimageOracle.Reset()
	m.memoryTracker.Reset(proof)

	if proof {
		insnProof := m.state.Memory.MerkleProof(m.state.Cpu.PC)
		encodedWitness, stateHash := m.state.EncodeWitness()
		wit = &mipsevm.StepWitness{
			State:     encodedWitness,
			StateHash: stateHash,
			ProofData: insnProof[:],
		}
	}
	err = m.mipsStep()
	if err != nil {
		return nil, err
	}

	if proof {
		memProof := m.memoryTracker.MemProof()
		wit.ProofData = append(wit.ProofData, memProof[:]...)
		lastPreimageKey, lastPreimage, lastPreimageOffset := m.preimageOracle.LastPreimage()
		if lastPreimageOffset != ^uint32(0) {
			wit.PreimageOffset = lastPreimageOffset
			wit.PreimageKey = lastPreimageKey
			wit.PreimageValue = lastPreimage
		}
	}
	return
}

func (m *InstrumentedState) CheckInfiniteLoop() bool {
	return m.sleepCheck(m.state.GetPC())
}

func (m *InstrumentedState) LastPreimage() ([32]byte, []byte, uint32) {
	return m.preimageOracle.LastPreimage()
}

func (m *InstrumentedState) GetState() mipsevm.FPVMState {
	return m.state
}

func (m *InstrumentedState) GetDebugInfo() *mipsevm.DebugInfo {
	return &mipsevm.DebugInfo{
		Pages:               m.state.Memory.PageCount(),
		MemoryUsed:          hexutil.Uint64(m.state.Memory.UsageRaw()),
		NumPreimageRequests: m.preimageOracle.NumPreimageRequests(),
		TotalPreimageSize:   m.preimageOracle.TotalPreimageSize(),
	}
}

func (m *InstrumentedState) Traceback() {
	m.stackTracker.Traceback()
}
