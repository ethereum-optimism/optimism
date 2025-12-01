package opcm

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
)

type DeployDisputeGameInput struct {
	Release                  string
	UseV2                    bool
	GameKind                 string
	GameType                 uint32
	AbsolutePrestate         common.Hash
	MaxGameDepth             *big.Int
	SplitDepth               *big.Int
	ClockExtension           uint64
	MaxClockDuration         uint64
	DelayedWethProxy         common.Address
	AnchorStateRegistryProxy common.Address
	VmAddress                common.Address
	L2ChainId                *big.Int
	Proposer                 common.Address
	Challenger               common.Address
}

type DeployDisputeGameOutput struct {
	DisputeGameImpl common.Address
}

type DeployDisputeGameScript script.DeployScriptWithOutput[DeployDisputeGameInput, DeployDisputeGameOutput]

// NewDeployDisputeGameScript loads and validates the DeployDisputeGame2 script contract
func NewDeployDisputeGameScript(host *script.Host) (DeployDisputeGameScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DeployDisputeGameInput, DeployDisputeGameOutput](host, "DeployDisputeGame.s.sol", "DeployDisputeGame")
}
