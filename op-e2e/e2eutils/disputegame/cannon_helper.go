package disputegame

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"io"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/contracts"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/preimages"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/cannon"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/outputs"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/split"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-e2e/bindings"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/transactions"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	preimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching"
	"github.com/ethereum-optimism/optimism/op-service/sources/batching/rpcblock"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type CannonHelper struct {
	t                        *testing.T
	require                  *require.Assertions
	client                   *ethclient.Client
	privKey                  *ecdsa.PrivateKey
	system                   DisputeSystem
	splitGame                *SplitGameHelper
	defaultChallengerOptions func() []challenger.Option
}

func NewCannonHelper(splitGameHelper *SplitGameHelper, defaultChallengerOptions func() []challenger.Option) *CannonHelper {
	return &CannonHelper{
		t:                        splitGameHelper.T,
		require:                  splitGameHelper.Require,
		client:                   splitGameHelper.Client,
		privKey:                  splitGameHelper.PrivKey,
		splitGame:                splitGameHelper,
		system:                   splitGameHelper.System,
		defaultChallengerOptions: defaultChallengerOptions,
	}
}

func (g *CannonHelper) StartChallenger(ctx context.Context, name string, options ...challenger.Option) *challenger.Helper {
	opts := g.defaultChallengerOptions()
	opts = append(opts, options...)
	c := challenger.NewChallenger(g.t, ctx, g.system, name, opts...)
	g.t.Cleanup(func() {
		_ = c.Close()
	})
	return c
}

// ChallengePeriod returns the challenge period fetched from the PreimageOracle contract.
// The returned uint64 value is the number of seconds for the challenge period.
func (g *CannonHelper) ChallengePeriod(ctx context.Context) uint64 {
	oracle := g.oracle(ctx)
	period, err := oracle.ChallengePeriod(ctx)
	g.require.NoError(err, "Failed to get challenge period")
	return period
}

// WaitForChallengePeriodStart waits for the challenge period to start for a given large preimage claim.
func (g *CannonHelper) WaitForChallengePeriodStart(ctx context.Context, sender common.Address, data *types.PreimageOracleData) {
	timedCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		ctx, cancel := context.WithTimeout(timedCtx, 30*time.Second)
		defer cancel()
		timestamp := g.ChallengePeriodStartTime(ctx, sender, data)
		g.t.Log("Waiting for challenge period start", "timestamp", timestamp, "key", data.OracleKey, "game", g.splitGame.Addr)
		return timestamp > 0, nil
	})
	if err != nil {
		g.splitGame.LogGameData(ctx)
		g.require.NoErrorf(err, "Failed to get challenge start period for preimage data %v", data)
	}
}

// ChallengePeriodStartTime returns the start time of the challenge period for a given large preimage claim.
// If the returned start time is 0, the challenge period has not started.
func (g *CannonHelper) ChallengePeriodStartTime(ctx context.Context, sender common.Address, data *types.PreimageOracleData) uint64 {
	oracle := g.oracle(ctx)
	uuid := preimages.NewUUID(sender, data)
	metadata, err := oracle.GetProposalMetadata(ctx, rpcblock.Latest, keccakTypes.LargePreimageIdent{
		Claimant: sender,
		UUID:     uuid,
	})
	g.require.NoError(err, "Failed to get proposal metadata")
	if len(metadata) == 0 {
		return 0
	}
	return metadata[0].Timestamp
}

func (g *CannonHelper) WaitForPreimageInOracle(ctx context.Context, data *types.PreimageOracleData) {
	timedCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	oracle := g.oracle(ctx)
	err := wait.For(timedCtx, time.Second, func() (bool, error) {
		g.t.Logf("Waiting for preimage (%v) to be present in oracle", common.Bytes2Hex(data.OracleKey))
		return oracle.GlobalDataExists(ctx, data)
	})
	g.require.NoErrorf(err, "Did not find preimage (%v) in oracle", common.Bytes2Hex(data.OracleKey))
}

