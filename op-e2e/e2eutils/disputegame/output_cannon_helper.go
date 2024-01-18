package disputegame

import (
	"context"
	"math/big"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
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
		challenger.WithCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), rollupEndpoint, l2Endpoint),
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
	opts := g.defaultChallengerOptions(l2Node)
	opts = append(opts, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system.NodeEndpoint("l1"), opts...)

	logger := testlog.Logger(g.t, log.LvlInfo).New("role", "HonestHelper", "game", g.addr)
	l2Client := g.system.NodeClient(l2Node)
	caller := batching.NewMultiCaller(g.system.NodeClient("l1").Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewFaultDisputeGameContract(g.addr, caller)
	g.require.NoError(err, "Failed to create game contact")

	prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	dir := filepath.Join(cfg.Datadir, "honest")
	splitDepth := g.SplitDepth(ctx)
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

func (g *OutputCannonGameHelper) CreateCannonTraceProvider(ctx context.Context, l2Node string, outputRootClaim *ClaimHelper, options ...challenger.Option) *cannon.CannonTraceProvider {
	splitDepth := g.SplitDepth(ctx)
	g.require.EqualValues(outputRootClaim.Depth(), splitDepth+1, "outputRootClaim must be the root of an execution game")

	logger := testlog.Logger(g.t, log.LvlInfo).New("role", "CannonTraceProvider", "game", g.addr)
	opt := g.defaultChallengerOptions(l2Node)
	opt = append(opt, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system.NodeEndpoint("l1"), opt...)

	caller := batching.NewMultiCaller(g.system.NodeClient("l1").Client(), batching.DefaultBatchSize)
	l2Client := g.system.NodeClient(l2Node)
	contract, err := contracts.NewFaultDisputeGameContract(g.addr, caller)
	g.require.NoError(err, "Failed to create game contact")

	prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	rollupClient := g.system.RollupClient(l2Node)
	prestateProvider := outputs.NewPrestateProvider(ctx, logger, rollupClient, prestateBlock)
	outputProvider := outputs.NewTraceProviderFromInputs(logger, prestateProvider, rollupClient, splitDepth, prestateBlock, poststateBlock)

	topLeaf := g.getClaim(ctx, int64(outputRootClaim.parentIndex))
	topLeafPosition := types.NewPositionFromGIndex(topLeaf.Position)
	var pre, post types.Claim
	if outputRootClaim.position.TraceIndex(outputRootClaim.Depth()).Cmp(topLeafPosition.TraceIndex(outputRootClaim.Depth())) > 0 {
		pre, err = contract.GetClaim(ctx, uint64(outputRootClaim.parentIndex))
		g.require.NoError(err, "Failed to construct pre claim")
		post, err = contract.GetClaim(ctx, uint64(outputRootClaim.index))
		g.require.NoError(err, "Failed to construct post claim")
	} else {
		post, err = contract.GetClaim(ctx, uint64(outputRootClaim.parentIndex))
		postTraceIdx := post.TraceIndex(splitDepth)
		if postTraceIdx.Cmp(big.NewInt(0)) == 0 {
			pre = types.Claim{}
		} else {
			g.require.NoError(err, "Failed to construct post claim")
			pre, err = contract.GetClaim(ctx, uint64(outputRootClaim.index))
			g.require.NoError(err, "Failed to construct pre claim")
		}
	}
	proposals, err := outputs.FetchProposals(ctx, outputProvider, pre, post)
	g.require.NoError(err, "Failed to fetch proposals")

	localInputs, err := cannon.FetchLocalInputsFromProposals(ctx, contract, l2Client, proposals[0], proposals[1])
	g.require.NoError(err, "Failed to fetch local inputs")
	localContext := outputs.CreateLocalContext(pre, post)
	dir := filepath.Join(cfg.Datadir, "honest-cannon")
	subdir := filepath.Join(dir, localContext.Hex())
	return cannon.NewTraceProvider(logger, metrics.NoopMetrics, cfg, localInputs, subdir, g.MaxDepth(ctx)-splitDepth-1)
}

func (g *OutputCannonGameHelper) defaultChallengerOptions(l2Node string) []challenger.Option {
	return []challenger.Option{
		challenger.WithCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), g.system.RollupEndpoint(l2Node), g.system.NodeEndpoint(l2Node)),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
}
