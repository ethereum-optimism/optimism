package opcm

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum-optimism/optimism/op-deployer/pkg/deployer/forge"
	"github.com/ethereum/go-ethereum/common"
)

type ReadSuperchainDeploymentInput struct {
	SuperchainConfigProxy common.Address
}

type ReadSuperchainDeploymentOutput struct {
	SuperchainConfigImpl      common.Address `abi:"superchainConfigImpl"`
	SuperchainConfigProxy     common.Address `abi:"superchainConfigProxy"`
	SuperchainProxyAdmin      common.Address `abi:"superchainProxyAdmin"`
	Guardian                  common.Address `abi:"guardian"`
	SuperchainProxyAdminOwner common.Address `abi:"superchainProxyAdminOwner"`
}

type ReadSuperchainDeploymentScript script.DeployScriptWithOutput[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput]

func NewReadSuperchainDeploymentScript(host *script.Host) (ReadSuperchainDeploymentScript, error) {
	return script.NewDeployScriptWithOutputFromFile[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput](host, "ReadSuperchainDeployment.s.sol", "ReadSuperchainDeployment")
}

func NewReadSuperchainDeploymentForgeCaller(client *forge.Client) forge.ScriptCaller[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/ReadSuperchainDeployment.s.sol:ReadSuperchainDeployment",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[ReadSuperchainDeploymentInput]{TypeName: "ReadSuperchainDeploymentInput"},
		&forge.BytesScriptDecoder[ReadSuperchainDeploymentOutput]{TypeName: "ReadSuperchainDeploymentOutput"},
	)
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
