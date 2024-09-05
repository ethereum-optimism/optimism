package multithreaded

import (
	"io"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
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

var _ mipsevm.FPVM = (*InstrumentedState)(nil)

func NewInstrumentedState(state *State, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) *InstrumentedState {
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

func NewInstrumentedStateFromFile(stateFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger) (*InstrumentedState, error) {
	state, err := jsonutil.LoadJSON[State](stateFile)
	if err != nil {
		return nil, err
	}
	return NewInstrumentedState(state, po, stdOut, stdErr, log), nil
}

func (m *InstrumentedState) InitDebug(meta *program.Metadata) error {
	stackTracker, err := NewThreadedStackTracker(m.state, meta)
	if err != nil {
		return err
	}
	m.stackTracker = stackTracker
	m.meta = meta
	return nil
}

func (m *InstrumentedState) Step(proof bool) (wit *mipsevm.StepWitness, err error) {
	m.preimageOracle.Reset()
	m.memoryTracker.Reset(proof)

	if proof {
		proofData := make([]byte, 0)
		threadProof := m.state.EncodeThreadProof()
		insnProof := m.state.Memory.MerkleProof(m.state.GetPC())
		proofData = append(proofData, threadProof[:]...)
		proofData = append(proofData, insnProof[:]...)

		encodedWitness, stateHash := m.state.EncodeWitness()
		wit = &mipsevm.StepWitness{
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
		if lastPreimageOffset != ^uint32(0) {
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

func (m *InstrumentedState) LookupSymbol(addr uint32) string {
	if m.meta == nil {
		return ""
	}
	return m.meta.LookupSymbol(addr)
}
