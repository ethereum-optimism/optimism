package disputegame

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
)

type OutputCannonGameHelper struct {
	OutputGameHelper
}

func (g *OutputCannonGameHelper) StartChallenger(
	ctx context.Context,
	l2Node string,
	name string,
	options ...challenger.Option,
) *challenger.Helper {
	rollupEndpoint := g.system.RollupEndpoint(l2Node)
	l2Endpoint := g.system.NodeEndpoint(l2Node)
	opts := []challenger.Option{
		challenger.WithOutputCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), rollupEndpoint, l2Endpoint),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, g.system.NodeEndpoint("l1"), name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}
