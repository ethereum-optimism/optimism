package embedded

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/artifacts"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

// GameType represents the type of dispute game.
type GameType uint32

const (
	GameTypeCannon             GameType = 0
	GameTypePermissionedCannon GameType = 1
	GameTypeSuperCannon        GameType = 4
	GameTypeSuperPermCannon    GameType = 5
	GameTypeCannonKona         GameType = 8
	GameTypeSuperCannonKona    GameType = 9
)

var (
	// This is used to encode the fault dispute game config for the upgrade input
	faultEncoder = w3.MustNewFunc("dummy((bytes32 absolutePrestate))", "")

	// This is used to encode the permissioned dispute game config for the upgrade input
	permEncoder = w3.MustNewFunc("dummy((bytes32 absolutePrestate,address proposer,address challenger))", "")

	// This is used to encode the upgrade input for the upgrade input
	upgradeInputEncoder = w3.MustNewFunc("dummy((address systemConfig,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(string key,bytes data)[] extraInstructions))",
		"")

	// This is used to encode the OP Chain config for the upgrade input
	opChainConfigEncoder = w3.MustNewFunc("dummy((address systemConfigProxy,bytes32 cannonPrestate,bytes32 cannonKonaPrestate)[])", "")
)

// ScriptInput represents the input struct that is actually passed to the script.
// It contains the prank, opcm, and upgrade input.
type ScriptInput struct {
	Prank        common.Address `evm:"prank"`
	Opcm         common.Address `evm:"opcm"`
	UpgradeInput []byte         `evm:"upgradeInput"`
}

// UpgradeOPChainInput represents the struct that is read from the config file.
// It contains both fields for the old and new upgrade input.
type UpgradeOPChainInput struct {
	Prank          common.Address  `json:"prank"`
	Opcm           common.Address  `json:"opcm"`
	ChainConfigs   []OPChainConfig `json:"chainConfigs,omitempty"`
	UpgradeInputV2 *UpgradeInputV2 `json:"upgradeInput,omitempty"`
}

// OPChainConfig represents the configuration for an OP Chain upgrade on OPCM v1.
type OPChainConfig struct {
	SystemConfigProxy  common.Address `json:"systemConfigProxy"`
	CannonPrestate     common.Hash    `json:"cannonPrestate"`
	CannonKonaPrestate common.Hash    `json:"cannonKonaPrestate"`
}

// UpgradeInputV2 represents the new upgrade input in OPCM v2.
type UpgradeInputV2 struct {
	SystemConfig       common.Address      `json:"systemConfig"`
	DisputeGameConfigs []DisputeGameConfig `json:"disputeGameConfigs"`
	ExtraInstructions  []ExtraInstruction  `json:"extraInstructions"`
}

// DisputeGameConfig represents the configuration for a dispute game.
type DisputeGameConfig struct {
	Enabled                       bool                           `json:"enabled"`
	InitBond                      *big.Int                       `json:"initBond"`
	GameType                      GameType                       `json:"gameType"`
	FaultDisputeGameConfig        *FaultDisputeGameConfig        `json:"faultDisputeGameConfig,omitempty"`
	PermissionedDisputeGameConfig *PermissionedDisputeGameConfig `json:"permissionedDisputeGameConfig,omitempty"`
}

// ExtraInstruction represents an additional upgrade instruction for the upgrade on OPCM v2.
type ExtraInstruction struct {
	Key  string `json:"key"`
	Data []byte `json:"data"`
}

// FaultDisputeGameConfig represents the configuration for a fault dispute game.
// It contains the absolute prestate of the fault dispute game.
type FaultDisputeGameConfig struct {
	AbsolutePrestate common.Hash `json:"absolutePrestate"`
}

// PermissionedDisputeGameConfig represents the configuration for a permissioned dispute game.
// It contains the absolute prestate, proposer, and challenger of the permissioned dispute game.
type PermissionedDisputeGameConfig struct {
	AbsolutePrestate common.Hash    `json:"absolutePrestate"`
	Proposer         common.Address `json:"proposer"`
	Challenger       common.Address `json:"challenger"`
}

// EncodableUpgradeInput is an intermediate struct that matches the encoder expectation for the UpgradeInputV2 struct.
type EncodableUpgradeInput struct {
	SystemConfig       common.Address
	DisputeGameConfigs []EncodableDisputeGameConfig
	ExtraInstructions  []ExtraInstruction
}

// EncodableDisputeGameConfig is an intermediate struct that matches the encoder expectation.
type EncodableDisputeGameConfig struct {
	Enabled  bool
	InitBond *big.Int
	GameType uint32
	GameArgs []byte
}

