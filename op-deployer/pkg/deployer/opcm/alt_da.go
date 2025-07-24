package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployAltDAInput struct {
	Salt                     common.Hash
	ProxyAdmin               common.Address
	ChallengeContractOwner   common.Address
	ChallengeWindow          *big.Int
	ResolveWindow            *big.Int
	BondSize                 *big.Int
	ResolverRefundPercentage *big.Int
}

type DeployAltDAOutput struct {
	DataAvailabilityChallengeProxy common.Address
	DataAvailabilityChallengeImpl  common.Address
}

type DeployAltDAScript script.DeployScriptWithOutput[DeployAltDAInput, DeployAltDAOutput]

// NewDeployAltDAScript loads and validates the DeployAltDA script contract
func NewDeployAltDAScript(host *script.Host) (DeployAltDAScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployAltDAInput, DeployAltDAOutput](host, "DeployAltDA.s.sol", "DeployAltDA")
}
