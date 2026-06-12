package karst

import (
	"testing"

	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// TestKarstUpgradeBlockElevatesGasLimit demonstrates that when Karst activates
// mid-chain, the activation block carries an elevated gas limit. op-node injects the
// Karst NUT bundle (the L2ContractsManager upgrade transactions) as deposits at that
// block and raises the block's gas limit by the bundle's gas so the upgrades run
// outside the regular per-block gas budget. The bump lasts exactly one block.
func TestKarstUpgradeBlockElevatesGasLimit(gt *testing.T) {
	t := devtest.ParallelT(gt)
	offset := uint64(10) // activate Karst a few blocks after genesis
	sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithKarstAtOffset(&offset)))

	activation := sys.L2Chain.AwaitActivation(t, opforks.Karst)

	sys.L2EL.VerifyActivationGasLimitBump(activation.Number)
}

// TestKarstAtGenesisKeepsUniformGasLimit demonstrates the contrast: when Karst is
// active at genesis there is no activation block, so no upgrade transactions are
// injected and no block receives the one-time gas-limit bump. The gas limit stays
// uniform across the early chain.
func TestKarstAtGenesisKeepsUniformGasLimit(gt *testing.T) {
	t := devtest.ParallelT(gt)
	sys := presets.NewMinimal(t, presets.WithDeployerOptions(sysgo.WithKarstAtGenesis))

	const tailBlock = 5 // a handful of post-genesis blocks is enough to show no bump occurs
	sys.L2EL.VerifyUniformGasLimit(0, tailBlock)
}
