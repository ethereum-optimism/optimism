package opcm

import (
	"fmt"

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

func NewDeployAlphabetVMForgeCaller(client *forge.Client) forge.ScriptCaller[DeployAlphabetVMInput, DeployAlphabetVMOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployAlphabetVM.s.sol:DeployAlphabetVM",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployAlphabetVMInput]{TypeName: "DeployAlphabetVMInput"},
		&forge.BytesScriptDecoder[DeployAlphabetVMOutput]{TypeName: "DeployAlphabetVMOutput"},
	)
}

// DeployAlphabetVM deploys Alphabet VM contracts using Forge
func DeployAlphabetVM(env *ForgeEnv, input DeployAlphabetVMInput) (DeployAlphabetVMOutput, error) {
	output, err := broadcastScript(env, NewDeployAlphabetVMForgeCaller, input)
	if err != nil {
		return output, fmt.Errorf("failed to deploy Alphabet VM with Forge: %w", err)
	}
	return output, nil
}