// CheckPreimageInOracle verifies that expectedData is stored on-chain in the PreimageOracle
func (g *CannonHelper) CheckPreimageInOracle(ctx context.Context, data *types.PreimageOracleData, expectedData [32]byte) {
	// Make sure data for this key and offset has been set before attempting to retrieve the data
	g.WaitForPreimageInOracle(ctx, data)

	// Pull preimage data from oracle and verify it matches our expectation
	globalData, err := g.oracle(ctx).GetGlobalData(ctx, data)
	g.require.NoError(err, "Failed to fetch global data")
	g.require.Equal(expectedData[:], globalData[:], "Preimage data is incorrect")
}

// GetPreimageAtOffset returns a slice of the preimage data (with size prefix) at the specified offset
// This can be used to determine what preimage part we expect to be stored in the PreimageOracle contract.
// This only works for some types of preimages, however.  For example, precompile preimages store data that is
// computed on-chain
func (g *CannonHelper) GetPreimageAtOffset(data *types.PreimageOracleData) [32]byte {
	var part [32]byte

	offset := data.OracleOffset
	copy(part[:], data.GetPreimageWithSize()[offset:])

	return part
}

func (g *CannonHelper) UploadPreimage(ctx context.Context, data *types.PreimageOracleData) {
	oracle := g.oracle(ctx)
	tx, err := oracle.AddGlobalDataTx(data)
	g.require.NoError(err, "Failed to create preimage upload tx")
	transactions.RequireSendTx(g.t, ctx, g.client, tx, g.privKey)
}

func (g *CannonHelper) oracle(ctx context.Context) contracts.PreimageOracleContract {
	oracle, err := g.splitGame.Game.GetOracle(ctx)
	g.require.NoError(err, "Failed to create oracle contract")
	return oracle
}

type PreimageLoadCheck func(types.TraceProvider, uint64) error

func (g *CannonHelper) CreateStepLargePreimageLoadCheck(ctx context.Context, sender common.Address) PreimageLoadCheck {
	return func(provider types.TraceProvider, targetTraceIndex uint64) error {
		// Fetch the challenge period
		challengePeriod := g.ChallengePeriod(ctx)

		// Get the preimage data
		execDepth := g.splitGame.ExecDepth(ctx)
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
		g.WaitForPreimageInOracle(ctx, preimageData)
		return nil
	}
}

// ExpectedDataGetter should return (true, data) if expected data should be checked
// Otherwise should return (false, empty data)
type ExpectedDataGetter = func(p *types.PreimageOracleData) (bool, [32]byte)

// CreateStepPreimageLoadCheck returns a PreimageLoadCheck that generates the expected preimage data and
// verifies that data for the expected key is stored on-chain (PreimageOracle.preimagePartOk[key][offset] == true).
func (g *CannonHelper) CreateStepPreimageLoadCheck(ctx context.Context) PreimageLoadCheck {
	noop := func(_ *types.PreimageOracleData) (bool, [32]byte) { return false, [32]byte{} }
	return g.createStepPreimageLoadCheck(ctx, noop)
}

// CreateStepPreimageLoadStrictCheck returns a PreimageLoadCheck that generates the expected preimage data and
// verifies that data for the expected key is stored on-chain (PreimageOracle.preimagePartOk[key][offset] == true).
// Additionally, it checks that the data stored on-chain (in PreimageOracle.preimageParts) is what we expect.
func (g *CannonHelper) CreateStepPreimageLoadStrictCheck(ctx context.Context, expectedData ExpectedDataGetter) PreimageLoadCheck {
	g.require.NotEmpty(expectedData)
	return g.createStepPreimageLoadCheck(ctx, expectedData)
}

func (g *CannonHelper) createStepPreimageLoadCheck(ctx context.Context, getExpectedData ExpectedDataGetter) PreimageLoadCheck {
	return func(provider types.TraceProvider, targetTraceIndex uint64) error {
		execDepth := g.splitGame.ExecDepth(ctx)
		_, _, preimageData, err := provider.GetStepData(ctx, types.NewPosition(execDepth, big.NewInt(int64(targetTraceIndex))))
		g.require.NoError(err)

		// Check that preimagePartOk[preimageData.OracleKey][preimageData.OracleOffset] == true
		g.WaitForPreimageInOracle(ctx, preimageData)

		// If requested, check that the expected data is actually in the oracle
		// Specifically, check that preimageParts[preimageData.OracleKey][preimageData.OracleOffset] == expectedDta
		shouldCheckData, expectedData := getExpectedData(preimageData)
		if shouldCheckData {
			g.CheckPreimageInOracle(ctx, preimageData, expectedData)
		}

		return nil
	}
}

