package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

var deploySuperchainSpec = ScriptSpec{
	ScriptFile:      "DeploySuperchain.s.sol",
	ContractName:    "DeploySuperchain",
	ForgeScriptPath: "scripts/deploy/DeploySuperchain.s.sol:DeploySuperchain",
}

type DeploySuperchainInput = ScriptInput
type DeploySuperchainOutput = ScriptOutput

type DeploySuperchainScript = ScriptWithOutput

// NewDeploySuperchainScript loads and validates the DeploySuperchain script contract
func NewDeploySuperchainScript(host *script.Host) (DeploySuperchainScript, error) {
	return NewScriptWithOutputFromFile(host, deploySuperchainSpec)
}

func NewDeploySuperchainForgeCaller(client *forge.Client) forge.ScriptCaller[DeploySuperchainInput, DeploySuperchainOutput] {
	return NewScriptForgeCaller(client, deploySuperchainSpec)
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
