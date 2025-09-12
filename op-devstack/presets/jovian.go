package presets

import (
	"github.com/ethereum-optimism/optimism/op-chain-ops/devkeys"
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/intentbuilder"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// WithJovianAtGenesis configures all L2s to activate the Jovian fork at genesis in sysgo mode.
func WithJovianAtGenesis() stack.CommonOption {
	return stack.MakeCommon(sysgo.WithDeployerOptions(
		func(p devtest.P, _ devkeys.Keys, builder intentbuilder.Builder) {
			for _, l2Cfg := range builder.L2s() {
				l2Cfg.WithForkAtGenesis(rollup.Jovian)
			}
		},
	))
}