type FindPreimageStepConfig struct {
	allowEvenFallback  bool
	skipNPreimageLoads int
}

type FindPreimageStepOpt func(cfg *FindPreimageStepConfig)

func AllowEvenFallback() FindPreimageStepOpt {
	return func(config *FindPreimageStepConfig) {
		config.allowEvenFallback = true
	}
}

func SkipNPreimageLoads(n int) FindPreimageStepOpt {
	return func(config *FindPreimageStepConfig) {
		config.skipNPreimageLoads = n
	}
}

// FindOddStepForPreimageLoad attempts to find an odd step that matches the PreimageOptConfig.
// If no such step is found, falls back to an even step if allowEvenFallback is true, otherwise fails.
func (g *CannonHelper) FindOddStepForPreimageLoad(ctx context.Context, cannonTraceProviderFunc CannonTraceProviderFunc, poConfig utils.PreimageOptConfig, opts ...FindPreimageStepOpt) uint64 {
	config := &FindPreimageStepConfig{}
	for _, opt := range opts {
		opt(config)
	}

	provider, _, _ := cannonTraceProviderFunc()

	var preimageOpt utils.PreimageOpt
	var lastStep uint64 = 0
	// First, if requested, skip some number of preimage loads
	for i := 0; i < config.skipNPreimageLoads; i++ {
		preimageOpt = poConfig.PreimageLoad()
		g.t.Logf("Skipping preimage load %v: %#v", i, poConfig)
		step, err := provider.FindStep(ctx, lastStep, preimageOpt)
		g.require.NoError(err)

		lastStep = step
		poConfig.AfterStep = step + 1
	}

	lastSkippedStep := lastStep
	for {
		preimageOpt = poConfig.PreimageLoad()
		g.t.Logf("Finding step with preimage load config %#v", poConfig)
		step, err := provider.FindStep(ctx, lastStep, preimageOpt)
		if errors.Is(err, io.EOF) {
			// Unlikely to happen if many preimage loads of the target type occur
			// Can cause flakes if the target preimage type is not used often
			if config.allowEvenFallback && lastStep > lastSkippedStep {
				// If we have advanced the last step past the last skipped step, then the lastStep must be even.
				// Otherwise, the last step was odd and we should have returned it.
				g.t.Log("Unable to find odd step that matches the specified preimage load - falling back to an even step")
				return lastStep
			} else {
				g.t.Fatalf("Trace does not contain an odd step that matches the specified preimage load")
			}
		}
		g.require.NoError(err, "Find step failed")
		if step%2 == 1 {
			return step
		}
		lastStep = step
		poConfig.AfterStep = step + 1
	}
}

// ChallengeToPreimageLoad challenges the supplied execution root claim by inducing a step that requires a preimage to be loaded
// It does this by:
// 1. Identifying the first state transition that loads a global preimage
// 2. Descending the execution game tree to reach the step that loads the preimage
// 3. Asserting that the preimage was indeed loaded by an honest challenger (assuming the preimage is not preloaded)
// This expects an odd execution game depth in order for the honest challenger to step on our leaf claim
func (g *CannonHelper) ChallengeToPreimageLoad(ctx context.Context, cannonTraceProviderFunc CannonTraceProviderFunc, preimage utils.PreimageOpt, preimageCheck PreimageLoadCheck, preloadPreimage bool) {
	// Identifying the first state transition that loads a global preimage
	provider, _, _ := cannonTraceProviderFunc()
	targetTraceIndex, err := provider.FindStep(ctx, 0, preimage)
	g.require.NoError(err)
	g.ChallengeToPreimageLoadAtTarget(ctx, cannonTraceProviderFunc, targetTraceIndex, preimageCheck, preloadPreimage)
}

