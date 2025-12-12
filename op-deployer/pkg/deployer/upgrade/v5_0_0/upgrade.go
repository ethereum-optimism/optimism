package v5_0_0

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
	GameTypeCannonKona         GameType = 2
)

// UpgradeOPChainInput is the top-level input for upgrading an OP Chain.
type UpgradeOPChainInput struct {
	Prank          common.Address `json:"prank"`
	Opcm           common.Address `json:"opcm"`
	UpgradeInputV2 UpgradeInputV2 `evm:"-" json:"upgradeInput"`
}

// UpgradeInput contains the configuration for upgrading an OP Chain.
type UpgradeInputV2 struct {
	SystemConfig       common.Address
	DisputeGameConfigs []DisputeGameConfig
	ExtraInstructions  []ExtraInstruction
}

// DisputeGameConfig contains configuration for a dispute game.
type DisputeGameConfig struct {
	Enabled  bool
	InitBond *big.Int
	GameType GameType
	GameArgs []byte
}

// ExtraInstruction represents additional upgrade instructions.
type ExtraInstruction struct {
	Key  string `json:"key"`
	Data []byte `json:"data"`
}

var upgradeInputEncoder = w3.MustNewFunc(
	"dummy((address systemConfig,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(string key,bytes data)[] extraInstructions))",
	"",
)

// UpgradeInput returns the ABI-encoded upgrade input.
func (u *UpgradeOPChainInput) UpgradeInput() ([]byte, error) {
	input := UpgradeInputV2{
		SystemConfig:       u.UpgradeInputV2.SystemConfig,
		DisputeGameConfigs: make([]DisputeGameConfig, len(u.UpgradeInputV2.DisputeGameConfigs)),
		ExtraInstructions:  make([]ExtraInstruction, len(u.UpgradeInputV2.ExtraInstructions)),
	}

	for i, dgc := range u.UpgradeInputV2.DisputeGameConfigs {
		input.DisputeGameConfigs[i] = DisputeGameConfig{
			Enabled:  dgc.Enabled,
			InitBond: (*big.Int)(dgc.InitBond),
			GameType: dgc.GameType,
			GameArgs: dgc.GameArgs,
		}
	}

	for i, ei := range u.UpgradeInputV2.ExtraInstructions {
		input.ExtraInstructions[i] = ExtraInstruction{
			Key:  ei.Key,
			Data: ei.Data,
		}
	}

	data, err := upgradeInputEncoder.EncodeArgs(&input)
	if err != nil {
		return nil, fmt.Errorf("failed to encode upgrade input: %w", err)
	}

	// Strip the 4-byte function selector
	return data[4:], nil
}

// UpgradeOPChain is the script interface for upgrading an OP Chain.
type UpgradeOPChain struct {
	Run func(input common.Address)
}

// Upgrade executes the OP Chain upgrade script.
func Upgrade(host *script.Host, input UpgradeOPChainInput) error {
	return opcm.RunScriptVoid(host, input, "UpgradeOPChain.s.sol", "UpgradeOPChain")
}

// Upgrader implements the upgrade interface for v5.0.0.
type Upgrader struct{}

// Upgrade executes the upgrade with the given input.
func (u *Upgrader) Upgrade(host *script.Host, input json.RawMessage) error {
	var upgradeInput UpgradeOPChainInput
	if err := json.Unmarshal(input, &upgradeInput); err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return Upgrade(host, upgradeInput)
}

// ArtifactsURL returns the URL for the artifacts for this version.
func (u *Upgrader) ArtifactsURL() string {
	return artifacts.CreateHttpLocator("579f43b5bbb43e74216b7ed33125280567df86eaf00f7621f354e4a68c07323e")
}

// DefaultUpgrader is the default upgrader instance for v5.0.0.
var DefaultUpgrader = new(Upgrader)
