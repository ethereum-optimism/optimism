package asterisc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"os/exec"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
	"github.com/ethereum-optimism/optimism/op-service/serialize"
)

var asteriscWitnessLen = 362

// The state struct will be read from json.
// other fields included in json are specific to FPVM implementation, and not required for trace provider.
type VMState struct {
	PC        uint64        `json:"pc"`
	Exited    bool          `json:"exited"`
	Step      uint64        `json:"step"`
	Witness   hexutil.Bytes `json:"witness"`
	StateHash common.Hash   `json:"stateHash"`
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
	state, err := LoadVMStateFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("invalid asterisc VM state %w", err)
	}
	if err := state.validateState(); err != nil {
		return nil, fmt.Errorf("invalid asterisc VM state %w", err)
	}
	return state, nil
}

type StateConverter struct {
	vmConfig    vm.Config
	cmdExecutor func(ctx context.Context, binary string, args ...string) (stdOut string, stdErr string, err error)
}

func NewStateConverter(vmConfig vm.Config) *StateConverter {
	return &StateConverter{
		vmConfig:    vmConfig,
		cmdExecutor: runCmd,
	}
}

func (c *StateConverter) ConvertStateToProof(ctx context.Context, statePath string) (*utils.ProofData, uint64, bool, error) {
	stdOut, stdErr, err := c.cmdExecutor(ctx, c.vmConfig.VmBin, "witness", "--input", statePath)
	if err != nil {
		return nil, 0, false, fmt.Errorf("state conversion failed: %w (%s)", err, stdErr)
	}
	var data VMState
	if err := json.Unmarshal([]byte(stdOut), &data); err != nil {
		return nil, 0, false, fmt.Errorf("failed to parse state data: %w", err)
	}
	// Extend the trace out to the full length using a no-op instruction that doesn't change any state
	// No execution is done, so no proof-data or oracle values are required.
	return &utils.ProofData{
		ClaimValue:   data.StateHash,
		StateData:    data.Witness,
		ProofData:    []byte{},
		OracleKey:    nil,
		OracleValue:  nil,
		OracleOffset: 0,
	}, data.Step, data.Exited, nil
}

func LoadVMStateFromFile(path string) (*VMState, error) {
	if !serialize.IsBinaryFile(path) {
		return jsonutil.LoadJSON[VMState](path)
	}
	return serialize.LoadSerializedBinary[VMState](path)
}

func runCmd(ctx context.Context, binary string, args ...string) (stdOut string, stdErr string, err error) {
	var outBuf bytes.Buffer
	var errBuf bytes.Buffer
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	stdOut = outBuf.String()
	stdErr = errBuf.String()
	return
}
