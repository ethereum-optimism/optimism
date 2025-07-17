package disputegame

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"path/filepath"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/super"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/vm"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
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
			challenger.WithDepset(t, system.DependencySet()),
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

// ChallengeToPreimageLoad challenges the supplied execution root claim by inducing a step that requires a preimage to be loaded
// It does this by:
// 1. Identifying the first state transition that loads a global preimage
// 2. Descending the execution game tree to reach the step that loads the preimage
// 3. Asserting that the preimage was indeed loaded by an honest challenger (assuming the preimage is not preloaded)
// This expects an even execution game depth in order for the honest challenger to step on our leaf claim
// PRECOND:
// - The topGameLeaf must be incorrect
// - The execution game depth must be even
func (g *SuperCannonGameHelper) ChallengeToPreimageLoad(ctx context.Context, topGameLeaf *ClaimHelper, challengerKey *ecdsa.PrivateKey, preimage utils.PreimageOpt, preimageCheck PreimageLoadCheck, preloadPreimage bool) {
	provider := g.createSuperCannonTraceProvider(ctx, topGameLeaf, challenger.WithPrivKey(challengerKey))

	targetTraceIndex, err := provider.FindStep(ctx, 0, preimage)
	g.require.NoError(err)

	splitDepth := g.splitGame.SplitDepth(ctx)
	execDepth := g.splitGame.ExecDepth(ctx)
	g.require.NotEqual(topGameLeaf.Position.TraceIndex(execDepth).Uint64(), targetTraceIndex, "cannot move to defend a terminal trace index")
	g.require.EqualValues(splitDepth+1, topGameLeaf.Depth(), "supplied claim must be the root of an execution game")
	g.require.EqualValues(execDepth%2, 0, "execution game depth must be even") // since we're supporting the execution root claim

	if preloadPreimage {
		_, _, preimageData, err := provider.GetStepData(ctx, types.NewPosition(execDepth, big.NewInt(int64(targetTraceIndex))))
		g.require.NoError(err)
		g.UploadPreimage(ctx, preimageData)
		g.WaitForPreimageInOracle(ctx, preimageData)
	}

	bisectTraceIndex := func(claim *ClaimHelper) *ClaimHelper {
		return traceBisection(g.t, ctx, claim, splitDepth, execDepth, targetTraceIndex, provider)
	}
	leafClaim := g.splitGame.DefendClaim(ctx, topGameLeaf, bisectTraceIndex, WithoutWaitingForStep())

	// Validate that the preimage was loaded correctly
	g.require.NoError(preimageCheck(provider, targetTraceIndex))

	// Now the preimage is available wait for the step call to succeed.
	leafClaim.WaitForCountered(ctx)
	g.splitGame.LogGameData(ctx)
}

// SupportClaimIntoTargetTraceIndex supports the specified claim while bisecting to the target trace index at split depth.
func (g *SuperCannonGameHelper) SupportClaimIntoTargetTraceIndex(ctx context.Context, claim *ClaimHelper, targetTraceIndexAtSplitDepth uint64) {
	provider := g.createSuperTraceProvider(ctx)
	g.SupportClaim(ctx, claim, func(claim *ClaimHelper) *ClaimHelper {
		if claim.IsOutputRoot(ctx) {
			return topGameTraceBisection(g.t, ctx, claim, g.splitGame.SplitDepth(ctx), targetTraceIndexAtSplitDepth, provider)
		} else {
			return claim.Attack(ctx, common.Hash{0xbb})
		}
	}, func(parentIdx int64) {
		g.splitGame.StepFails(ctx, parentIdx, false, []byte{}, []byte{})
		g.splitGame.StepFails(ctx, parentIdx, true, []byte{}, []byte{})
	})
	g.splitGame.LogGameData(ctx)
}

func (g *SuperCannonGameHelper) createSuperCannonTraceProvider(ctx context.Context, proposal *ClaimHelper, options ...challenger.Option) *cannon.CannonTraceProviderForTest {
	splitDepth := g.splitGame.SplitDepth(ctx)
	g.require.EqualValues(proposal.Depth(), splitDepth+1, "outputRootClaim must be the root of an execution game")

	logger := testlog.Logger(g.t, log.LevelInfo).New("role", "CannonTraceProvider", "game", g.splitGame.Addr)
	opt := g.defaultChallengerOptions()
	opt = append(opt, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system, "", opt...)

	l1Head := g.GetL1Head(ctx)
	_, poststateTimestamp, err := g.Game.GetGameRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	superProvider := g.createSuperTraceProvider(ctx)

	var localContext common.Hash
	selector := split.NewSplitProviderSelector(superProvider, splitDepth, func(ctx context.Context, depth types.Depth, pre types.Claim, post types.Claim) (types.TraceProvider, error) {
		claimInfo, err := super.FetchClaimInfo(ctx, superProvider, pre, post)
		g.require.NoError(err, "failed to fetch claim info")
		localInputs := utils.LocalGameInputs{
			L1Head:           l1Head.Hash,
			AgreedPreState:   claimInfo.AgreedPrestate,
			L2Claim:          claimInfo.Claim,
			L2SequenceNumber: new(big.Int).SetUint64(poststateTimestamp),
		}
		localContext = split.CreateLocalContext(pre, post)
		dir := filepath.Join(cfg.Datadir, "super-cannon-trace")
		subdir := filepath.Join(dir, localContext.Hex())
		return cannon.NewTraceProviderForTest(logger, metrics.NoopMetrics.ToTypedVmMetrics(types.TraceTypeCannon.String()), cfg, localInputs, subdir, g.splitGame.MaxDepth(ctx)-splitDepth-1), nil
	})

	claims, err := g.splitGame.Game.GetAllClaims(ctx, rpcblock.Latest)
	g.require.NoError(err)
	game := types.NewGameState(claims, g.splitGame.MaxDepth(ctx))

	provider, err := selector(ctx, game, game.Claims()[proposal.ParentIndex], proposal.Position)
	g.require.NoError(err)
	translatingProvider := provider.(*trace.TranslatingProvider)
	return translatingProvider.Original().(*cannon.CannonTraceProviderForTest)
}

func (g *SuperCannonGameHelper) createSuperTraceProvider(ctx context.Context) *super.SuperTraceProvider {
	logger := testlog.Logger(g.t, log.LevelInfo).New("role", "superTraceProvider", "game", g.splitGame.Addr)
	rootProvider := g.System.SupervisorClient()
	splitDepth := g.splitGame.SplitDepth(ctx)
	l1Head := g.GetL1Head(ctx)
	prestateTimestamp, poststateTimestamp, err := g.Game.GetGameRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	prestateProvider := super.NewSuperRootPrestateProvider(rootProvider, prestateTimestamp)
	rollupCfgs, err := super.NewRollupConfigsFromParsed(g.System.RollupCfgs()...)
	require.NoError(g.T, err, "failed to create rollup configs")
	return super.NewSuperTraceProvider(logger, rollupCfgs, prestateProvider, rootProvider, l1Head, splitDepth, prestateTimestamp, poststateTimestamp)
}
