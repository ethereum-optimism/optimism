package disputegame

import (
	"context"
	"crypto/ecdsa"
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
	"github.com/ethereum/go-ethereum/common"
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

// ChallengeToFirstGlobalPreimageLoad challenges the supplied execution root claim by inducing a step that requires a preimage to be loaded
// It does this by:
// 1. Identifying the first state transition that loads a global preimage
// 2. Descending the execution game tree to reach the step that loads the preimage
// 3. Asserting that the preimage was indeed loaded by an honest challenger (assuming the preimage is not preloaded)
// This expects an odd execution game depth in order for the honest challenger to step on our leaf claim
func (g *OutputCannonGameHelper) ChallengeToFirstGlobalPreimageLoad(ctx context.Context, outputRootClaim *ClaimHelper, challengerKey *ecdsa.PrivateKey, preloadPreimage bool) {
	// 1. Identifying the first state transition that loads a global preimage
	provider := g.createCannonTraceProvider(ctx, "sequencer", outputRootClaim, challenger.WithPrivKey(challengerKey))
	targetTraceIndex, err := provider.FindStepReferencingPreimage(ctx, 0)
	g.require.NoError(err)

	splitDepth := g.SplitDepth(ctx)
	execDepth := g.ExecDepth(ctx)
	g.require.NotEqual(outputRootClaim.position.TraceIndex(execDepth).Uint64(), targetTraceIndex, "cannot move to defend a terminal trace index")
	g.require.EqualValues(splitDepth+1, outputRootClaim.Depth(), "supplied claim must be the root of an execution game")
	g.require.EqualValues(execDepth%2, 1, "execution game depth must be odd") // since we're challenging the execution root claim

	if preloadPreimage {
		_, _, preimageData, err := provider.GetStepData(ctx, types.NewPosition(execDepth, big.NewInt(int64(targetTraceIndex))))
		g.require.NoError(err)
		g.uploadPreimage(ctx, preimageData, challengerKey)
		g.require.True(g.preimageExistsInOracle(ctx, preimageData))
	}

	// 2. Descending the execution game tree to reach the step that loads the preimage
	bisectTraceIndex := func(claim *ClaimHelper) *ClaimHelper {
		execClaimPosition, err := claim.position.RelativeToAncestorAtDepth(splitDepth + 1)
		g.require.NoError(err)

		claimTraceIndex := execClaimPosition.TraceIndex(execDepth).Uint64()
		g.t.Logf("Bisecting: Into targetTraceIndex %v: claimIndex=%v at depth=%v. claimPosition=%v execClaimPosition=%v claimTraceIndex=%v",
			targetTraceIndex, claim.index, claim.Depth(), claim.position, execClaimPosition, claimTraceIndex)

		// We always want to position ourselves such that the challenger generates proofs for the targetTraceIndex as prestate
		if execClaimPosition.Depth() == execDepth-1 {
			if execClaimPosition.TraceIndex(execDepth).Uint64() == targetTraceIndex {
				newPosition := execClaimPosition.Attack()
				correct, err := provider.Get(ctx, newPosition)
				g.require.NoError(err)
				g.t.Logf("Bisecting: Attack correctly for step at newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
				return claim.Attack(ctx, correct)
			} else if execClaimPosition.TraceIndex(execDepth).Uint64() > targetTraceIndex {
				g.t.Logf("Bisecting: Attack incorrectly for step")
				return claim.Attack(ctx, common.Hash{0xdd})
			} else if execClaimPosition.TraceIndex(execDepth).Uint64()+1 == targetTraceIndex {
				g.t.Logf("Bisecting: Defend incorrectly for step")
				return claim.Defend(ctx, common.Hash{0xcc})
			} else {
				newPosition := execClaimPosition.Defend()
				correct, err := provider.Get(ctx, newPosition)
				g.require.NoError(err)
				g.t.Logf("Bisecting: Defend correctly for step at newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
				return claim.Defend(ctx, correct)
			}
		}

		// Attack or Defend depending on whether the claim we're responding to is to the left or right of the trace index
		// Induce the honest challenger to attack or defend depending on whether our new position will be to the left or right of the trace index
		if execClaimPosition.TraceIndex(execDepth).Uint64() < targetTraceIndex && claim.Depth() != splitDepth+1 {
			newPosition := execClaimPosition.Defend()
			if newPosition.TraceIndex(execDepth).Uint64() < targetTraceIndex {
				g.t.Logf("Bisecting: Defend correct. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
				correct, err := provider.Get(ctx, newPosition)
				g.require.NoError(err)
				return claim.Defend(ctx, correct)
			} else {
				g.t.Logf("Bisecting: Defend incorrect. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
				return claim.Defend(ctx, common.Hash{0xaa})
			}
		} else {
			newPosition := execClaimPosition.Attack()
			if newPosition.TraceIndex(execDepth).Uint64() < targetTraceIndex {
				g.t.Logf("Bisecting: Attack correct. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
				correct, err := provider.Get(ctx, newPosition)
				g.require.NoError(err)
				return claim.Attack(ctx, correct)
			} else {
				g.t.Logf("Bisecting: Attack incorrect. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
				return claim.Attack(ctx, common.Hash{0xbb})
			}
		}
	}

	g.LogGameData(ctx)
	// Initial bisect to put us on defense
	claim := bisectTraceIndex(outputRootClaim)
	g.DefendClaim(ctx, claim, bisectTraceIndex)

	// 3. Asserts that the preimage was indeed loaded by an honest challenger
	_, _, preimageData, err := provider.GetStepData(ctx, types.NewPosition(execDepth, big.NewInt(int64(targetTraceIndex))))
	g.require.NoError(err)
	g.require.True(g.preimageExistsInOracle(ctx, preimageData))
}

func (g *OutputCannonGameHelper) createCannonTraceProvider(ctx context.Context, l2Node string, outputRootClaim *ClaimHelper, options ...challenger.Option) *cannon.CannonTraceProviderForTest {
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
	agreed, disputed, err := outputs.FetchProposals(ctx, outputProvider, pre, post)
	g.require.NoError(err, "Failed to fetch proposals")

	localInputs, err := cannon.FetchLocalInputsFromProposals(ctx, contract, l2Client, agreed, disputed)
	g.require.NoError(err, "Failed to fetch local inputs")
	localContext := outputs.CreateLocalContext(pre, post)
	dir := filepath.Join(cfg.Datadir, "cannon-trace")
	subdir := filepath.Join(dir, localContext.Hex())
	return cannon.NewTraceProviderForTest(logger, metrics.NoopMetrics, cfg, localInputs, subdir, g.MaxDepth(ctx)-splitDepth-1)
}

func (g *OutputCannonGameHelper) defaultChallengerOptions(l2Node string) []challenger.Option {
	return []challenger.Option{
		challenger.WithCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), g.system.RollupEndpoint(l2Node), g.system.NodeEndpoint(l2Node)),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
}
