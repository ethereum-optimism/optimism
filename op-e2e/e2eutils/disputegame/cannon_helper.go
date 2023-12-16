package disputegame

import (
	"context"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
)

type CannonGameHelper struct {
	FaultGameHelper
}

func (g *CannonGameHelper) StartChallenger(ctx context.Context, l2Node string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), g.system.NodeEndpoint(l2Node)),
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

func (g *CannonGameHelper) CreateHonestActor(ctx context.Context, l2Node string, options ...challenger.Option) *HonestHelper {
	opts := []challenger.Option{
		challenger.WithCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), g.system.NodeEndpoint(l2Node)),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
	opts = append(opts, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system.NodeEndpoint("l1"), opts...)
	logger := testlog.Logger(g.t, log.LvlInfo).New("role", "CorrectTrace")
	maxDepth := g.MaxDepth(ctx)
	gameContract, err := contracts.NewFaultDisputeGameContract(g.addr, batching.NewMultiCaller(g.system.NodeClient("l1").Client(), batching.DefaultBatchSize))
	g.require.NoError(err, "Create game contract bindings")
	l2Client := g.system.NodeClient(l2Node)
	localInputs, err := cannon.FetchLocalInputs(ctx, gameContract, l2Client)
	g.require.NoError(err, "fetch cannon local inputs")
	provider := cannon.NewTraceProvider(logger, metrics.NoopMetrics, cfg, types.NoLocalContext, localInputs, filepath.Join(cfg.Datadir, "honest"), uint64(maxDepth))

	return &HonestHelper{
		t:            g.t,
		require:      g.require,
		game:         &g.FaultGameHelper,
		correctTrace: provider,
	}
}
