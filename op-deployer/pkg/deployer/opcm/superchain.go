package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
)

type DeploySuperchainInput struct {
	Guardian                  common.Address `toml:"guardian"`
	SuperchainProxyAdminOwner common.Address `toml:"superchainProxyAdminOwner"`
	Paused                    bool           `toml:"paused"`
}

type DeploySuperchainOutput struct {
	SuperchainConfigImpl  common.Address `json:"superchainConfigImplAddress"`
	SuperchainConfigProxy common.Address `json:"superchainConfigProxyAddress"`
	SuperchainProxyAdmin  common.Address `json:"proxyAdminAddress"`
}

type DeploySuperchainScript script.DeployScriptWithOutput[DeploySuperchainInput, DeploySuperchainOutput]

// NewDeploySuperchainScript loads and validates the DeploySuperchain script contract
func NewDeploySuperchainScript(host *script.Host) (DeploySuperchainScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeploySuperchainInput, DeploySuperchainOutput](host, "DeploySuperchain.s.sol", "DeploySuperchain")
}
