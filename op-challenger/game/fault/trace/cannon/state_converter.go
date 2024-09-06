package cannon

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm/singlethreaded"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-service/ioutil"
)

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
	witness, witnessHash := state.EncodeWitness()
	return &utils.ProofData{
		ClaimValue:   witnessHash,
		StateData:    witness,
		ProofData:    []byte{},
		OracleKey:    nil,
		OracleValue:  nil,
		OracleOffset: 0,
	}, state.Step, state.Exited, nil
}

func parseState(path string) (*singlethreaded.State, error) {
	file, err := ioutil.OpenDecompressed(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open state file (%v): %w", path, err)
	}
	return parseStateFromReader(file)
}

func parseStateFromReader(in io.ReadCloser) (*singlethreaded.State, error) {
	defer in.Close()
	var state singlethreaded.State
	if err := json.NewDecoder(in).Decode(&state); err != nil {
		return nil, fmt.Errorf("invalid mipsevm state: %w", err)
	}
	return &state, nil
}
