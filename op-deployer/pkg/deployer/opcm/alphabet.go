package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

var deployAlphabetVMSpec = ScriptSpec{
	ScriptFile:      "DeployAlphabetVM.s.sol",
	ContractName:    "DeployAlphabetVM",
	ForgeScriptPath: "scripts/deploy/DeployAlphabetVM.s.sol:DeployAlphabetVM",
}

type DeployAlphabetVMInput = ScriptInput
type DeployAlphabetVMOutput = ScriptOutput

type DeployAlphabetVMScript = ScriptWithOutput

// NewDeployAlphabetVMScript loads and validates the DeployAlphabetVM2 script contract
func NewDeployAlphabetVMScript(host *script.Host) (DeployAlphabetVMScript, error) {
	return NewScriptWithOutputFromFile(host, deployAlphabetVMSpec)
}

func NewDeployAlphabetVMForgeCaller(client *forge.Client) forge.ScriptCaller[DeployAlphabetVMInput, DeployAlphabetVMOutput] {
	return NewScriptForgeCaller(client, deployAlphabetVMSpec)
}

// DeployAlphabetVMViaForge deploys Alphabet VM contracts using Forge
func DeployAlphabetVMViaForge(env *ForgeEnv, input DeployAlphabetVMInput) (DeployAlphabetVMOutput, error) {
	var output DeployAlphabetVMOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployAlphabetVMForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy Alphabet VM with Forge: %w", err)
	}
	return output, nil
}
