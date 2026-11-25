package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

// DeploySuperchainInput must mirror the Solidity DeploySuperchain.s.sol input
// struct exactly — script ABI matching is checked at load time.
type DeploySuperchainInput struct {
	Guardian                  common.Address `toml:"guardian"`
	SuperchainProxyAdminOwner common.Address `toml:"superchainProxyAdminOwner"`
	Paused                    bool           `toml:"paused"`
}

// DeploySuperchainOutput must mirror DeploySuperchain.s.sol's output struct.
type DeploySuperchainOutput struct {
	SuperchainConfigImpl  common.Address `json:"superchainConfigImplAddress"`
	SuperchainConfigProxy common.Address `json:"superchainConfigProxyAddress"`
	SuperchainProxyAdmin  common.Address `json:"proxyAdminAddress"`
}

func NewDeploySuperchainForgeCaller(client *forge.Client) forge.ScriptCaller[DeploySuperchainInput, DeploySuperchainOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeploySuperchain.s.sol:DeploySuperchain",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeploySuperchainInput]{TypeName: "DeploySuperchainInput"},
		&forge.BytesScriptDecoder[DeploySuperchainOutput]{TypeName: "DeploySuperchainOutput"},
	)
}

// DeploySuperchain deploys superchain contracts using Forge
func DeploySuperchain(env *ForgeEnv, input DeploySuperchainInput) (DeploySuperchainOutput, error) {
	output, err := broadcastScript(env, NewDeploySuperchainForgeCaller, input)
	if err != nil {
		return output, fmt.Errorf("failed to deploy superchain with Forge: %w", err)
	}
	return output, nil
}
