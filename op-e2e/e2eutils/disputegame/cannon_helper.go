package disputegame

import (
	"context"
	"path/filepath"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
)

type CannonGameHelper struct {
	FaultGameHelper
}

func (g *CannonGameHelper) StartChallenger(ctx context.Context, rollupCfg *rollup.Config, l2Genesis *core.Genesis, l1Endpoint string, l2Endpoint string, name string, options ...challenger.Option) *challenger.Helper {
	opts := []challenger.Option{
		challenger.WithCannon(g.t, rollupCfg, l2Genesis, l2Endpoint),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, l1Endpoint, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

func (g *CannonGameHelper) CreateHonestActor(ctx context.Context, rollupCfg *rollup.Config, l2Genesis *core.Genesis, l1Client bind.ContractCaller, l1Endpoint string, l2Endpoint string, options ...challenger.Option) *HonestHelper {
	opts := []challenger.Option{
		challenger.WithCannon(g.t, rollupCfg, l2Genesis, l2Endpoint),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
	opts = append(opts, options...)
	cfg := challenger.NewChallengerConfig(g.t, l1Endpoint, opts...)
	logger := testlog.Logger(g.t, log.LvlInfo).New("role", "CorrectTrace")
	provider, err := cannon.NewTraceProvider(ctx, logger, cfg, l1Client, filepath.Join(cfg.Datadir, "honest"), g.addr)
	g.require.NoError(err, "create cannon trace provider")

	return &HonestHelper{
		t:            g.t,
		require:      g.require,
		game:         &g.FaultGameHelper,
		correctTrace: provider,
	}
}
