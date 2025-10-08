package v5_0_0

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/opcm"
	"github.com/ethereum/go-ethereum/common"
)

type UpgradeSuperchainConfigInput struct {
	Prank                common.Address `json:"prank"`
	Opcm                 common.Address `json:"opcm"`
	SuperchainConfig     common.Address `json:"superchainConfig"`
	SuperchainProxyAdmin common.Address `json:"superchainProxyAdmin"`
}

type UpgradeSuperchainConfigScript struct {
	Run func(input common.Address)
}

func UpgradeSuperchainConfig(host *script.Host, input UpgradeSuperchainConfigInput) error {
	return opcm.RunScriptVoid[UpgradeSuperchainConfigInput](host, input, "UpgradeSuperchainConfig.s.sol", "UpgradeSuperchainConfig")
}

func (u *Upgrader) UpgradeSuperchainConfig(host *script.Host, input json.RawMessage) error {
	var upgradeInput UpgradeSuperchainConfigInput
	if err := json.Unmarshal(input, &upgradeInput); err != nil {
		return fmt.Errorf("failed to unmarshal input: %w", err)
	}
	return UpgradeSuperchainConfig(host, upgradeInput)
}
