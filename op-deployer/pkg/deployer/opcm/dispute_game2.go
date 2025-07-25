package opcm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

type DeployDisputeGame2Input struct {
	Release                  string
	StandardVersionsToml     string
	GameKind                 string
	GameType                 *big.Int
	AbsolutePrestate         common.Hash
	MaxGameDepth             *big.Int
	SplitDepth               *big.Int
	ClockExtension           *big.Int
	MaxClockDuration         *big.Int
	DelayedWethProxy         common.Address
	AnchorStateRegistryProxy common.Address
	VmAddress                common.Address `abi:"vm"`
	L2ChainId                *big.Int
	Proposer                 common.Address
	Challenger               common.Address
}

type DeployDisputeGame2Output struct {
	DisputeGameImpl common.Address
}

type DeployDisputeGameScript script.DeployScriptWithOutput[DeployDisputeGame2Input, DeployDisputeGame2Output]

// NewDeployDisputeGameScript loads and validates the DeployDisputeGame2 script contract
func NewDeployDisputeGameScript(host *script.Host) (DeployDisputeGameScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployDisputeGame2Input, DeployDisputeGame2Output](host, "DeployDisputeGame2.s.sol", "DeployDisputeGame2")
}
