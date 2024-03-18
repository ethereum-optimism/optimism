package disputegame

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"io"
	"math/big"
	"path/filepath"
	"time"

	"github.com/ethereum-optimism/optimism/op-bindings/bindings"
	"github.com/ethereum-optimism/optimism/op-chain-ops/deployer"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
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
	c := challenger.NewChallenger(g.t, ctx, g.system, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

func (g *OutputCannonGameHelper) CreateHonestActor(ctx context.Context, l2Node string, options ...challenger.Option) *OutputHonestHelper {
	opts := g.defaultChallengerOptions(l2Node)
	opts = append(opts, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system, opts...)

	logger := testlog.Logger(g.t, log.LevelInfo).New("role", "HonestHelper", "game", g.addr)
	l2Client := g.system.NodeClient(l2Node)
	caller := batching.NewMultiCaller(g.system.NodeClient("l1").Client(), batching.DefaultBatchSize)
	contract, err := contracts.NewFaultDisputeGameContract(g.addr, caller)
	g.require.NoError(err, "Failed to create game contact")

	prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	dir := filepath.Join(cfg.Datadir, "honest")
	splitDepth := g.SplitDepth(ctx)
	rollupClient := g.system.RollupClient(l2Node)
	prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
	l1Head := g.getL1Head(ctx)
	accessor, err := outputs.NewOutputCannonTraceAccessor(
		logger, metrics.NoopMetrics, cfg, l2Client, prestateProvider, rollupClient, dir, l1Head, splitDepth, prestateBlock, poststateBlock)
	g.require.NoError(err, "Failed to create output cannon trace accessor")
	return &OutputHonestHelper{
		t:            g.t,
		require:      g.require,
		game:         &g.OutputGameHelper,
		contract:     contract,
		correctTrace: accessor,
	}
}

type PreimageLoadCheck func(types.TraceProvider, uint64) error

func (g *OutputCannonGameHelper) CreateStepLargePreimageLoadCheck(ctx context.Context, sender common.Address) PreimageLoadCheck {
	return func(provider types.TraceProvider, targetTraceIndex uint64) error {
		// Fetch the challenge period
		challengePeriod := g.ChallengePeriod(ctx)

		// Get the preimage data
		execDepth := g.ExecDepth(ctx)
		_, _, preimageData, err := provider.GetStepData(ctx, types.NewPosition(execDepth, big.NewInt(int64(targetTraceIndex))))
		g.require.NoError(err)

		// Wait until the challenge period has started by checking until the challenge
		// period start time is not zero by calling the ChallengePeriodStartTime method
		g.WaitForChallengePeriodStart(ctx, sender, preimageData)

		challengePeriodStart := g.ChallengePeriodStartTime(ctx, sender, preimageData)
		challengePeriodEnd := challengePeriodStart + challengePeriod

		// Time travel past the challenge period.
		g.system.AdvanceTime(time.Duration(challengePeriod) * time.Second)
		g.require.NoError(wait.ForBlockWithTimestamp(ctx, g.system.NodeClient("l1"), challengePeriodEnd))

		// Assert that the preimage was indeed loaded by an honest challenger
		g.waitForPreimageInOracle(ctx, preimageData)
		return nil
	}
}

func (g *OutputCannonGameHelper) CreateStepPreimageLoadCheck(ctx context.Context) PreimageLoadCheck {
	return func(provider types.TraceProvider, targetTraceIndex uint64) error {
		execDepth := g.ExecDepth(ctx)
		_, _, preimageData, err := provider.GetStepData(ctx, types.NewPosition(execDepth, big.NewInt(int64(targetTraceIndex))))
		g.require.NoError(err)
		g.waitForPreimageInOracle(ctx, preimageData)
		return nil
	}
}

// ChallengeToPreimageLoad challenges the supplied execution root claim by inducing a step that requires a preimage to be loaded
// It does this by:
// 1. Identifying the first state transition that loads a global preimage
// 2. Descending the execution game tree to reach the step that loads the preimage
// 3. Asserting that the preimage was indeed loaded by an honest challenger (assuming the preimage is not preloaded)
// This expects an odd execution game depth in order for the honest challenger to step on our leaf claim
func (g *OutputCannonGameHelper) ChallengeToPreimageLoad(ctx context.Context, outputRootClaim *ClaimHelper, challengerKey *ecdsa.PrivateKey, preimage cannon.PreimageOpt, preimageCheck PreimageLoadCheck, preloadPreimage bool) {
	// Identifying the first state transition that loads a global preimage
	provider, _ := g.createCannonTraceProvider(ctx, "sequencer", outputRootClaim, challenger.WithPrivKey(challengerKey))
	targetTraceIndex, err := provider.FindStep(ctx, 0, preimage)
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
		g.waitForPreimageInOracle(ctx, preimageData)
	}

	// Descending the execution game tree to reach the step that loads the preimage
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
	mover := bisectTraceIndex(outputRootClaim)
	leafClaim := g.DefendClaim(ctx, mover, bisectTraceIndex, WithoutWaitingForStep())

	// Validate that the preimage was loaded correctly
	g.require.NoError(preimageCheck(provider, targetTraceIndex))

	// Now the preimage is available wait for the step call to succeed.
	leafClaim.WaitForCountered(ctx)
	g.LogGameData(ctx)
}

