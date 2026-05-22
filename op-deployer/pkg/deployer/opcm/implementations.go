package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

var deployImplementationsSpec = ScriptSpec{
	ScriptFile:      "DeployImplementations.s.sol",
	ContractName:    "DeployImplementations",
	ForgeScriptPath: "scripts/deploy/DeployImplementations.s.sol:DeployImplementations",
}

type DeployImplementationsInput = ScriptInput
type DeployImplementationsOutput = ScriptOutput

type DeployImplementationsScript = ScriptWithOutput

// NewDeployImplementationsScript loads and validates the DeployImplementations script contract
func NewDeployImplementationsScript(host *script.Host) (DeployImplementationsScript, error) {
	return NewScriptWithOutputFromFile(host, deployImplementationsSpec)
}

func NewDeployImplementationsForgeCaller(client *forge.Client) forge.ScriptCaller[DeployImplementationsInput, DeployImplementationsOutput] {
	return NewScriptForgeCaller(client, deployImplementationsSpec)
}

// DeployImplementationsViaForge deploys implementation contracts using Forge
func DeployImplementationsViaForge(env *ForgeEnv, input DeployImplementationsInput) (DeployImplementationsOutput, error) {
	var output DeployImplementationsOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployImplementationsForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy implementations with Forge: %w", err)
	}
	return output, nil
}
