package disputegame

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
)

type OutputAlphabetGameHelper struct {
	OutputGameHelper
	claimedAlphabet string
}

func (g *OutputAlphabetGameHelper) StartChallenger(
	ctx context.Context,
	l2Node string,
	name string,
	options ...challenger.Option,
) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithOutputAlphabet(g.claimedAlphabet, g.system.RollupEndpoint(l2Node)),
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
