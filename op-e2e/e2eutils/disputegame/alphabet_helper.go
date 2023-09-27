package disputegame

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"

	"github.com/ethereum/go-ethereum/common"
)

type AlphabetGameHelper struct {
	FaultGameHelper
	claimedAlphabet string
}

func (g *AlphabetGameHelper) StartChallenger(ctx context.Context, l1Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		func(c *config.Config) {
			c.GameFactoryAddress = g.factoryAddr
			c.GameAllowlist = []common.Address{g.addr}
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

func (g *AlphabetGameHelper) CreateHonestActor(alphabetTrace string, depth uint64) *HonestHelper {
	return &HonestHelper{
		t:            g.t,
		require:      g.require,
		game:         &g.FaultGameHelper,
		correctTrace: alphabet.NewTraceProvider(alphabetTrace, depth),
	}
}

func (g *AlphabetGameHelper) CreateDishonestHelper(alphabetTrace string, depth uint64, defender bool) *DishonestHelper {
	return newDishonestHelper(&g.FaultGameHelper, g.CreateHonestActor(alphabetTrace, depth), defender)
}
