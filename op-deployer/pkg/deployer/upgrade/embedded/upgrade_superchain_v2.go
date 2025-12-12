package embedded

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
	"github.com/lmittmann/w3"
)

type UpgradeSuperchainV2Input struct {
	Prank                  common.Address     `json:"prank"`
	Opcm                   common.Address     `json:"opcm"`
	SuperchainConfig       common.Address     `evm:"-" json:"superchainConfig"`
	SuperchainInstructions []ExtraInstruction `evm:"-" json:"superchainInstructions"`
}

type SuperchainUpgradeInputV2 struct {
	SuperchainConfig  common.Address
	ExtraInstructions []ExtraInstruction
}

var superchainUpgradeInputEncoder = w3.MustNewFunc(
	"dummy((address superchainConfig,(string key,bytes data)[] extraInstructions))",
	"",
)

func (u *UpgradeSuperchainV2Input) SuperchainUpgradeInput() ([]byte, error) {
	input := SuperchainUpgradeInputV2{
		SuperchainConfig:  u.SuperchainConfig,
		ExtraInstructions: u.SuperchainInstructions,
	}

	data, err := superchainUpgradeInputEncoder.EncodeArgs(&input)
	if err != nil {
		return nil, fmt.Errorf("failed to encode superchain upgrade input: %w", err)
	}

	// Strip the 4-byte function selector
	return data[4:], nil
}

func UpgradeSuperchainV2(host *script.Host, input UpgradeSuperchainV2Input) error {
	return opcm.RunScriptVoid(host, input, "UpgradeSuperchainV2.s.sol", "UpgradeSuperchain")
}
