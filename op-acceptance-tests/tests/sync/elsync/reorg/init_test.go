package reorg

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func reorgOpts() []presets.Option {
	return []presets.Option{
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
			cfg.NoDiscovery = true
		})),
	}
}

func newReorgSystem(t devtest.T) *presets.SingleChainMultiNodeWithTestSeq {
	return presets.NewSingleChainMultiNodeWithTestSeq(t, reorgOpts()...)
}
