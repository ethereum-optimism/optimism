package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployAsteriscInput struct {
	PreimageOracle common.Address
}

type DeployAsteriscOutput struct {
	AsteriscSingleton common.Address
}

type DeployAsteriscScript script.DeployScriptWithOutput[DeployAsteriscInput, DeployAsteriscOutput]

// NewDeployAsteriscScript loads and validates the DeployAsterisc script contract
func NewDeployAsteriscScript(host *script.Host) (DeployAsteriscScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployAsteriscInput, DeployAsteriscOutput](host, "DeployAsterisc.s.sol", "DeployAsterisc")
}
