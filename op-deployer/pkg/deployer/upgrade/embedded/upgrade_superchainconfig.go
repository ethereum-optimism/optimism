package embedded

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type UpgradeSuperchainConfigInput struct {
	Prank            common.Address `json:"prank"`
	Opcm             common.Address `json:"opcm"`
	SuperchainConfig common.Address `json:"superchainConfig"`
}

type UpgradeSuperchainConfigScript script.DeployScriptWithoutOutput[UpgradeSuperchainConfigInput]

// NewDeployImplementationsScript loads and validates the DeployImplementations script contract
func NewUpgradeSuperchainConfigScript(host *script.Host) (UpgradeSuperchainConfigScript, error) {
	return script.NewDeployScriptWithoutOutputFromFile[UpgradeSuperchainConfigInput](host, "UpgradeSuperchainConfig.s.sol", "UpgradeSuperchainConfig")
}

func UpgradeSuperchainConfig(host *script.Host, input UpgradeSuperchainConfigInput) error {
	upgradeScript, err := NewUpgradeSuperchainConfigScript(host)
	if err != nil {
		return fmt.Errorf("failed to load UpgradeSuperchainConfig script: %w", err)
	}
	err = upgradeScript.Run(input)
	if err != nil {
		return fmt.Errorf("failed to run UpgradeSuperchainConfig script: %w", err)
	}
	return nil
}

func (u *Upgrader) UpgradeSuperchainConfig(host *script.Host, input json.RawMessage) error {
	var upgradeInput UpgradeSuperchainConfigInput
	if err := json.Unmarshal(input, &upgradeInput); err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return UpgradeSuperchainConfig(host, upgradeInput)
}
