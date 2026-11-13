package follow_l2

import (
	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

func newSingleChainTwoVerifiersFollowL2(t devtest.T, opts ...presets.Option) *presets.SingleChainTwoVerifiers {
	return presets.NewSingleChainTwoVerifiersWithoutCheck(t, append([]presets.Option{
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
			func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
				cfg.NoDiscovery = true
			})),
	}, opts...)...)
}
