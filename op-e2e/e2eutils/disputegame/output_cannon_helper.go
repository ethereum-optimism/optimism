package disputegame

import (
	"context"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
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

func (g *OutputCannonGameHelper) CreateHonestActor(ctx context.Context, l2Node string, options ...challenger.Option) *OutputHonestHelper {
	opts := []challenger.Option{
		challenger.WithOutputCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), g.system.RollupEndpoint(l2Node), g.system.NodeEndpoint(l2Node)),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
	opts = append(opts, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system.NodeEndpoint("l1"), opts...)

	logger := testlog.Logger(g.t, log.LvlInfo).New("role", "HonestHelper", "game", g.addr)
	l2Client := g.system.NodeClient(l2Node)
	caller := batching.NewMultiCaller(g.system.NodeClient("l1").Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewOutputBisectionGameContract(g.addr, caller)
	g.require.NoError(err, "Failed to create game contact")

	prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	dir := filepath.Join(cfg.Datadir, "honest")
	splitDepth := uint64(g.SplitDepth(ctx))
	rollupClient := g.system.RollupClient(l2Node)
	prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
	accessor, err := outputs.NewOutputCannonTraceAccessor(
		logger, metrics.NoopMetrics, cfg, l2Client, contract, prestateProvider, rollupClient, dir, splitDepth, prestateBlock, poststateBlock)
	g.require.NoError(err, "Failed to create output cannon trace accessor")
	return &OutputHonestHelper{
		t:            g.t,
		require:      g.require,
		game:         &g.OutputGameHelper,
		contract:     contract,
		correctTrace: accessor,
	}
}
