package opcm

import (
	"fmt"

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

func NewReadSuperchainDeploymentForgeCaller(client *forge.Client) forge.ScriptCaller[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput] {
	return forge.NewScriptCaller(
		client,
		"scripts/deploy/ReadSuperchainDeployment.s.sol:ReadSuperchainDeployment",
		"runWithBytes(bytes)",
		&forge.BytesScriptEncoder[ReadSuperchainDeploymentInput]{TypeName: "ReadSuperchainDeploymentInput"},
		&forge.BytesScriptDecoder[ReadSuperchainDeploymentOutput]{TypeName: "ReadSuperchainDeploymentOutput"},
	)
}

// ReadSuperchainDeployment reads superchain deployment addresses using Forge
func ReadSuperchainDeployment(env *ForgeEnv, input ReadSuperchainDeploymentInput) (ReadSuperchainDeploymentOutput, error) {
	output, err := readScript(env, NewReadSuperchainDeploymentForgeCaller, input)
	if err != nil {
		return output, fmt.Errorf("failed to run ReadSuperchainDeployment with Forge: %w", err)
	}
	return output, nil
}
