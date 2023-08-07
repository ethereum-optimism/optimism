package disputegame

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
)

type AlphabetGameHelper struct {
	FaultGameHelper
	claimedAlphabet string
}

func (g *AlphabetGameHelper) StartChallenger(ctx context.Context, l1Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		func(c *config.Config) {
			c.GameAddress = g.addr
			c.GameDepth = alphabetGameDepth
			c.TraceType = config.TraceTypeAlphabet
			// By default the challenger agrees with the root claim (thus disagrees with the proposed output)
			// This can be overridden by passing in options
			c.AlphabetTrace = g.claimedAlphabet
			c.AgreeWithProposedOutput = false
		},
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, l1Endpoint, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}
