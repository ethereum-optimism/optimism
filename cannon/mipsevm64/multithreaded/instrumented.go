package multithreaded

import (
	"io"
	"os"

	"github.com/ethereum-optimism/optimism/cannon/logutil"
	"github.com/ethereum-optimism/optimism/cannon/run"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm64"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm64/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm64/program"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

type InstrumentedState struct {
	state *State

	log    log.Logger
	stdOut io.Writer
	stdErr io.Writer

	memoryTracker *exec.MemoryTrackerImpl
	stackTracker  ThreadedStackTracker

	preimageOracle *exec.TrackingPreimageOracleReader
	meta           *program.Metadata
}

var _ mipsevm64.FPVM = (*InstrumentedState)(nil)

func NewInstrumentedState(state *State, po mipsevm64.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) *InstrumentedState {
	return &InstrumentedState{
		state:          state,
		log:            log,
		stdOut:         stdOut,
		stdErr:         stdErr,
		memoryTracker:  exec.NewMemoryTracker(state.Memory),
		stackTracker:   &NoopThreadedStackTracker{},
		preimageOracle: exec.NewTrackingPreimageOracleReader(po),
	}
}

func NewInstrumentedStateFromFile(stateFile string, po mipsevm64.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) (*InstrumentedState, error) {
	state, err := jsonutil.LoadJSON[State](stateFile)
	if err != nil {
		return nil, err
	}
	return NewInstrumentedState(state, po, stdOut, stdErr, log), nil
}

func (m *InstrumentedState) InitDebug(meta *program.Metadata) error {
	m.meta = meta
	stackTracker, err := NewThreadedStackTracker(m.state, meta)
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
		proofData := make([]byte, 0)
		threadProof := m.state.EncodeThreadProof()
		insnProof := m.state.Memory.MerkleProof(m.state.GetPC())
		proofData = append(proofData, threadProof[:]...)
		proofData = append(proofData, insnProof[:]...)

		encodedWitness, stateHash := m.state.EncodeWitness()
		wit = &run.StepWitness{
			State:     encodedWitness,
			StateHash: stateHash,
			ProofData: proofData,
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
		if lastPreimageOffset != ^uint64(0) {
			wit.PreimageOffset = lastPreimageOffset
			wit.PreimageKey = lastPreimageKey
			wit.PreimageValue = lastPreimage
		}
	}
	return
}

func (m *InstrumentedState) CheckInfiniteLoop() bool {
	return false
}

func (m *InstrumentedState) LastPreimage() ([32]byte, []byte, uint64) {
	return m.preimageOracle.LastPreimage()
}

func (m *InstrumentedState) GetState() mipsevm64.FPVMState {
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
	return m.state.GetPC()
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
