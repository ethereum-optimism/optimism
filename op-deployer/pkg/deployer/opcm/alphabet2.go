package opcm

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployAlphabetVM2Input struct {
	AbsolutePrestate common.Hash
	PreimageOracle   common.Address
}

type DeployAlphabetVM2Output struct {
	AlphabetVM common.Address
}

type DeployAlphabetVMScript script.DeployScriptWithOutput[DeployAlphabetVM2Input, DeployAlphabetVM2Output]

// NewDeployAlphabetVMScript loads and validates the DeployAlphabetVM2 script contract
func NewDeployAlphabetVMScript(host *script.Host) (DeployAlphabetVMScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployAlphabetVM2Input, DeployAlphabetVM2Output](host, "DeployAlphabetVM2.s.sol", "DeployAlphabetVM2")
}
