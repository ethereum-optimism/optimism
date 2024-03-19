package disputegame

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
)

type AlphabetGameHelper struct {
	FaultGameHelper
}

func (g *AlphabetGameHelper) StartChallenger(ctx context.Context, sys challenger.EndpointProvider, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
		challenger.WithAlphabet(g.system.RollupEndpoint("sequencer")),
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, sys, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

func (g *AlphabetGameHelper) CreateHonestActor(alphabetTrace string, depth types.Depth) *HonestHelper {
	return &HonestHelper{
		t:            g.t,
		require:      g.require,
		game:         &g.FaultGameHelper,
		correctTrace: alphabet.NewTraceProvider(big.NewInt(0), depth),
	}
}