func (g *OutputCannonGameHelper) VerifyPreimage(ctx context.Context, outputRootClaim *ClaimHelper, preimageKey preimage.Key) {
	execDepth := g.ExecDepth(ctx)

	// Identifying the first state transition that loads a global preimage
	provider, localContext := g.createCannonTraceProvider(ctx, "sequencer", outputRootClaim, challenger.WithPrivKey(deployer.TestKey))
	start := uint64(0)
	found := false
	for offset := uint32(0); ; offset += 4 {
		preimageOpt := cannon.PreimageLoad(preimageKey, offset)
		g.t.Logf("Searching for step with key %x and offset %v", preimageKey.PreimageKey(), offset)
		targetTraceIndex, err := provider.FindStep(ctx, start, preimageOpt)
		if errors.Is(err, io.EOF) {
			// Did not find any more reads
			g.require.True(found, "Should have found at least one preimage read")
			g.t.Logf("Searching for step with key %x and offset %v did not find another read", preimageKey.PreimageKey(), offset)
			return
		}
		g.require.NoError(err, "Failed to find step that loads requested preimage")
		start = targetTraceIndex
		found = true

		g.t.Logf("Target trace index: %v", targetTraceIndex)
		pos := types.NewPosition(execDepth, new(big.Int).SetUint64(targetTraceIndex))
		g.require.Equal(targetTraceIndex, pos.TraceIndex(execDepth).Uint64())

		prestate, proof, oracleData, err := provider.GetStepData(ctx, pos)
		g.require.NoError(err, "Failed to get step data")
		g.require.NotNil(oracleData, "Should have had required preimage oracle data")
		g.require.Equal(common.Hash(preimageKey.PreimageKey()).Bytes(), oracleData.OracleKey, "Must have correct preimage key")

		tx, err := g.game.AddLocalData(g.opts,
			oracleData.GetIdent(),
			big.NewInt(outputRootClaim.index),
			new(big.Int).SetUint64(uint64(oracleData.OracleOffset)))
		g.require.NoError(err)
		_, err = wait.ForReceiptOK(ctx, g.client, tx.Hash())
		g.require.NoError(err)

		expectedPostState, err := provider.Get(ctx, pos)
		g.require.NoError(err, "Failed to get expected post state")

		callOpts := &bind.CallOpts{Context: ctx}
		vmAddr, err := g.game.Vm(callOpts)
		g.require.NoError(err, "Failed to get VM address")

		abi, err := bindings.MIPSMetaData.GetAbi()
		g.require.NoError(err, "Failed to load MIPS ABI")
		caller := batching.NewMultiCaller(g.client.Client(), batching.DefaultBatchSize)
		result, err := caller.SingleCall(ctx, rpcblock.Latest, &batching.ContractCall{
			Abi:    abi,
			Addr:   vmAddr,
			Method: "step",
			Args: []interface{}{
				prestate, proof, localContext,
			},
			From: g.addr,
		})
		g.require.NoError(err, "Failed to call step")
		actualPostState := result.GetBytes32(0)
		g.require.Equal(expectedPostState, common.Hash(actualPostState))
	}
}

