package singlethreaded

import (
	"errors"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/exec"
	"github.com/ethereum-optimism/optimism/cannon/mipsevm/program"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
)

type Debug struct {
	stack  []uint32
	caller []uint32
	meta   *program.Metadata
}

type InstrumentedState struct {
	state *State

	stdOut io.Writer
	stdErr io.Writer

	lastMemAccess   uint32
	memProofEnabled bool
	memProof        [28 * 32]byte

	preimageOracle *exec.TrackingPreimageOracleReader

	debug        Debug
	debugEnabled bool
}

func NewInstrumentedState(state *State, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) *InstrumentedState {
	return &InstrumentedState{
		state:          state,
		stdOut:         stdOut,
		stdErr:         stdErr,
		preimageOracle: exec.NewTrackingPreimageOracleReader(po),
	}
}

func NewInstrumentedStateFromFile(stateFile string, po mipsevm.PreimageOracle, stdOut, stdErr io.Writer) (*InstrumentedState, error) {
	state, err := jsonutil.LoadJSON[State](stateFile)
	if err != nil {
		return nil, err
	}
	return NewInstrumentedState(state, po, stdOut, stdErr), nil
}

func (m *InstrumentedState) InitDebug(meta *program.Metadata) error {
	if meta == nil {
		return errors.New("metadata is nil")
	}
	m.debugEnabled = true
	m.debug.meta = meta
	return nil
}

func (m *InstrumentedState) Step(proof bool) (wit *mipsevm.StepWitness, err error) {
	m.preimageOracle.Reset()
	m.memProofEnabled = proof
	m.lastMemAccess = ^uint32(0)

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
		wit.ProofData = append(wit.ProofData, m.memProof[:]...)
		lastPreimageKey, lastPreimage, lastPreimageOffset := m.preimageOracle.LastPreimage()
		if lastPreimageOffset != ^uint32(0) {
			wit.PreimageOffset = lastPreimageOffset
			wit.PreimageKey = lastPreimageKey
			wit.PreimageValue = lastPreimage
		}
	}
	return
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
		NumPreimageRequests: m.preimageOracle.NumPreimageRequests,
		TotalPreimageSize:   m.preimageOracle.TotalPreimageSize,
	}
}
