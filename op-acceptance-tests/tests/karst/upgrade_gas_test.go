package karst

import (
	"testing"

	opforks "github.com/ethereum-optimism/optimism/op-core/forks"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// karstUpgradeGasOffset schedules Karst a few blocks after L2 genesis so the test can
// observe the pre-activation, activation, and post-activation blocks within a short window.
const karstUpgradeGasOffset uint64 = 10

// TestKarstUpgradeGas checks the one-time "upgrade gas" added to the Karst activation block's
// gas limit. By default it must not leak onto later blocks (the next block reverts to the
// steady state); with the keep_karst_upgrade_gas opt-out the chain deliberately keeps the
// inflated limit instead.
func TestKarstUpgradeGas(gt *testing.T) {
	gt.Run("reverts", func(gt *testing.T) { testKarstUpgradeGas(gt, false) })
	gt.Run("kept", func(gt *testing.T) { testKarstUpgradeGas(gt, true) })
}

// testKarstUpgradeGas activates Karst mid-chain, inspects the gas limit across the activation
// boundary, and proves the span with kona-client. The activation block carries a one-time
// upgrade-gas bump for the NUT bundle, so it must exceed the pre-activation gas limit. The block
// after activation reverts to the steady state, unless keep is set — the chain opted out via
// keep_karst_upgrade_gas — in which case it keeps the inflated activation-block limit.
func testKarstUpgradeGas(gt *testing.T, keep bool) {
	t := devtest.ParallelT(gt)
	offset := karstUpgradeGasOffset

	deployOpts := []sysgo.DeployerOption{sysgo.WithKarstAtOffset(&offset)}
	if keep {
		deployOpts = append(deployOpts, sysgo.WithKeepKarstUpgradeGas)
	}
	sys := presets.NewMinimalWithKona(t, presets.WithDeployerOptions(deployOpts...))

	// Karst must activate mid-chain so a pre-activation block exists.
	activationBlock := sys.L2Chain.AwaitActivation(t, opforks.Karst).Number
	t.Require().Greater(activationBlock, uint64(1),
		"Karst must activate mid-chain so a pre-activation block exists")

	postBlock := activationBlock + 1
	sys.L2EL.WaitForBlockNumber(postBlock)
	pre := sys.L2EL.GasLimitAtBlock(activationBlock - 1)
	activation := sys.L2EL.GasLimitAtBlock(activationBlock)
	post := sys.L2EL.GasLimitAtBlock(postBlock)
	t.Logger().Info("Karst activation gas limits", "activationBlock", activationBlock,
		"pre", pre, "activation", activation, "post", post)

	t.Require().Greater(activation, pre,
		"Karst activation block must carry the one-time upgrade gas above the pre-activation gas limit")
	if keep {
		t.Require().Equal(activation, post,
			"with keep_karst_upgrade_gas set, the post-activation gas limit must keep the inflated activation-block limit")
	} else {
		t.Require().Equal(pre, post,
			"post-activation gas limit must revert to the pre-activation steady state")
	}

	// Prove the boundary with kona-client too: agree on the pre-activation block and prove
	// forward across the activation into the post-activation block. kona-host cannot prefetch
	t.Require().True(sys.RunKonaNative(activationBlock-1, postBlock))
}