func (g *OutputCannonGameHelper) createCannonTraceProvider(ctx context.Context, l2Node string, outputRootClaim *ClaimHelper, options ...challenger.Option) (*cannon.CannonTraceProviderForTest, common.Hash) {
	splitDepth := g.SplitDepth(ctx)
	g.require.EqualValues(outputRootClaim.Depth(), splitDepth+1, "outputRootClaim must be the root of an execution game")

	logger := testlog.Logger(g.t, log.LevelInfo).New("role", "CannonTraceProvider", "game", g.addr)
	opt := g.defaultChallengerOptions(l2Node)
	opt = append(opt, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system, opt...)

	caller := batching.NewMultiCaller(g.system.NodeClient("l1").Client(), batching.DefaultBatchSize)
	l2Client := g.system.NodeClient(l2Node)
	contract, err := contracts.NewFaultDisputeGameContract(g.addr, caller)
	g.require.NoError(err, "Failed to create game contact")

	prestateBlock, poststateBlock, err := contract.GetBlockRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	rollupClient := g.system.RollupClient(l2Node)
	prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
	l1Head := g.getL1Head(ctx)
	outputProvider := outputs.NewTraceProvider(logger, prestateProvider, rollupClient, l1Head, splitDepth, prestateBlock, poststateBlock)

	var localContext common.Hash
	selector := split.NewSplitProviderSelector(outputProvider, splitDepth, func(ctx context.Context, depth types.Depth, pre types.Claim, post types.Claim) (types.TraceProvider, error) {
		agreed, disputed, err := outputs.FetchProposals(ctx, outputProvider, pre, post)
		g.require.NoError(err)
		g.t.Logf("Using trace between blocks %v and %v\n", agreed.L2BlockNumber, disputed.L2BlockNumber)
		localInputs, err := cannon.FetchLocalInputsFromProposals(ctx, l1Head.Hash, l2Client, agreed, disputed)
		g.require.NoError(err, "Failed to fetch local inputs")
		localContext = outputs.CreateLocalContext(pre, post)
		dir := filepath.Join(cfg.Datadir, "cannon-trace")
		subdir := filepath.Join(dir, localContext.Hex())
		return cannon.NewTraceProviderForTest(logger, metrics.NoopMetrics, cfg, localInputs, subdir, g.MaxDepth(ctx)-splitDepth-1), nil
	})

	claims, err := contract.GetAllClaims(ctx, rpcblock.Latest)
	g.require.NoError(err)
	game := types.NewGameState(claims, g.MaxDepth(ctx))

	provider, err := selector(ctx, game, game.Claims()[outputRootClaim.parentIndex], outputRootClaim.position)
	g.require.NoError(err)
	translatingProvider := provider.(*trace.TranslatingProvider)
	return translatingProvider.Original().(*cannon.CannonTraceProviderForTest), localContext
}

func (g *OutputCannonGameHelper) defaultChallengerOptions(l2Node string) []challenger.Option {
	return []challenger.Option{
		challenger.WithCannon(g.t, g.system.RollupCfg(), g.system.L2Genesis(), g.system.RollupEndpoint(l2Node), g.system.NodeEndpoint(l2Node)),
		challenger.WithFactoryAddress(g.factoryAddr),
		challenger.WithGameAddress(g.addr),
	}
}
