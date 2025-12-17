package multithreaded

import (
	"io"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/arch"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
)

type InstructionDetails struct {
	insn, opcode, fun uint32
}
type InstrumentedState struct {
	state *State

	log    log.Logger
	stdOut io.Writer
	stdErr io.Writer

	memoryTracker *exec.MemoryTrackerImpl
	stackTracker  ThreadedStackTracker
	statsTracker  StatsTracker

	preimageOracle *exec.TrackingPreimageOracleReader
	meta           mipsevm.Metadata

	cached_decode []InstructionDetails
	features      mipsevm.FeatureToggles
}

var _ mipsevm.FPVM = (*InstrumentedState)(nil)

func NewInstrumentedState(state *State, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer, log log.Logger, meta mipsevm.Metadata, features mipsevm.FeatureToggles) *InstrumentedState {
	memLen := len(state.Memory.MappedRegions[0].Data)
	cached_decode := make([]InstructionDetails, memLen/4)

	// Perform eager decode of all mapped code
	for pc := Word(0); pc < Word(memLen); pc += 4 {
		insn, opcode, fun := exec.GetInstructionDetails(pc, state.Memory)
		cached_decode[pc/4] = InstructionDetails{insn, opcode, fun}
	}

	return &InstrumentedState{
		state:          state,
		log:            log,
		stdOut:         stdOut,
		stdErr:         stdErr,
		memoryTracker:  exec.NewMemoryTracker(state.Memory),
		stackTracker:   &NoopThreadedStackTracker{},
		statsTracker:   NoopStatsTracker(),
		preimageOracle: exec.NewTrackingPreimageOracleReader(po),
		meta:           meta,
		cached_decode:  cached_decode,
		features:       features,
	}
}

func (m *InstrumentedState) InitDebug() error {
	stackTracker, err := NewThreadedStackTracker(m.state, m.meta)
	if err != nil {
		return err
	}
	m.stackTracker = stackTracker
	return nil
}

func (m *InstrumentedState) EnableStats() {
	m.statsTracker = NewStatsTracker()
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
		memProof2 := m.memoryTracker.MemProof2()
		wit.ProofData = append(wit.ProofData, memProof[:]...)
		wit.ProofData = append(wit.ProofData, memProof2[:]...)
		lastPreimageKey, lastPreimage, lastPreimageOffset := m.preimageOracle.LastPreimage()
		if lastPreimageOffset != ^arch.Word(0) {
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

func (m *InstrumentedState) LastPreimage() ([32]byte, []byte, arch.Word) {
	return m.preimageOracle.LastPreimage()
}

func (m *InstrumentedState) GetState() mipsevm.FPVMState {
	return m.state
}

func (m *InstrumentedState) GetDebugInfo() *mipsevm.DebugInfo {
	debugInfo := &mipsevm.DebugInfo{
		Pages:               m.state.Memory.PageCount(),
		MemoryUsed:          hexutil.Uint64(m.state.Memory.UsageRaw()),
		NumPreimageRequests: m.preimageOracle.NumPreimageRequests(),
		TotalPreimageSize:   m.preimageOracle.TotalPreimageSize(),
		TotalSteps:          m.state.GetStep(),
	}
	m.statsTracker.populateDebugInfo(debugInfo)
	return debugInfo
}

func (m *InstrumentedState) Traceback() {
	m.stackTracker.Traceback()
}

func (m *InstrumentedState) LookupSymbol(addr arch.Word) string {
	if m.meta == nil {
		return ""
	}
	return m.meta.LookupSymbol(addr)
}

func (m *InstrumentedState) UpdateInstructionCache(pc arch.Word) {
	idx := pc / 4
	if int(idx) < len(m.cached_decode) {
		insn, opcode, fun := exec.GetInstructionDetails(pc, m.state.Memory)
		m.cached_decode[idx] = InstructionDetails{insn, opcode, fun}
	}
}
