package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type ReadSuperchainDeploymentInput struct {
	OPCMAddress common.Address `abi:"opcmAddress"`
}

type ReadSuperchainDeploymentOutput struct {
	SuperchainConfigImpl  common.Address
	SuperchainConfigProxy common.Address
	SuperchainProxyAdmin  common.Address

	Guardian                  common.Address
	SuperchainProxyAdminOwner common.Address
}

type ReadSuperchainDeploymentScript script.DeployScriptWithOutput[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput]

func NewReadSuperchainDeploymentScript(host *script.Host) (ReadSuperchainDeploymentScript, error) {
	return script.NewDeployScriptWithOutputFromFile[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput](host, "ReadSuperchainDeployment.s.sol", "ReadSuperchainDeployment")
}
