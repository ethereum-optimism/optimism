package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DeployAltDA2Input struct {
	Salt                     common.Hash
	ProxyAdmin               common.Address
	ChallengeContractOwner   common.Address
	ChallengeWindow          *big.Int
	ResolveWindow            *big.Int
	BondSize                 *big.Int
	ResolverRefundPercentage *big.Int
}

type DeployAltDA2Output struct {
	DataAvailabilityChallengeProxy common.Address
	DataAvailabilityChallengeImpl  common.Address
}

type DeployAltDA2Script script.DeployScriptWithOutput[DeployAltDA2Input, DeployAltDA2Output]

// NewDeployAltDAScript loads and validates the DeployAltDA2 script contract
func NewDeployAltDAScript(host *script.Host) (DeployAltDA2Script, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployAltDA2Input, DeployAltDA2Output](host, "DeployAltDA2.s.sol", "DeployAltDA2")
}
