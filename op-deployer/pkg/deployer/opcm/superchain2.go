package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type DeploySuperchain2Input struct {
	Guardian                   common.Address         `toml:"guardian"`
	ProtocolVersionsOwner      common.Address         `toml:"protocolVersionsOwner"`
	SuperchainProxyAdminOwner  common.Address         `toml:"superchainProxyAdminOwner"`
	Paused                     bool                   `toml:"paused"`
	RecommendedProtocolVersion params.ProtocolVersion `toml:"recommendedProtocolVersion"`
	RequiredProtocolVersion    params.ProtocolVersion `toml:"requiredProtocolVersion"`
}

type DeploySuperchain2Output struct {
	ProtocolVersionsImpl  common.Address `json:"protocolVersionsImplAddress"`
	ProtocolVersionsProxy common.Address `json:"protocolVersionsProxyAddress"`
	SuperchainConfigImpl  common.Address `json:"superchainConfigImplAddress"`
	SuperchainConfigProxy common.Address `json:"superchainConfigProxyAddress"`
	SuperchainProxyAdmin  common.Address `json:"proxyAdminAddress"`
}

type DeploySuperchainScript script.DeployScriptWithOutput[DeploySuperchain2Input, DeploySuperchain2Output]

// NewDeploySuperchainScript loads and validates the DeploySuperchain2 script contract
func NewDeploySuperchainScript(host *script.Host) (DeploySuperchainScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeploySuperchain2Input, DeploySuperchain2Output](host, "DeploySuperchain2.s.sol", "DeploySuperchain2")
}
