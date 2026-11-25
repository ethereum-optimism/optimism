package follow_l2

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/presets"
	"github.com/ethereum-optimism/optimism/op-devstack/sysgo"
)

// followSource is the CL kind of the sequencer L2CL and of L2CLB, the deriving verifier L2CLC
// follows. L2CLC runs the kind DEVSTACK_L2CL_KIND selects.
type followSource sysgo.MixedL2CLKind

func (s followSource) String() string {
	return "follow-source=" + string(s)
}

// forEachFollowSource runs fn twice in parallel: with the follow source on L2CLC's CL kind, and
// on the other CL, so each CL job also checks that its CL follows the other one.
func forEachFollowSource(gt *testing.T, fn func(t devtest.T, source followSource)) {
	t := devtest.ParallelT(gt)
	follower := sysgo.ResolveMixedL2CLKind(t)
	other := sysgo.MixedL2CLKona
	if follower == sysgo.MixedL2CLKona {
		other = sysgo.MixedL2CLOpNode
	}
	devtest.RunParallel(t, []followSource{followSource(follower), followSource(other)}, fn)
}

func newSingleChainTwoVerifiersFollowL2(t devtest.T, source followSource, opts ...presets.Option) *presets.SingleChainTwoVerifiers {
	return presets.NewSingleChainTwoVerifiersWithoutCheck(t, append([]presets.Option{
		presets.WithFollowSourceCLKind(sysgo.MixedL2CLKind(source)),
		presets.WithGlobalL2CLOption(sysgo.L2CLOptionFn(
			func(_ devtest.T, _ sysgo.ComponentTarget, cfg *sysgo.L2CLConfig) {
				cfg.NoDiscovery = true
			})),
	}, opts...)...)
}
