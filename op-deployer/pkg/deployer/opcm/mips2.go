package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployMIPS2Input struct {
	PreimageOracle common.Address
	MipsVersion    *big.Int
}

type DeployMIPS2Output struct {
	MipsSingleton common.Address
}

type DeployMIPSScript script.DeployScriptWithOutput[DeployMIPS2Input, DeployMIPS2Output]

// NewDeployMIPSScript loads and validates the DeployMIPS2 script contract
func NewDeployMIPSScript(host *script.Host) (DeployMIPSScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployMIPS2Input, DeployMIPS2Output](host, "DeployMIPS2.s.sol", "DeployMIPS2")
}
