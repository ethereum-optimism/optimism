package opcm

import (
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
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

type DeployMIPSScript script.DeployScriptWithOutput[DeployMIPSInput, DeployMIPSOutput]

// NewDeployMIPSScript loads and validates the DeployMIPS script contract
func NewDeployMIPSScript(host *script.Host) (DeployMIPSScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployMIPSInput, DeployMIPSOutput](host, "DeployMIPS.s.sol", "DeployMIPS")
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
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/DeployMIPS.s.sol:DeployMIPS",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[DeployMIPSInput]{TypeName: "DeployMIPSInput"},
		&forge.BytesScriptDecoder[DeployMIPSOutput]{TypeName: "DeployMIPSOutput"},
	)
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
