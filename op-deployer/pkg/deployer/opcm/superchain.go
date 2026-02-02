package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type DeploySuperchainInput struct {
	Guardian                   common.Address         `toml:"guardian"`
	ProtocolVersionsOwner      common.Address         `toml:"protocolVersionsOwner"`
	SuperchainProxyAdminOwner  common.Address         `toml:"superchainProxyAdminOwner"`
	Paused                     bool                   `toml:"paused"`
	RecommendedProtocolVersion params.ProtocolVersion `toml:"recommendedProtocolVersion"`
	RequiredProtocolVersion    params.ProtocolVersion `toml:"requiredProtocolVersion"`
}

type DeploySuperchainOutput struct {
	ProtocolVersionsImpl  common.Address `json:"protocolVersionsImplAddress"`
	ProtocolVersionsProxy common.Address `json:"protocolVersionsProxyAddress"`
	SuperchainConfigImpl  common.Address `json:"superchainConfigImplAddress"`
	SuperchainConfigProxy common.Address `json:"superchainConfigProxyAddress"`
	SuperchainProxyAdmin  common.Address `json:"proxyAdminAddress"`
}

type DeploySuperchainScript script.DeployScriptWithOutput[DeploySuperchainInput, DeploySuperchainOutput]

// NewDeploySuperchainScript loads and validates the DeploySuperchain script contract
func NewDeploySuperchainScript(host *script.Host) (DeploySuperchainScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeploySuperchainInput, DeploySuperchainOutput](host, "DeploySuperchain.s.sol", "DeploySuperchain")
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

// DeploySuperchainViaForge deploys superchain contracts using Forge
func DeploySuperchainViaForge(env *ForgeEnv, input DeploySuperchainInput) (DeploySuperchainOutput, error) {
	var output DeploySuperchainOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeploySuperchainForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy superchain with Forge: %w", err)
	}
	return output, nil
}
