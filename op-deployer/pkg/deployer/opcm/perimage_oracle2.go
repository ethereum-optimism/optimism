package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployPreimageOracle2Input struct {
	MinProposalSize *big.Int
	ChallengePeriod *big.Int
}

type DeployPreimageOracle2Output struct {
	PreimageOracle common.Address
}

type DeployPreimageOracleScript script.DeployScriptWithOutput[DeployPreimageOracle2Input, DeployPreimageOracle2Output]

// NewDeployPreimageOracleScript loads and validates the DeployPreimageOracle2 script contract
func NewDeployPreimageOracleScript(host *script.Host) (DeployPreimageOracleScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployPreimageOracle2Input, DeployPreimageOracle2Output](host, "DeployPreimageOracle2.s.sol", "DeployPreimageOracle2")
}
