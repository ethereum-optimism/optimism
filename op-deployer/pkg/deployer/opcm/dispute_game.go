package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
)

var deployDisputeGameSpec = ScriptSpec{
	ScriptFile:      "DeployDisputeGame.s.sol",
	ContractName:    "DeployDisputeGame",
	ForgeScriptPath: "scripts/deploy/DeployDisputeGame.s.sol:DeployDisputeGame",
}

type DeployDisputeGameInput = ScriptInput
type DeployDisputeGameOutput = ScriptOutput

type DeployDisputeGameScript = ScriptWithOutput

// NewDeployDisputeGameScript loads and validates the DeployDisputeGame2 script contract
func NewDeployDisputeGameScript(host *script.Host) (DeployDisputeGameScript, error) {
	return NewScriptWithOutputFromFile(host, deployDisputeGameSpec)
}

func NewDeployDisputeGameForgeCaller(client *forge.Client) forge.ScriptCaller[DeployDisputeGameInput, DeployDisputeGameOutput] {
	return NewScriptForgeCaller(client, deployDisputeGameSpec)
}

// DeployDisputeGameViaForge deploys dispute game contracts using Forge
func DeployDisputeGameViaForge(env *ForgeEnv, input DeployDisputeGameInput) (DeployDisputeGameOutput, error) {
	var output DeployDisputeGameOutput
	if err := env.validate(true); err != nil {
		return output, err
	}
	forgeCaller := NewDeployDisputeGameForgeCaller(env.Client)
	var err error
	output, _, err = forgeCaller(env.Context, input, env.buildForgeOpts()...)
	if err != nil {
		return output, fmt.Errorf("failed to deploy dispute game with Forge: %w", err)
	}
	return output, nil
}
