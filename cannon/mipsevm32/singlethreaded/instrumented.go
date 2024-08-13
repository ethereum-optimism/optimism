package singlethreaded

import (
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/cannon/logutil"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm32"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm32/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm32/program"
	"github.com/ethereum-optimism/optimism/cannon/run"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum/go-ethereum/common"
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

var _ mipsevm32.FPVM = (*InstrumentedState)(nil)

func NewInstrumentedState(state *State, po mipsevm32.PreimageOracle, stdOut, stdErr io.Writer, meta *program.Metadata) *InstrumentedState {
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

func NewInstrumentedStateFromFile(stateFile string, po mipsevm32.PreimageOracle, stdOut, stdErr io.Writer, meta *program.Metadata) (*InstrumentedState, error) {
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

func (m *InstrumentedState) Step(proof bool) (wit *run.StepWitness, err error) {
	m.preimageOracle.Reset()
	m.memoryTracker.Reset(proof)

	if proof {
		insnProof := m.state.Memory.MerkleProof(m.state.Cpu.PC)
		encodedWitness, stateHash := m.state.EncodeWitness()
		wit = &run.StepWitness{
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
			wit.PreimageOffset = uint64(lastPreimageOffset)
			wit.PreimageKey = lastPreimageKey
			wit.PreimageValue = lastPreimage
		}
	}
	return
}

func (m *InstrumentedState) CheckInfiniteLoop() bool {
	return m.sleepCheck(m.state.GetPC())
}

func (m *InstrumentedState) LastPreimage() ([32]byte, []byte, uint64) {
	preimageKey, preimage, preimageOffset := m.preimageOracle.LastPreimage()
	return preimageKey, preimage, uint64(preimageOffset)
}

func (m *InstrumentedState) GetState() mipsevm32.FPVMState {
	return m.state
}

func (m *InstrumentedState) GetDebugInfo() *run.DebugInfo {
	return &run.DebugInfo{
		Pages:               m.state.Memory.PageCount(),
		MemoryUsed:          hexutil.Uint64(m.state.Memory.UsageRaw()),
		NumPreimageRequests: m.preimageOracle.NumPreimageRequests(),
		TotalPreimageSize:   m.preimageOracle.TotalPreimageSize(),
	}
}

func (m *InstrumentedState) Traceback() {
	m.stackTracker.Traceback()
}

func (m *InstrumentedState) GetStep() uint64 {
	return m.state.GetStep()
}

func (m *InstrumentedState) GetPC() uint64 {
	return uint64(m.state.GetPC())
}

func (m *InstrumentedState) EncodeWitness() (witness []byte, hash common.Hash) {
	return m.state.EncodeWitness()
}

func (m *InstrumentedState) WriteState(path string, perm os.FileMode) error {
	return jsonutil.WriteJSON(path, m.state, perm)
}

func (m *InstrumentedState) GetExited() bool {
	return m.state.GetExited()
}

func (m *InstrumentedState) GetExitCode() uint8 {
	return m.state.GetExitCode()
}

func (m *InstrumentedState) InfoLogVars() []any {
	state := m.state
	return []any{
		"pc", logutil.HexU32(state.GetPC()),
		"insn", logutil.HexU32(state.GetMemory().GetMemory(state.GetPC())),
		"pages", state.GetMemory().PageCount(),
		"mem", state.GetMemory().Usage(),
		"name", m.meta.LookupSymbol(state.GetPC()),
	}
}
