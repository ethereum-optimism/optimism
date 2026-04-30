package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

// DeploySuperchainInput must mirror the Solidity DeploySuperchain.s.sol input
// struct exactly — script ABI matching is checked at load time. The
// ProtocolVersions* fields are no longer meaningful (the contract is dead in
// op-node since #20258) but stay until PR 2 of #20309 strips them from the
// Solidity script. Versions use `common.Hash` (= `bytes32`) to avoid the
// op-geth `params.ProtocolVersion` import that #20266 needs gone.
type DeploySuperchainInput struct {
	Guardian                   common.Address `toml:"guardian"`
	ProtocolVersionsOwner      common.Address `toml:"protocolVersionsOwner"`
	SuperchainProxyAdminOwner  common.Address `toml:"superchainProxyAdminOwner"`
	Paused                     bool           `toml:"paused"`
	RecommendedProtocolVersion common.Hash    `toml:"recommendedProtocolVersion"`
	RequiredProtocolVersion    common.Hash    `toml:"requiredProtocolVersion"`
}

// DeploySuperchainOutput must mirror DeploySuperchain.s.sol's output struct.
// The ProtocolVersions* addresses are no longer read by callers, but stay
// here until PR 2 of #20309 strips them from Solidity.
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
