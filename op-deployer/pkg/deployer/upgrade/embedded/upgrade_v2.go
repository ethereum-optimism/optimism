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

type GameType uint32

const (
	GameTypeCannon             GameType = 0
	GameTypePermissionedCannon GameType = 1
	GameTypeCannonKona         GameType = 2
)

type UpgradeOPChainV2Input struct {
	Prank          common.Address `json:"prank"`
	Opcm           common.Address `json:"opcm"`
	UpgradeInputV2 UpgradeInputV2 `evm:"-" json:"upgradeInputV2"`
}

type UpgradeInputV2 struct {
	SystemConfig       common.Address      `json:"systemConfig"`
	DisputeGameConfigs []DisputeGameConfig `json:"disputeGameConfigs"`
	ExtraInstructions  []ExtraInstruction  `json:"extraInstructions"`
}

type DisputeGameConfig struct {
	Enabled  bool     `json:"enabled"`
	InitBond *big.Int `json:"initBond"`
	GameType GameType `json:"gameType"`
	GameArgs []byte   `json:"gameArgs"`
}

type ExtraInstruction struct {
	Key  string `json:"key"`
	Data []byte `json:"data"`
}

var upgradeInputEncoder = w3.MustNewFunc(
	"dummy((address systemConfig,(bool enabled,uint256 initBond,uint32 gameType,bytes gameArgs)[] disputeGameConfigs,(string key,bytes data)[] extraInstructions))",
	"",
)

func (u *UpgradeOPChainV2Input) UpgradeInput() ([]byte, error) {
	data, err := upgradeInputEncoder.EncodeArgs(&u.UpgradeInputV2)
	if err != nil {
		return nil, fmt.Errorf("failed to encode upgrade input: %w", err)
	}

	// Strip the 4-byte function selector
	return data[4:], nil
}

type UpgradeOPChainV2 struct {
	Run func(input common.Address)
}

func UpgradeV2(host *script.Host, input UpgradeOPChainV2Input) error {
	return opcm.RunScriptVoid(host, input, "UpgradeOPChainV2.s.sol", "UpgradeOPChain")
}

type UpgraderV2 struct{}

func (u *UpgraderV2) Upgrade(host *script.Host, input json.RawMessage) error {
	var upgradeInput UpgradeOPChainV2Input
	if err := json.Unmarshal(input, &upgradeInput); err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return UpgradeV2(host, upgradeInput)
}

func (u *UpgraderV2) ArtifactsURL() string {
	return artifacts.EmbeddedLocatorString
}

var DefaultUpgraderV2 = new(UpgraderV2)
