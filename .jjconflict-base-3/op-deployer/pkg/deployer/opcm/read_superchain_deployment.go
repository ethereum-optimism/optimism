package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type ReadSuperchainDeploymentInput struct {
	OPCMAddress           common.Address `abi:"opcmAddress"` // TODO(#18612): Remove OPCMAddress field when OPCMv1 gets deprecated
	SuperchainConfigProxy common.Address `abi:"superchainConfigProxy"`
}

type ReadSuperchainDeploymentOutput struct {
	// TODO(#18612): Remove ProtocolVersions fields when OPCMv1 gets deprecated
	ProtocolVersionsImpl       common.Address
	ProtocolVersionsProxy      common.Address
	ProtocolVersionsOwner      common.Address
	RecommendedProtocolVersion [32]byte
	RequiredProtocolVersion    [32]byte

	SuperchainConfigImpl      common.Address
	SuperchainConfigProxy     common.Address
	SuperchainProxyAdmin      common.Address
	Guardian                  common.Address
	SuperchainProxyAdminOwner common.Address
}

type ReadSuperchainDeploymentScript script.DeployScriptWithOutput[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput]

func NewReadSuperchainDeploymentScript(host *script.Host) (ReadSuperchainDeploymentScript, error) {
	return script.NewDeployScriptWithOutputFromFile[ReadSuperchainDeploymentInput, ReadSuperchainDeploymentOutput](host, "ReadSuperchainDeployment.s.sol", "ReadSuperchainDeployment")
}
