package manage

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

// ScriptInput represents the input struct that is actually passed to the script.
// It contains the prank address, OPCM address, ABI-encoded migrate input, and optional
// ABI-encoded extra instructions.
type ScriptInput struct {
	Prank             common.Address `evm:"prank"`
	Opcm              common.Address `evm:"opcm"`
	MigrateInput      []byte         `evm:"migrateInput"`
	ExtraInstructions []byte         `evm:"extraInstructions"`
}

// InteropMigrationInput represents the struct that is read from the config file.
type InteropMigrationInput struct {
	Prank             common.Address     `json:"prank"`
	Opcm              common.Address     `json:"opcm"`
	MigrateInputV2    *MigrateInputV2    `json:"migrateInputV2,omitempty"`
	ExtraInstructions []ExtraInstruction `json:"extraInstructions,omitempty"`
}

// ExtraInstruction represents a one-off OPCM migration instruction.
type ExtraInstruction struct {
	Key  string `json:"key"`
	Data []byte `json:"data"`
}

// MigrateInputV2 represents the migrate input format for OPCM v2 (>= 7.0.0).
// Corresponds to IOPContractsManagerMigrator.MigrateInput
type MigrateInputV2 struct {
	ChainSystemConfigs        []common.Address    `json:"chainSystemConfigs"`
	DisputeGameConfigs        []DisputeGameConfig `json:"disputeGameConfigs"`
	StartingAnchorRoot        Proposal            `json:"startingAnchorRoot"`
	StartingRespectedGameType uint32              `json:"startingRespectedGameType"`
}

// DisputeGameConfig defines the configuration for a specific dispute game type.
// Corresponds to IOPContractsManagerMigrator.DisputeGameConfig
type DisputeGameConfig struct {
	Enabled  bool     `json:"enabled"`
	InitBond *big.Int `json:"initBond"`
	GameType uint32   `json:"gameType"`
	GameArgs []byte   `json:"gameArgs"`
}

// Proposal represents an L2 output root proposal used as the starting anchor for dispute games.
type Proposal struct {
	Root             common.Hash `json:"root"`
	L2SequenceNumber *big.Int    `json:"l2SequenceNumber"`
}

// InteropMigrationOutput contains the output of the interop migration script.
type InteropMigrationOutput struct {
	DisputeGameFactory common.Address `json:"disputeGameFactory"`
}

var migrateInputV2Encoder = w3.MustNewFunc(
	"dummy((address[] chainSystemConfigs,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(bytes32 root,uint256 l2SequenceNumber) startingAnchorRoot,uint32 startingRespectedGameType))",
	"",
)

var extraInstructionsEncoder = w3.MustNewFunc(
	"dummy((string key,bytes data)[])",
	"",
)

func (i *InteropMigrationInput) EncodedMigrateInputV2() ([]byte, error) {
	if i.MigrateInputV2 == nil {
		return nil, fmt.Errorf("MigrateInputV2 is nil")
	}
	data, err := migrateInputV2Encoder.EncodeArgs(i.MigrateInputV2)
	if err != nil {
		return nil, fmt.Errorf("failed to encode migrate input v2: %w", err)
	}

	if len(data) < 4 {
		return nil, fmt.Errorf("failed to encode migrate input v2: data is too short")
	}

	// Skip the function selector (first 4 bytes)
	return data[4:], nil
}

func (i *InteropMigrationInput) EncodedExtraInstructions() ([]byte, error) {
	if len(i.ExtraInstructions) == 0 {
		return nil, nil
	}

	data, err := extraInstructionsEncoder.EncodeArgs(i.ExtraInstructions)
	if err != nil {
		return nil, fmt.Errorf("failed to encode extra instructions: %w", err)
	}
	if len(data) < 4 {
		return nil, fmt.Errorf("failed to encode extra instructions: data is too short")
	}

	return data[4:], nil
}

func (output *InteropMigrationOutput) CheckOutput(input common.Address) error {
	return nil
}

func Migrate(host *script.Host, input InteropMigrationInput) (InteropMigrationOutput, error) {
	if input.MigrateInputV2 == nil {
		return InteropMigrationOutput{}, fmt.Errorf("MigrateInputV2 is required")
	}

	encodedMigrateInput, err := input.EncodedMigrateInputV2()
	if err != nil {
		return InteropMigrationOutput{}, err
	}
	encodedExtraInstructions, err := input.EncodedExtraInstructions()
	if err != nil {
		return InteropMigrationOutput{}, err
	}

	scriptInput := ScriptInput{
		Prank:             input.Prank,
		Opcm:              input.Opcm,
		MigrateInput:      encodedMigrateInput,
		ExtraInstructions: encodedExtraInstructions,
	}
	return opcm.RunScriptSingle[ScriptInput, InteropMigrationOutput](host, scriptInput, "InteropMigration.s.sol", "InteropMigration")
}
