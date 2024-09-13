package asterisc

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

var asteriscWitnessLen = 362

// The state struct will be read from json.
// other fields included in json are specific to FPVM implementation, and not required for trace provider.
type VMState struct {
	PC        uint64      `json:"pc"`
	Exited    bool        `json:"exited"`
	Step      uint64      `json:"step"`
	Witness   []byte      `json:"witness"`
	StateHash common.Hash `json:"stateHash"`
}

func (state *VMState) validateStateHash() error {
	exitCode := state.StateHash[0]
	if exitCode >= 4 {
		return fmt.Errorf("invalid stateHash: unknown exitCode %d", exitCode)
	}
	if (state.Exited && exitCode == mipsevm.VMStatusUnfinished) || (!state.Exited && exitCode != mipsevm.VMStatusUnfinished) {
		return fmt.Errorf("invalid stateHash: invalid exitCode %d", exitCode)
	}
	return nil
}

func (state *VMState) validateWitness() error {
	witnessLen := len(state.Witness)
	if witnessLen != asteriscWitnessLen {
		return fmt.Errorf("invalid witness: Length must be 362 but got %d", witnessLen)
	}
	return nil
}

// validateState performs verification of state; it is not perfect.
// It does not recalculate whether witness nor stateHash is correctly set from state.
func (state *VMState) validateState() error {
	if err := state.validateStateHash(); err != nil {
		return err
	}
	if err := state.validateWitness(); err != nil {
		return err
	}
	return nil
}

// parseState parses state from json and goes on state validation
func parseState(path string) (*VMState, error) {
	file, err := ioutil.OpenDecompressed(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open state file (%v): %w", path, err)
	}
	return parseStateFromReader(file)
}

func parseStateFromReader(in io.ReadCloser) (*VMState, error) {
	defer in.Close()
	var state VMState
	if err := json.NewDecoder(in).Decode(&state); err != nil {
		return nil, fmt.Errorf("invalid asterisc VM state %w", err)
	}
	if err := state.validateState(); err != nil {
		return nil, fmt.Errorf("invalid asterisc VM state %w", err)
	}
	return &state, nil
}

type StateConverter struct {
}

func NewStateConverter() *StateConverter {
	return &StateConverter{}
}

func (c *StateConverter) ConvertStateToProof(statePath string) (*utils.ProofData, uint64, bool, error) {
	state, err := parseState(statePath)
	if err != nil {
		return nil, 0, false, fmt.Errorf("cannot read final state: %w", err)
	}
	// Extend the trace out to the full length using a no-op instruction that doesn't change any state
	// No execution is done, so no proof-data or oracle values are required.
	return &utils.ProofData{
		ClaimValue:   state.StateHash,
		StateData:    state.Witness,
		ProofData:    []byte{},
		OracleKey:    nil,
		OracleValue:  nil,
		OracleOffset: 0,
	}, state.Step, state.Exited, nil
}
