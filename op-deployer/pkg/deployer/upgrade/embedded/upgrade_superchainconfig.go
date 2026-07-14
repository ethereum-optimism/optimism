package embedded

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/scriptbackend"
	"github.com/ethereum/go-ethereum/common"
)

type UpgradeSuperchainConfigInput struct {
	Prank             common.Address     `json:"prank"`
	Opcm              common.Address     `json:"opcm"`
	SuperchainConfig  common.Address     `json:"superchainConfig"`
	ExtraInstructions []ExtraInstruction `json:"extraInstructions,omitempty"`
}

func UpgradeSuperchainConfig(backend scriptbackend.Backend, input UpgradeSuperchainConfigInput) error {
	upgradeScript, err := scriptbackend.DeployScriptWithoutOutput[UpgradeSuperchainConfigInput](backend, "UpgradeSuperchainConfig.s.sol", "UpgradeSuperchainConfig")
	if err != nil {
		return fmt.Errorf("failed to load UpgradeSuperchainConfig script: %w", err)
	}
	err = upgradeScript.Run(input)
	if err != nil {
		return fmt.Errorf("failed to run UpgradeSuperchainConfig script: %w", err)
	}
	return nil
}

func (u *Upgrader) UpgradeSuperchainConfig(backend scriptbackend.Backend, input json.RawMessage) error {
	var upgradeInput UpgradeSuperchainConfigInput
	if err := json.Unmarshal(input, &upgradeInput); err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return UpgradeSuperchainConfig(backend, upgradeInput)
}
