package opcm

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

type DeployMIPSInput struct {
	PreimageOracle common.Address
	MipsVersion    *big.Int
}

type DeployMIPSOutput struct {
	MipsSingleton common.Address
}

func NewDeployMIPSForgeCaller(client *forge.Client) forge.ScriptCaller[DeployMIPSInput, DeployMIPSOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployMIPS.s.sol:DeployMIPS",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployMIPSInput]{TypeName: "DeployMIPSInput"},
		&forge.BytesScriptDecoder[DeployMIPSOutput]{TypeName: "DeployMIPSOutput"},
	)
}

// DeployMIPS deploys MIPS contracts using Forge
func DeployMIPS(env *ForgeEnv, input DeployMIPSInput) (DeployMIPSOutput, error) {
	output, err := broadcastScript(env, NewDeployMIPSForgeCaller, input)
	if err != nil {
		return output, fmt.Errorf("failed to deploy MIPS VM with Forge: %w", err)
	}
	return output, nil
}
