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

// UpgradeInputV2 represents the new upgrade input in OPCM v2.
type UpgradeInputV2 struct {
	SystemConfig       common.Address      `json:"systemConfig"`
	DisputeGameConfigs []DisputeGameConfig `json:"disputeGameConfigs"`
	ExtraInstructions  []ExtraInstruction  `json:"extraInstructions"`
}

// DisputeGameConfig represents the configuration for a dispute game.
type DisputeGameConfig struct {
	Enabled  bool     `json:"enabled"`
	InitBond *big.Int `json:"initBond"`
	GameType GameType `json:"gameType"`
	GameArgs []byte   `json:"gameArgs"`
}

// ExtraInstruction represents an additional upgrade instruction for the upgrade on OPCM v2.
type ExtraInstruction struct {
	Key  string `json:"key"`
	Data []byte `json:"data"`
}

// GameType represents the type of dispute game.
type GameType uint32

const (
	GameTypeCannon             GameType = 0
	GameTypePermissionedCannon GameType = 1
	GameTypeCannonKona         GameType = 8
)

// OPChainConfig represents the configuration for an OP Chain upgrade on OPCM v1.
type OPChainConfig struct {
	SystemConfigProxy  common.Address `json:"systemConfigProxy"`
	CannonPrestate     common.Hash    `json:"cannonPrestate"`
	CannonKonaPrestate common.Hash    `json:"cannonKonaPrestate"`
}

var upgradeInputEncoder = w3.MustNewFunc("dummy((address systemConfig,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(string key,bytes data)[] extraInstructions))",
	"")

var opChainConfigEncoder = w3.MustNewFunc("dummy((address systemConfigProxy,bytes32 cannonPrestate,bytes32 cannonKonaPrestate)[])", "")

func (u *UpgradeOPChainInput) EncodedOpChainConfigs() ([]byte, error) {
	data, err := opChainConfigEncoder.EncodeArgs(u.ChainConfigs)
	if err != nil {
		return nil, fmt.Errorf("failed to encode chain configs: %w", err)
	}
	return data[4:], nil
}

func (u *UpgradeOPChainInput) EncodedUpgradeInputV2() ([]byte, error) {
	data, err := upgradeInputEncoder.EncodeArgs(u.UpgradeInputV2)
	if err != nil {
		return nil, fmt.Errorf("failed to encode upgrade input: %w", err)
	}

	return data[4:], nil
}

type UpgradeOPChain struct {
	Run func(input common.Address)
}

func Upgrade(host *script.Host, input UpgradeOPChainInput) error {
	// We need to check which of the two versions of the input we are using.
	var encodedUpgradeInput []byte
	var encodedError error
	if input.UpgradeInputV2 == nil && len(input.ChainConfigs) == 0 {
		return fmt.Errorf("failed to read either an upgrade input or config array")
	} else if input.UpgradeInputV2 != nil {
		encodedUpgradeInput, encodedError = input.EncodedUpgradeInputV2()
	} else {
		encodedUpgradeInput, encodedError = input.EncodedOpChainConfigs()
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