// ChallengeToPreimageLoadAtTarget challenges the supplied execution root claim by inducing a step that requires a preimage to be loaded
// It does this by:
// 1. Descending the execution game tree to reach the target step that loads the preimage
// 2. Asserting that the preimage was indeed loaded by an honest challenger (assuming the preimage is not preloaded)
// This expects an odd execution game depth in order for the honest challenger to step on our leaf claim
func (g *CannonHelper) ChallengeToPreimageLoadAtTarget(ctx context.Context, cannonTraceProviderFunc CannonTraceProviderFunc, targetTraceIndex uint64, preimageCheck PreimageLoadCheck, preloadPreimage bool) {
	provider, _, outputRootClaim := cannonTraceProviderFunc()
	splitDepth := g.splitGame.SplitDepth(ctx)
	execDepth := g.splitGame.ExecDepth(ctx)
	g.require.NotEqual(outputRootClaim.Position.TraceIndex(execDepth).Uint64(), targetTraceIndex, "cannot move to defend a terminal trace index")
	g.require.EqualValues(splitDepth+1, outputRootClaim.Depth(), "supplied claim must be the root of an execution game")
	g.require.EqualValues(execDepth%2, 1, "execution game depth must be odd") // since we're challenging the execution root claim

	if preloadPreimage {
		_, _, preimageData, err := provider.GetStepData(ctx, types.NewPosition(execDepth, big.NewInt(int64(targetTraceIndex))))
		g.require.NoError(err)
		g.UploadPreimage(ctx, preimageData)
		g.WaitForPreimageInOracle(ctx, preimageData)
	}

	bisectTraceIndex := func(claim *ClaimHelper) *ClaimHelper {
		return traceBisection(g.t, ctx, claim, splitDepth, execDepth, targetTraceIndex, provider)
	}
	// Initial bisect to put us on defense
	mover := bisectTraceIndex(outputRootClaim)
	// Descending the execution game tree to reach the step that loads the preimage
	leafClaim := g.splitGame.DefendClaim(ctx, mover, bisectTraceIndex, WithoutWaitingForStep())

	// Validate that the preimage was loaded correctly
	g.require.NoError(preimageCheck(provider, targetTraceIndex))

	// Now the preimage is available wait for the step call to succeed.
	leafClaim.WaitForCountered(ctx)
	g.splitGame.LogGameData(ctx)
}

func (g *CannonHelper) VerifyPreimage(ctx context.Context, cannonTraceProviderFunc CannonTraceProviderFunc, preimageKey preimage.Key) {
	// Identifying the first state transition that loads a global preimage
	provider, _, _ := cannonTraceProviderFunc()
	start := uint64(0)
	found := false
	for offset := uint32(0); ; offset += 4 {
		preimageOpt := utils.PreimageLoad(preimageKey, offset)
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

		g.VerifyPreimageAtTarget(ctx, cannonTraceProviderFunc, targetTraceIndex, g.GetOracleKeyValidator(preimageKey), true)
	}
}

type OracleDataValidator = func(oracleData *types.PreimageOracleData)

func (g *CannonHelper) GetOracleKeyPrefixValidator(prefix byte) OracleDataValidator {
	return func(oracleData *types.PreimageOracleData) {
		g.require.Equal(prefix, oracleData.OracleKey[0], "Must have correct preimage key prefix")
	}
}

func (g *CannonHelper) GetOracleKeyValidator(key preimage.Key) OracleDataValidator {
	return func(oracleData *types.PreimageOracleData) {
		g.require.Equal(common.Hash(key.PreimageKey()).Bytes(), oracleData.OracleKey, "Must have correct preimage key")
	}
}

