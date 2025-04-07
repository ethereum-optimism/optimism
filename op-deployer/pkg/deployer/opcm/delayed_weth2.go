package opcm

import (
	"math/big"

	"github.com/ethereum-optimism/optimism/op-chain-ops/script"
	"github.com/ethereum/go-ethereum/common"
)

type DelayedWETH2Input struct {
	Release               string
	ProxyAdmin            common.Address
	SuperchainConfigProxy common.Address
	DelayedWethImpl       common.Address
	DelayedWethOwner      common.Address
	DelayedWethDelay      *big.Int
}

type DelayedWETH2Output struct {
	DelayedWethImpl  common.Address
	DelayedWethProxy common.Address
}

type DelayedWETHScript script.DeployScriptWithOutput[DelayedWETH2Input, DelayedWETH2Output]

// NewDelayedWETHScript loads and validates the DelayedWETH2 script contract
func NewDelayedWETHScript(host *script.Host) (DelayedWETHScript, error) {
	return script.NewDeployScriptWithOutputFromFile[DelayedWETH2Input, DelayedWETH2Output](host, "DeployDelayedWETH2.s.sol", "DeployDelayedWETH2")
}
