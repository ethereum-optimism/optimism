package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

type DeployAlphabetVMInput struct {
	AbsolutePrestate common.Hash
	PreimageOracle   common.Address
}

type DeployAlphabetVMOutput struct {
	AlphabetVM common.Address
}

type DeployAlphabetVMScript script.DeployScriptWithOutput[DeployAlphabetVMInput, DeployAlphabetVMOutput]

// NewDeployAlphabetVMScript loads and validates the DeployAlphabetVM2 script contract
func NewDeployAlphabetVMScript(host *script.Host) (DeployAlphabetVMScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployAlphabetVMInput, DeployAlphabetVMOutput](host, "DeployAlphabetVM.s.sol", "DeployAlphabetVM")
}

func NewDeployAlphabetVMForgeCaller(client *forge.Client) forge.ScriptCaller[DeployAlphabetVMInput, DeployAlphabetVMOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployAlphabetVM.s.sol:DeployAlphabetVM",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployAlphabetVMInput]{TypeName: "DeployAlphabetVMInput"},
		&forge.BytesScriptDecoder[DeployAlphabetVMOutput]{TypeName: "DeployAlphabetVMOutput"},
	)
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
