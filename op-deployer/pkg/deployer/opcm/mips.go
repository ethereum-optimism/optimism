package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

var deployMIPSSpec = ScriptSpec{
	ScriptFile:      "DeployMIPS.s.sol",
	ContractName:    "DeployMIPS",
	ForgeScriptPath: "scripts/deploy/DeployMIPS.s.sol:DeployMIPS",
}

type DeployMIPSInput = ScriptInput
type DeployMIPSOutput = ScriptOutput

type DeployMIPSScript = ScriptWithOutput

// NewDeployMIPSScript loads and validates the DeployMIPS script contract
func NewDeployMIPSScript(host *script.Host) (DeployMIPSScript, error) {
	return NewScriptWithOutputFromFile(host, deployMIPSSpec)
}

func DeployMIPS(
	host *script.Host,
	input DeployMIPSInput,
) (DeployMIPSOutput, error) {
	deployScript, err := NewDeployMIPSScript(host)
	if err != nil {
		var zero DeployMIPSOutput
		return zero, err
	}
	return deployScript.Run(input)
}

func NewDeployMIPSForgeCaller(client *forge.Client) forge.ScriptCaller[DeployMIPSInput, DeployMIPSOutput] {
	return NewScriptForgeCaller(client, deployMIPSSpec)
}

// DeployMIPSViaForge deploys MIPS contracts using Forge
func DeployMIPSViaForge(env *ForgeEnv, input DeployMIPSInput) (DeployMIPSOutput, error) {
	var output DeployMIPSOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployMIPSForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy MIPS VM with Forge: %w", err)
	}
	return output, nil
}
