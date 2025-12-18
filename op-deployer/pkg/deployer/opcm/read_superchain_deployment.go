package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type ReadSuperchainDeploymentInput struct {
	OPCMAddress common.Address `abi:"opcmAddress"`
}

type ReadSuperchainDeploymentOutput struct {
	ProtocolVersionsImpl  common.Address
	ProtocolVersionsProxy common.Address
	SuperchainConfigImpl  common.Address
	SuperchainConfigProxy common.Address
	SuperchainProxyAdmin  common.Address

	Guardian                   common.Address
	ProtocolVersionsOwner      common.Address
	SuperchainProxyAdminOwner  common.Address
	RecommendedProtocolVersion [32]byte
	RequiredProtocolVersion    [32]byte
}

type ReadSuperchainDeploymentScript script.DeployScriptWithOutput[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput]

func NewReadSuperchainDeploymentScript(host *script.Host) (ReadSuperchainDeploymentScript, error) {
	return script.NewDeployScriptWithOutputFromFile[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput](host, "ReadSuperchainDeployment.s.sol", "ReadSuperchainDeployment")
}
