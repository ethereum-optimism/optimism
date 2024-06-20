package asterisc

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
	"github.com/ethereum/go-ethereum/common"
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