func (g *CannonHelper) VerifyPreimageAtTarget(ctx context.Context, cannonTraceProviderFunc CannonTraceProviderFunc, targetTraceIndex uint64, oracleDataValidator OracleDataValidator, uploadOracleData bool) {
	execDepth := g.splitGame.ExecDepth(ctx)
	provider, localContext, outputRootClaim := cannonTraceProviderFunc()

	pos := types.NewPosition(execDepth, new(big.Int).SetUint64(targetTraceIndex))
	g.require.Equal(targetTraceIndex, pos.TraceIndex(execDepth).Uint64())

	prestate, proof, oracleData, err := provider.GetStepData(ctx, pos)
	g.require.NoError(err, "Failed to get step data")
	g.require.NotNil(oracleData, "Should have had required preimage oracle data")
	oracleDataValidator(oracleData)

	if uploadOracleData {
		txCandidate, err := g.splitGame.Game.UpdateOracleTx(ctx, uint64(outputRootClaim.Index), oracleData)
		g.require.NoError(err, "failed to get oracle")
		transactions.RequireSendTx(g.t, ctx, g.client, txCandidate, g.privKey)
	}

	expectedPostState, err := provider.Get(ctx, pos)
	g.require.NoError(err, "Failed to get expected post state")

	vm, err := g.splitGame.Game.Vm(ctx)
	g.require.NoError(err, "Failed to get VM address")

	abi, err := bindings.MIPSMetaData.GetAbi()
	g.require.NoError(err, "Failed to load MIPS ABI")
	caller := batching.NewMultiCaller(g.client.Client(), batching.DefaultBatchSize)
	result, err := caller.SingleCall(ctx, rpcblock.Latest, &batching.ContractCall{
		Abi:    abi,
		Addr:   vm.Addr(),
		Method: "step",
		Args: []interface{}{
			prestate, proof, localContext,
		},
		From: g.splitGame.Addr,
	})
	g.require.NoError(err, "Failed to call step")
	actualPostState := result.GetBytes32(0)
	g.require.Equal(expectedPostState, common.Hash(actualPostState))
}

type CannonTraceProviderFunc func() (*cannon.CannonTraceProviderForTest, common.Hash, *ClaimHelper)

// NewMemoizedCannonTraceProvider returns a function that will generate a cannon trace provider once, memoize it
// and return the same trace provider on subsequent calls
func (g *CannonHelper) NewMemoizedCannonTraceProvider(ctx context.Context, l2Node string, outputRootClaim *ClaimHelper, options ...challenger.Option) CannonTraceProviderFunc {
	var provider *cannon.CannonTraceProviderForTest
	var localContext common.Hash
	return func() (*cannon.CannonTraceProviderForTest, common.Hash, *ClaimHelper) {
		if provider == nil {
			provider, localContext = g.createCannonTraceProvider(ctx, l2Node, outputRootClaim, options...)
		}
		return provider, localContext, outputRootClaim
	}
}

func (g *CannonHelper) createCannonTraceProvider(ctx context.Context, l2Node string, outputRootClaim *ClaimHelper, options ...challenger.Option) (*cannon.CannonTraceProviderForTest, common.Hash) {
	splitDepth := g.splitGame.SplitDepth(ctx)
	g.require.EqualValues(outputRootClaim.Depth(), splitDepth+1, "outputRootClaim must be the root of an execution game")

	logger := testlog.Logger(g.t, log.LevelInfo).New("role", "CannonTraceProvider", "game", g.splitGame.Addr)
	opt := g.defaultChallengerOptions()
	opt = append(opt, options...)
	cfg := challenger.NewChallengerConfig(g.t, g.system, l2Node, opt...)

	l2Client := g.system.NodeClient(l2Node)

	prestateBlock, poststateBlock, err := g.splitGame.Game.GetGameRange(ctx)
	g.require.NoError(err, "Failed to load block range")
	rollupClient := g.system.RollupClient(l2Node)
	prestateProvider := outputs.NewPrestateProvider(rollupClient, prestateBlock)
	l1Head := g.splitGame.GetL1Head(ctx)
	outputProvider := outputs.NewTraceProvider(logger, prestateProvider, rollupClient, l2Client, l1Head, splitDepth, prestateBlock, poststateBlock)

	var localContext common.Hash
	selector := split.NewSplitProviderSelector(outputProvider, splitDepth, func(ctx context.Context, depth types.Depth, pre types.Claim, post types.Claim) (types.TraceProvider, error) {
		agreed, disputed, err := outputs.FetchProposals(ctx, outputProvider, pre, post)
		g.require.NoError(err)
		g.t.Logf("Using trace between blocks %v and %v\n", agreed.L2BlockNumber, disputed.L2BlockNumber)
		localInputs, err := utils.FetchLocalInputsFromProposals(ctx, l1Head.Hash, l2Client, agreed, disputed)
		g.require.NoError(err, "Failed to fetch local inputs")
		localContext = split.CreateLocalContext(pre, post)
		dir := filepath.Join(cfg.Datadir, "cannon-trace")
		subdir := filepath.Join(dir, localContext.Hex())
		return cannon.NewTraceProviderForTest(logger, metrics.NoopMetrics.ToTypedVmMetrics(types.TraceTypeCannon.String()), cfg, localInputs, subdir, g.splitGame.MaxDepth(ctx)-splitDepth-1), nil
	})

	claims, err := g.splitGame.Game.GetAllClaims(ctx, rpcblock.Latest)
	g.require.NoError(err)
	game := types.NewGameState(claims, g.splitGame.MaxDepth(ctx))

	provider, err := selector(ctx, game, game.Claims()[outputRootClaim.ParentIndex], outputRootClaim.Position)
	g.require.NoError(err)
	translatingProvider := provider.(*trace.TranslatingProvider)
	return translatingProvider.Original().(*cannon.CannonTraceProviderForTest), localContext
}

