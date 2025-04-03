package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployAsterisc2Input struct {
	PreimageOracle common.Address
}

type DeployAsterisc2Output struct {
	AsteriscSingleton common.Address
}

type DeployAsteriscScript script.DeployScriptWithOutput[DeployAsterisc2Input, DeployAsterisc2Output]

// NewDeployAsteriscScript loads and validates the DeployAsterisc2 script contract
func NewDeployAsteriscScript(host *script.Host) (DeployAsteriscScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployAsterisc2Input, DeployAsterisc2Output](host, "DeployAsterisc2.s.sol", "DeployAsterisc2")
}