// EncodedOpChainConfigs encodes the OP Chain configs for the upgrade input, assumes is not nil
func (u *UpgradeOPChainInput) EncodedOpChainConfigs() ([]byte, error) {
	data, err := opChainConfigEncoder.EncodeArgs(u.ChainConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode chain configs: %w", err)
	}
	return data[4:], nil
}

// EncodedUpgradeInputV2 encodes the upgrade input for the upgrade input, assumes is not nil
func (u *UpgradeOPChainInput) EncodedUpgradeInputV2() ([]byte, error) {

	encodableConfigs := make([]EncodableDisputeGameConfig, len(u.UpgradeInputV2.DisputeGameConfigs))

	// Validate and encode each game config.
	// We iterate over the game configs in the upgrade input config and encode them into the encodable configs.
	// We return an error if a game config is not valid.
	for i, gameConfig := range u.UpgradeInputV2.DisputeGameConfigs {
		var gameArgs []byte
		var err error

		if gameConfig.Enabled {
			switch gameConfig.GameType {
			case GameTypeCannon, GameTypeCannonKona:
				if gameConfig.FaultDisputeGameConfig == nil {
					return nil, fmt.Errorf("faultDisputeGameConfig is required for game type %d", gameConfig.GameType)
				}
				// Encode the fault dispute game args
				gameArgs, err = faultEncoder.EncodeArgs(gameConfig.FaultDisputeGameConfig)
				if err != nil {
					return nil, fmt.Errorf("failed to encode fault game config: %w", err)
				}
			case GameTypePermissionedCannon:
				if gameConfig.PermissionedDisputeGameConfig == nil {
					return nil, fmt.Errorf("permissionedDisputeGameConfig is required for game type %d", gameConfig.GameType)
				}
				// Encode the permissioned dispute game args
				gameArgs, err = permEncoder.EncodeArgs(gameConfig.PermissionedDisputeGameConfig)
				if err != nil {
					return nil, fmt.Errorf("failed to encode permissioned game config: %w", err)
				}
			default:
				return nil, fmt.Errorf("invalid game type %d for opcm v2", gameConfig.GameType)
			}

			// Edge case check when the encoded game args length is less than 4
			if len(gameArgs) < 4 {
				return nil, fmt.Errorf("encoded game args length is less than 4 for game type %d", gameConfig.GameType)
			}

			// Skip the selector bytes
			gameArgs = gameArgs[4:]
		}

		encodableConfigs[i] = EncodableDisputeGameConfig{
			Enabled:  gameConfig.Enabled,
			InitBond: gameConfig.InitBond,
			GameType: uint32(gameConfig.GameType),
			GameArgs: gameArgs,
		}
	}

	// Create encodable input
	encodableInput := EncodableUpgradeInput{
		SystemConfig:       u.UpgradeInputV2.SystemConfig,
		DisputeGameConfigs: encodableConfigs,
		ExtraInstructions:  u.UpgradeInputV2.ExtraInstructions,
	}

	data, err := upgradeInputEncoder.EncodeArgs(encodableInput)
	if err != nil {
		return nil, fmt.Errorf("failed to encode upgrade input: %w", err)
	}

	return data[4:], nil
}

type UpgradeOPChain struct {
	Run func(input common.Address)
}

func Upgrade(host *script.Host, input UpgradeOPChainInput) error {
	// Determine which input format to use and encode it
	var encodedUpgradeInput []byte
	var encodedError error

	if input.UpgradeInputV2 != nil {
		// Prefer V2 input if present
		encodedUpgradeInput, encodedError = input.EncodedUpgradeInputV2()
	} else if len(input.ChainConfigs) > 0 {
		// Fall back to V1 input if V2 is not present
		encodedUpgradeInput, encodedError = input.EncodedOpChainConfigs()
	} else {
		// Neither input format is present
		return fmt.Errorf("failed to read either an upgrade input or config array")
	}

	if encodedError != nil {
		return encodedError
	}

	scriptInput := ScriptInput{
		Prank:        input.Prank,
		Opcm:         input.Opcm,
		UpgradeInput: encodedUpgradeInput,
	}
	return opcm.RunScriptVoid[ScriptInput](host, scriptInput, "UpgradeOPChain.s.sol", "UpgradeOPChain")
}

type Upgrader struct{}

func (u *Upgrader) Upgrade(host *script.Host, input json.RawMessage) error {
	var upgradeInput UpgradeOPChainInput
	if err := json.Unmarshal(input, &upgradeInput); err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return Upgrade(host, upgradeInput)
}

func (u *Upgrader) ArtifactsURL() string {
	return artifacts.EmbeddedLocatorString
}

var DefaultUpgrader = new(Upgrader)
