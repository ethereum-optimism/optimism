package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

var readSuperchainDeploymentSpec = ScriptSpec{
	ScriptFile:      "ReadSuperchainDeployment.s.sol",
	ContractName:    "ReadSuperchainDeployment",
	ForgeScriptPath: "scripts/deploy/ReadSuperchainDeployment.s.sol:ReadSuperchainDeployment",
}

type ReadSuperchainDeploymentInput = ScriptInput
type ReadSuperchainDeploymentOutput = ScriptOutput

type ReadSuperchainDeploymentScript = ScriptWithOutput

func NewReadSuperchainDeploymentScript(host *script.Host) (ReadSuperchainDeploymentScript, error) {
	return NewScriptWithOutputFromFile(host, readSuperchainDeploymentSpec)
}

func NewReadSuperchainDeploymentForgeCaller(client *forge.Client) forge.ScriptCaller[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput] {
	return NewScriptForgeCaller(client, readSuperchainDeploymentSpec)
}

// ReadSuperchainDeploymentViaForge reads superchain deployment addresses using Forge
func ReadSuperchainDeploymentViaForge(env *ForgeEnv, input ReadSuperchainDeploymentInput) (ReadSuperchainDeploymentOutput, error) {
	var output ReadSuperchainDeploymentOutput
	if err := env.validate(false); err != nil {
		return output, err
	}
	forgeCaller := NewReadSuperchainDeploymentForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOptsReadOnly()...)
	if err != nil {
		return output, fmt.Errorf("failed to run ReadSuperchainDeployment with Forge: %w", err)
	}
	return output, nil
}
