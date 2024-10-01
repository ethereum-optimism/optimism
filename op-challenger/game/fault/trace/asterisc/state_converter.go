package asterisc

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/cannon/mipsevm"
	"github.com/ethereum-optimism/optimism/cannon/serialize"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-service/jsonutil"
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

// Because the binary file is provided by Asterisc with all of its fields,
// we must keep in mind the rest of the fields that aren't used, but still serialized.
func (s *VMState) Deserialize(in io.Reader) error {
	bin := serialize.NewBinaryReader(in)

	// Memory
	var pageCount uint64
	if err := binary.Read(in, binary.BigEndian, &pageCount); err != nil {
		return err
	}
	for i := uint64(0); i < pageCount; i++ {
		var pageIndex uint64
		var page [4096]byte
		if err := binary.Read(in, binary.BigEndian, &pageIndex); err != nil {
			return err
		}
		if _, err := io.ReadFull(in, page[:]); err != nil {
			return err
		}
	}

	var preimageKey common.Hash
	if err := bin.ReadHash(&preimageKey); err != nil { //PreimageKey
		return err
	}
	var preimageOffset uint64
	if err := bin.ReadUInt(&preimageOffset); err != nil { // PreimageOffset
		return err
	}
	if err := bin.ReadUInt(&s.PC); err != nil {
		return err
	}
	var exitCode uint8
	if err := bin.ReadUInt(&exitCode); err != nil { // ExitCode
		return err
	}
	if err := bin.ReadBool(&s.Exited); err != nil {
		return err
	}
	if err := bin.ReadUInt(&s.Step); err != nil {
		return err
	}
	var heap uint64
	if err := bin.ReadUInt(&heap); err != nil { // Heap
		return err
	}
	var loadReservation uint64
	if err := bin.ReadUInt(&loadReservation); err != nil { // LoadReservation
		return err
	}
	for i := 0; i < 32; i++ {
		var register uint64
		if err := bin.ReadUInt(&register); err != nil { // Registers
			return err
		}
	}
	var lastHint []byte
	if err := bin.ReadBytes(&lastHint); err != nil { // LastHint
		return err
	}
	if err := bin.ReadBytes(&s.Witness); err != nil {
		return err
	}
	if err := bin.ReadHash(&s.StateHash); err != nil {
		return err
	}

	return nil
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
}

func NewStateConverter() *StateConverter {
	return &StateConverter{}
}

func (c *StateConverter) ConvertStateToProof(_ context.Context, statePath string) (*utils.ProofData, uint64, bool, error) {
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

func LoadVMStateFromFile(path string) (*VMState, error) {
	if !serialize.IsBinaryFile(path) {
		return jsonutil.LoadJSON[VMState](path)
	}
	return serialize.LoadSerializedBinary[VMState](path)
}
