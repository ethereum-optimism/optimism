package disputegame

import (
	"context"
	"crypto/ecdsa"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type SuperCannonGameHelper struct {
	SuperGameHelper
	CannonHelper
}

func NewSuperCannonGameHelper(t *testing.T, client *ethclient.Client, opts *bind.TransactOpts, key *ecdsa.PrivateKey, game contracts.FaultDisputeGameContract, factoryAddr common.Address, gameAddr common.Address, provider *super.SuperTraceProvider, system DisputeSystem) *SuperCannonGameHelper {
	superGameHelper := NewSuperGameHelper(t, require.New(t), client, opts, key, game, factoryAddr, gameAddr, provider, system)
	defaultChallengerOptions := func() []challenger.Option {
		return []challenger.Option{
			challenger.WithSuperCannon(t, system),
			challenger.WithFactoryAddress(factoryAddr),
			challenger.WithGameAddress(gameAddr),
		}
	}
	return &SuperCannonGameHelper{
		SuperGameHelper: *superGameHelper,
		CannonHelper:    *NewCannonHelper(&superGameHelper.SplitGameHelper, defaultChallengerOptions),
	}
}

func (g *SuperCannonGameHelper) CreateHonestActor(ctx context.Context, options ...HonestActorOpt) *OutputHonestHelper {
	logger := testlog.Logger(g.T, log.LevelInfo).New("role", "HonestHelper", "game", g.Addr)

	realPrestateBlock, realPostStateBlock, err := g.Game.GetGameRange(ctx)
	g.Require.NoError(err, "Failed to load block range")
	splitDepth := g.SplitDepth(ctx)
	supervisorClient := g.System.SupervisorClient()
	actorCfg := &HonestActorConfig{
		PrestateSequenceNumber:  realPrestateBlock,
		PoststateSequenceNumber: realPostStateBlock,
		ChallengerOpts:          g.defaultChallengerOptions(),
	}
	for _, option := range options {
		option(actorCfg)
	}

	cfg := challenger.NewChallengerConfig(g.T, g.System, "", actorCfg.ChallengerOpts...)
	dir := filepath.Join(cfg.Datadir, "honest")
	prestateProvider := super.NewSuperRootPrestateProvider(supervisorClient, actorCfg.PrestateSequenceNumber)
	l1Head := g.GetL1Head(ctx)
	accessor, err := super.NewSuperCannonTraceAccessor(
		logger,
		metrics.NoopMetrics,
		cfg.Cannon,
		vm.NewOpProgramServerExecutor(logger),
		prestateProvider,
		supervisorClient,
		cfg.CannonAbsolutePreState,
		dir,
		l1Head,
		splitDepth,
		actorCfg.PrestateSequenceNumber,
		actorCfg.PoststateSequenceNumber,
	)
	g.Require.NoError(err, "Failed to create output cannon trace accessor")
	return NewOutputHonestHelper(g.T, g.Require, &g.SuperGameHelper.SplitGameHelper, g.Game, accessor)

}
