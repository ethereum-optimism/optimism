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
	GameTypeSuperPermCannon    GameType = 5
	GameTypeCannonKona         GameType = 8
	GameTypeSuperCannonKona    GameType = 9
	GameTypeZKDisputeGame      GameType = 10
)

var (
	// This is used to encode the fault dispute game config for the upgrade input
	faultEncoder = w3.MustNewFunc("dummy((bytes32 absolutePrestate))", "")

	// This is used to encode the permissioned dispute game config for the upgrade input
	permEncoder = w3.MustNewFunc("dummy((bytes32 absolutePrestate,address proposer,address challenger))", "")

	// This is used to encode the ZK dispute game config for the upgrade input
	zkEncoder = w3.MustNewFunc("dummy((bytes32 absolutePrestate,address verifier,uint64 maxChallengeDuration,uint64 maxProveDuration,uint256 challengerBond))", "")

	// This is used to encode the upgrade input for the upgrade input
	upgradeInputEncoder = w3.MustNewFunc("dummy((address systemConfig,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(string key,bytes data)[] extraInstructions))",
		"")
)

// ScriptInput represents the input struct that is actually passed to the script.
// It contains the prank, opcm, and upgrade input.
type ScriptInput struct {
	Prank        common.Address `evm:"prank"`
	Opcm         common.Address `evm:"opcm"`
	UpgradeInput []byte         `evm:"upgradeInput"`
}

// UpgradeOPChainInput represents the struct that is read from the config file.
type UpgradeOPChainInput struct {
	Prank          common.Address  `json:"prank"`
	Opcm           common.Address  `json:"opcm"`
	UpgradeInputV2 *UpgradeInputV2 `json:"upgradeInput,omitempty"`
}

// UpgradeInputV2 represents the upgrade input for OPCM v2.
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
	ZKDisputeGameConfig           *ZKDisputeGameConfig           `json:"zkDisputeGameConfig,omitempty"`
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

// ZKDisputeGameConfig represents the configuration for a ZK dispute game.
// It contains the absolute prestate, verifier address, challenge/prove durations, and challenger bond.
type ZKDisputeGameConfig struct {
	AbsolutePrestate     common.Hash    `json:"absolutePrestate"`
	Verifier             common.Address `json:"verifier"`
	MaxChallengeDuration uint64         `json:"maxChallengeDuration"`
	MaxProveDuration     uint64         `json:"maxProveDuration"`
	ChallengerBond       *big.Int       `json:"challengerBond"`
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

// EncodedUpgradeInputV2 encodes the upgrade input, assumes UpgradeInputV2 is not nil
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
			case GameTypeCannon, GameTypeCannonKona, GameTypeSuperCannonKona:
				if gameConfig.FaultDisputeGameConfig == nil {
					return nil, fmt.Errorf("faultDisputeGameConfig is required for game type %d", gameConfig.GameType)
				}
				// Encode the fault dispute game args
				gameArgs, err = faultEncoder.EncodeArgs(gameConfig.FaultDisputeGameConfig)
				if err != nil {
					return nil, fmt.Errorf("failed to encode fault game config: %w", err)
				}
			case GameTypePermissionedCannon, GameTypeSuperPermCannon:
				if gameConfig.PermissionedDisputeGameConfig == nil {
					return nil, fmt.Errorf("permissionedDisputeGameConfig is required for game type %d", gameConfig.GameType)
				}
				// Encode the permissioned dispute game args
				gameArgs, err = permEncoder.EncodeArgs(gameConfig.PermissionedDisputeGameConfig)
				if err != nil {
					return nil, fmt.Errorf("failed to encode permissioned game config: %w", err)
				}
			case GameTypeZKDisputeGame:
				if gameConfig.ZKDisputeGameConfig == nil {
					return nil, fmt.Errorf("zkDisputeGameConfig is required for game type %d", gameConfig.GameType)
				}
				// Encode the ZK dispute game args
				gameArgs, err = zkEncoder.EncodeArgs(gameConfig.ZKDisputeGameConfig)
				if err != nil {
					return nil, fmt.Errorf("failed to encode ZK game config: %w", err)
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
	if input.UpgradeInputV2 == nil {
		return fmt.Errorf("UpgradeInputV2 is required")
	}

	encodedUpgradeInput, err := input.EncodedUpgradeInputV2()
	if err != nil {
		return err
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
