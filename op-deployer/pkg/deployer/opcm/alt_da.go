package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

var deployAltDASpec = ScriptSpec{
	ScriptFile:      "DeployAltDA.s.sol",
	ContractName:    "DeployAltDA",
	ForgeScriptPath: "scripts/deploy/DeployAltDA.s.sol:DeployAltDA",
}

type DeployAltDAInput = ScriptInput
type DeployAltDAOutput = ScriptOutput

type DeployAltDAScript = ScriptWithOutput

// NewDeployAltDAScript loads and validates the DeployAltDA script contract
func NewDeployAltDAScript(host *script.Host) (DeployAltDAScript, error) {
	return NewScriptWithOutputFromFile(host, deployAltDASpec)
}

func NewDeployAltDAForgeCaller(client *forge.Client) forge.ScriptCaller[DeployAltDAInput, DeployAltDAOutput] {
	return NewScriptForgeCaller(client, deployAltDASpec)
}

// DeployAltDAViaForge deploys AltDA contracts using Forge
func DeployAltDAViaForge(env *ForgeEnv, input DeployAltDAInput) (DeployAltDAOutput, error) {
	var output DeployAltDAOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployAltDAForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy alt-da contracts with Forge: %w", err)
	}
	return output, nil
}
