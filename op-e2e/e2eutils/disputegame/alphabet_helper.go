package disputegame

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
)

type AlphabetGameHelper struct {
	FaultGameHelper
	claimedAlphabet string
}

func (g *AlphabetGameHelper) StartChallenger(ctx context.Context, l1Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
		challenger.WithAlphabet(g.claimedAlphabet),
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
