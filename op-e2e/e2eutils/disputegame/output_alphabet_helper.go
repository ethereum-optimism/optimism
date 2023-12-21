package disputegame

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
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

func (g *OutputAlphabetGameHelper) CreateHonestActor(ctx context.Context, alphabetTrace string, depth uint64, l2Node string) *OutputHonestHelper {
	logger := testlog.Logger(g.t, log.LvlInfo).New("role", "HonestHelper", "game", g.addr)
	caller := batching.NewMultiCaller(g.system.NodeClient("l1").Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewOutputBisectionGameContract(g.addr, caller)
	g.require.NoError(err, "Failed to create game contact")

	prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	splitDepth := uint64(g.SplitDepth(ctx))
	rollupClient := g.system.RollupClient(l2Node)
	prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
	accessor, err := outputs.NewOutputAlphabetTraceAccessor(
		logger, metrics.NoopMetrics, prestateProvider, rollupClient, splitDepth, prestateBlock, poststateBlock)
	g.require.NoError(err, "Failed to create output cannon trace accessor")
	return &OutputHonestHelper{
		t:            g.t,
		require:      g.require,
		game:         &g.OutputGameHelper,
		contract:     contract,
		correctTrace: accessor,
	}
}

func (g *OutputAlphabetGameHelper) CreateDishonestHelper(ctx context.Context, alphabetTrace string, depth uint64, l2Node string, defender bool) *DishonestHelper {
	return newDishonestHelper(g.t, &g.OutputGameHelper, g.CreateHonestActor(ctx, alphabetTrace, depth, l2Node), defender)
}