// traceBisection performs a bisection of the trace to the desired targetTraceIndex
func traceBisection(
	t *testing.T,
	ctx context.Context,
	claim *ClaimHelper,
	splitDepth types.Depth,
	execDepth types.Depth,
	targetTraceIndex uint64,
	provider *cannon.CannonTraceProviderForTest,
) *ClaimHelper {
	execClaimPosition, err := claim.Position.RelativeToAncestorAtDepth(splitDepth + 1)
	require.NoError(t, err)

	claimTraceIndex := execClaimPosition.TraceIndex(execDepth).Uint64()
	t.Logf("Bisecting: Into targetTraceIndex %v: claimIndex=%v at depth=%v. claimPosition=%v execClaimPosition=%v claimTraceIndex=%v",
		targetTraceIndex, claim.Index, claim.Depth(), claim.Position, execClaimPosition, claimTraceIndex)

	// We always want to position ourselves such that the challenger generates proofs for the targetTraceIndex as prestate
	if execClaimPosition.Depth() == execDepth-1 {
		if execClaimPosition.TraceIndex(execDepth).Uint64() == targetTraceIndex {
			newPosition := execClaimPosition.Attack()
			correct, err := provider.Get(ctx, newPosition)
			require.NoError(t, err)
			t.Logf("Bisecting: Attack correctly for step at newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
			return claim.Attack(ctx, correct)
		} else if execClaimPosition.TraceIndex(execDepth).Uint64() > targetTraceIndex {
			t.Logf("Bisecting: Attack incorrectly for step")
			return claim.Attack(ctx, common.Hash{0xdd})
		} else if execClaimPosition.TraceIndex(execDepth).Uint64()+1 == targetTraceIndex {
			t.Logf("Bisecting: Defend incorrectly for step")
			return claim.Defend(ctx, common.Hash{0xcc})
		} else {
			newPosition := execClaimPosition.Defend()
			correct, err := provider.Get(ctx, newPosition)
			require.NoError(t, err)
			t.Logf("Bisecting: Defend correctly for step at newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
			return claim.Defend(ctx, correct)
		}
	}

	// Attack or Defend depending on whether the claim we're responding to is to the left or right of the trace index
	// Induce the honest challenger to attack or defend depending on whether our new position will be to the left or right of the trace index
	if execClaimPosition.TraceIndex(execDepth).Uint64() < targetTraceIndex && claim.Depth() != splitDepth+1 {
		newPosition := execClaimPosition.Defend()
		if newPosition.TraceIndex(execDepth).Uint64() < targetTraceIndex {
			t.Logf("Bisecting: Defend correct. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
			correct, err := provider.Get(ctx, newPosition)
			require.NoError(t, err)
			return claim.Defend(ctx, correct)
		} else {
			t.Logf("Bisecting: Defend incorrect. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
			return claim.Defend(ctx, common.Hash{0xaa})
		}
	} else {
		newPosition := execClaimPosition.Attack()
		if newPosition.TraceIndex(execDepth).Uint64() < targetTraceIndex {
			t.Logf("Bisecting: Attack correct. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
			correct, err := provider.Get(ctx, newPosition)
			require.NoError(t, err)
			return claim.Attack(ctx, correct)
		} else {
			t.Logf("Bisecting: Attack incorrect. newPosition=%v execIndexAtDepth=%v", newPosition, newPosition.TraceIndex(execDepth))
			return claim.Attack(ctx, common.Hash{0xbb})
		}
	}
}
