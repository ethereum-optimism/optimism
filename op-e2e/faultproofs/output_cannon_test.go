package faultproofs

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/utils"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-e2e/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame/preimage"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	oppreimage "github.com/ethereum-optimism/optimism/op-preimage"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestOutputCannonGame(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonGame)
}

func testOutputCannonGame(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 4, common.Hash{0x01})
	arena := createOutputGameArena(t, sys, game)
	testCannonGame(t, ctx, arena, &game.SplitGameHelper)
}

func TestOutputCannon_ChallengeAllZeroClaim(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonChallengeAllZeroClaim)
}

func testOutputCannonChallengeAllZeroClaim(t *testing.T, allocType config.AllocType) {
	// The dishonest actor always posts claims with all zeros.
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 3, common.Hash{})
	arena := createOutputGameArena(t, sys, game)
	testCannonChallengeAllZeroClaim(t, ctx, arena, &game.SplitGameHelper)
}

func TestOutputCannon_PublishCannonRootClaim(t *testing.T) {
	type TestCase struct {
		disputeL2BlockNumber uint64
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("Dispute_%v_%v", test.disputeL2BlockNumber, vm)
	}
	tests := []TestCase{
		{7}, // Post-state output root is invalid
		{8}, // Post-state output root is valid
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		ctx := context.Background()
		sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))

		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
		game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", test.disputeL2BlockNumber, common.Hash{0x01})
		game.DisputeLastBlock(ctx)
		game.LogGameData(ctx)

		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

		splitDepth := game.SplitDepth(ctx)
		game.WaitForClaimAtDepth(ctx, splitDepth+1)
	}, WithTestName(testName))
}

func TestOutputCannonDisputeGame(t *testing.T) {
	type TestCase struct {
		name             string
		defendClaimDepth types.Depth
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", test.name, vm)
	}
	tests := []TestCase{
		{"StepFirst", 0},
		{"StepMiddle", 28},
		{"StepInExtension", 1},
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		ctx := context.Background()
		sys, l1Client := StartFaultDisputeSystem(t, WithAllocType(allocType))
		t.Cleanup(sys.Close)

		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
		game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
		require.NotNil(t, game)
		game.LogGameData(ctx)

		outputClaim := game.DisputeLastBlock(ctx)
		splitDepth := game.SplitDepth(ctx)

		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

		game.DefendClaim(
			ctx,
			outputClaim,
			func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				if claim.Depth()+1 == splitDepth+test.defendClaimDepth {
					return claim.Defend(ctx, common.Hash{byte(claim.Depth())})
				} else {
					return claim.Attack(ctx, common.Hash{byte(claim.Depth())})
				}
			})

		sys.TimeTravelClock.AdvanceTime(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, l1Client))

		game.LogGameData(ctx)
		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
	}, WithTestName(testName))
}

func TestOutputCannonDefendStep(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonDefendStep)
}

func testOutputCannonDefendStep(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
	arena := createOutputGameArena(t, sys, game)
	testCannonDefendStep(t, ctx, arena, &game.SplitGameHelper)
}

func TestOutputCannonStepWithLargePreimage(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonStepWithLargePreimage)
}

func testOutputCannonStepWithLargePreimage(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithBatcherStopped(), WithAllocType(allocType))
	t.Cleanup(sys.Close)

	// Manually send a tx from the correct batcher key to the batcher input with very large (invalid) data
	// This forces op-program to load a large preimage.
	sys.BatcherHelper().SendLargeInvalidBatch(ctx)

	require.NoError(t, sys.BatchSubmitter.Start(ctx))

	safeHead, err := wait.ForNextSafeBlock(ctx, sys.NodeClient("sequencer"))
	require.NoError(t, err, "Batcher should resume submitting valid batches")

	l2BlockNumber := safeHead.NumberU64()
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Dispute any block - it will have to read the L1 batches to see if the block is reached
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", l2BlockNumber, common.Hash{0x01, 0xaa})
	require.NotNil(t, game)
	outputRootClaim := game.DisputeBlock(ctx, l2BlockNumber)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	// Wait for the honest challenger to dispute the outputRootClaim.
	// This creates a root of an execution game that we challenge by
	// coercing a step at a preimage trace index.
	outputRootClaim = outputRootClaim.WaitForCounterClaim(ctx)

	game.LogGameData(ctx)
	// Now the honest challenger is positioned as the defender of the
	// execution game. We then move to challenge it to induce a large preimage load.
	sender := sys.Cfg.Secrets.Addresses().Alice
	preimageLoadCheck := game.CreateStepLargePreimageLoadCheck(ctx, sender)
	providerFunc := game.NewMemoizedCannonTraceProvider(ctx, "sequencer", outputRootClaim, challenger.WithPrivKey(sys.Cfg.Secrets.Alice))
	game.ChallengeToPreimageLoad(ctx, providerFunc, utils.PreimageLargerThan(preimage.MinPreimageSize), preimageLoadCheck, false)
	// The above method already verified the image was uploaded and step called successfully
	// So we don't waste time resolving the game - that's tested elsewhere.
}

func TestOutputCannonStepWithPreimage_nonExistingPreimage(t *testing.T) {
	type TestCase struct {
		name         string
		preimageType oppreimage.KeyType
		opts         []disputegame.FindPreimageStepOpt
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", test.name, vm)
	}
	tests := []TestCase{
		{name: "keccak", preimageType: oppreimage.Keccak256KeyType},
		// Sha256 preimages are relatively rare, so allow fallback to even step to avoid flakes
		{name: "sha256", preimageType: oppreimage.Sha256KeyType, opts: []disputegame.FindPreimageStepOpt{disputegame.AllowEvenFallback()}},
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, testcase TestCase) {
		conf := utils.PreimageOptConfigForType(oppreimage.Keccak256KeyType)
		testPreimageStep(t, allocType, conf, false, testcase.opts...)
	}, WithTestName(testName))
}

func TestOutputCannonStepWithPreimage_nonExistingBlobPreimage(t *testing.T) {
	type TestCase struct {
		blobOffset    uint32
		blobSkipCount int
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("non-existing preimage-blob-%v skip-%v [%v]", strconv.Itoa(int(test.blobOffset)), test.blobSkipCount, vm)
	}

	testCases := make([]TestCase, 0)
	blobOffsets := []uint32{0, 8, 16, 24, 32}
	skipCounts := []int{0, 1, 2, 11}
	for _, offset := range blobOffsets {
		for _, skip := range skipCounts {
			testCases = append(testCases, TestCase{blobOffset: offset, blobSkipCount: skip})
		}
	}

	RunTestsAcrossVmTypes(t, testCases, func(t *testing.T, allocType config.AllocType, testcase TestCase) {
		conf := utils.PreimageOptConfigForType(oppreimage.BlobKeyType)
		conf.Offset = testcase.blobOffset

		// In order to target non-zero blob field indices, skip some preimage load steps.
		// Because field elements are retrieved sequentially, this should ensure we advance to
		// a field element at an index >= skip
		testPreimageStep(t, allocType, conf, false, disputegame.SkipNPreimageLoads(testcase.blobSkipCount))
	}, WithTestName(testName))
}

func TestOutputCannonStepWithPreimage_existingPreimage(t *testing.T) {
	// Only test pre-existing images with one type to save runtime
	RunTestAcrossVmTypes(t, func(t *testing.T, allocType config.AllocType) {
		conf := utils.PreimageOptConfigForType(oppreimage.Keccak256KeyType)
		testPreimageStep(t, allocType, conf, true)
	})
}

func testPreimageStep(t *testing.T, allocType config.AllocType, preimageOptConfig utils.PreimageOptConfig, preloadPreimage bool, opts ...disputegame.FindPreimageStepOpt) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithBlobBatches(), WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
	require.NotNil(t, game)
	outputRootClaim := game.DisputeLastBlock(ctx)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	// Wait for the honest challenger to dispute the outputRootClaim. This creates a root of an execution game that we challenge by coercing
	// a step at a preimage trace index.
	outputRootClaim = outputRootClaim.WaitForCounterClaim(ctx)

	// Now the honest challenger is positioned as the defender of the execution game
	// We then move to challenge it to induce a preimage load
	// Check that the preimage is loaded into the oracle with data matching our expectation
	getExpectedData := func(p *types.PreimageOracleData) (bool, [32]byte) { return true, game.GetPreimageAtOffset(p) }
	preimageLoadCheck := game.CreateStepPreimageLoadStrictCheck(ctx, getExpectedData)
	// We need the honest challenger to step-defend the STF from A -> B such that A loads the preimage
	// The ChallengeToPreimageLoadAtTarget method will induce a step-defend on odd numbered trace index from the honest challenger.
	providerFunc := game.NewMemoizedCannonTraceProvider(ctx, "sequencer", outputRootClaim, challenger.WithPrivKey(sys.Cfg.Secrets.Alice))
	step := game.FindOddStepForPreimageLoad(ctx, providerFunc, preimageOptConfig, opts...)
	game.ChallengeToPreimageLoadAtTarget(ctx, providerFunc, step, preimageLoadCheck, preloadPreimage)
	// The above method already verified the image was uploaded and step called successfully
	// So we don't waste time resolving the game - that's tested elsewhere.

	// Finally, validate that we can manually invoke step at this point in the game and produce the expected post-state
	game.VerifyPreimageAtTarget(ctx, providerFunc, step, game.GetOracleKeyPrefixValidator(preimageOptConfig.KeyPrefix), false)
}

func TestOutputCannonStepWithKZGPointEvaluation(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonStepWithKzgPointEvaluation)
}

func testOutputCannonStepWithKzgPointEvaluation(t *testing.T, allocType config.AllocType) {
	testPreimageStep := func(t *testing.T, preloadPreimage bool) {
		ctx := context.Background()
		sys, _ := StartFaultDisputeSystem(t, WithEcotone(), WithAllocType(allocType))
		t.Cleanup(sys.Close)

		// NOTE: Flake prevention
		// Ensure that the L1 origin including the point eval tx isn't on the genesis epoch.
		safeBlock, err := wait.ForNextSafeBlock(ctx, sys.NodeClient("sequencer"))
		require.NoError(t, err)
		require.NoError(t, wait.ForSafeBlock(ctx, sys.RollupClient("sequencer"), safeBlock.NumberU64()+3))

		receipt := SendKZGPointEvaluationTx(t, sys, "sequencer", sys.Cfg.Secrets.Alice)
		precompileBlock := receipt.BlockNumber
		t.Logf("KZG Point Evaluation block number: %d", precompileBlock)

		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
		game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", precompileBlock.Uint64(), common.Hash{0x01, 0xaa})
		require.NotNil(t, game)
		outputRootClaim := game.DisputeLastBlock(ctx)
		game.LogGameData(ctx)

		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

		// Wait for the honest challenger to dispute the outputRootClaim. This creates a root of an execution game that we challenge by coercing
		// a step at a preimage trace index.
		outputRootClaim = outputRootClaim.WaitForCounterClaim(ctx)

		// Now the honest challenger is positioned as the defender of the execution game
		// We then move to challenge it to induce a preimage load
		preimageLoadCheck := game.CreateStepPreimageLoadCheck(ctx)
		providerFunc := game.NewMemoizedCannonTraceProvider(ctx, "sequencer", outputRootClaim, challenger.WithPrivKey(sys.Cfg.Secrets.Alice))
		game.ChallengeToPreimageLoad(ctx, providerFunc, utils.FirstPrecompilePreimageLoad(), preimageLoadCheck, preloadPreimage)
		// The above method already verified the image was uploaded and step called successfully
		// So we don't waste time resolving the game - that's tested elsewhere.
	}

	t.Run("non-existing preimage", func(t *testing.T) {
		testPreimageStep(t, false)
	})
	t.Run("preimage already exists", func(t *testing.T) {
		testPreimageStep(t, true)
	})
}

func TestOutputCannonProposedOutputRootValid(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonProposedOutputRootValid_AttackWithCorrectTrace)
}

func testOutputCannonProposedOutputRootValid_AttackWithCorrectTrace(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", 1)
	arena := createOutputGameArena(t, sys, game)
	testCannonProposalValid_AttackWithCorrectTrace(t, ctx, arena, &game.SplitGameHelper)
}

func TestOutputCannonProposedOutputRootValid_DefendWithCorrectTrace(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonProposedOutputRootValid_DefendWithCorrectTrace)
}

func testOutputCannonProposedOutputRootValid_DefendWithCorrectTrace(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", 1)
	arena := createOutputGameArena(t, sys, game)
	testCannonProposalValid_DefendWithCorrectTrace(t, ctx, arena, &game.SplitGameHelper)
}

func TestOutputCannonPoisonedPostState(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonPoisonedPostState)
}

func testOutputCannonPoisonedPostState(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0xaa})
	arena := createOutputGameArena(t, sys, game)
	testCannonPoisonedPostState(t, ctx, arena, &game.SplitGameHelper)
}

func TestDisputeOutputRootBeyondProposedBlock_ValidOutputRoot(t *testing.T) {
	RunTestAcrossVmTypes(t, testDisputeOutputRootBeyondProposedBlockValidOutputRoot)
}

func testDisputeOutputRootBeyondProposedBlockValidOutputRoot(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", 1)
	arena := createOutputGameArena(t, sys, game)
	testDisputeRootBeyondProposedBlockValidOutputRoot(t, ctx, arena, &game.SplitGameHelper)
}

func TestDisputeOutputRootBeyondProposedBlock_InvalidOutputRoot(t *testing.T) {
	RunTestAcrossVmTypes(t, testDisputeOutputRootBeyondProposedBlockInvalidOutputRoot)
}

func testDisputeOutputRootBeyondProposedBlockInvalidOutputRoot(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0xaa})
	arena := createOutputGameArena(t, sys, game)
	testDisputeRootBeyondProposedBlockInvalidOutputRoot(t, ctx, arena, &game.SplitGameHelper)
}

func TestTestDisputeOutputRoot_ChangeClaimedOutputRoot(t *testing.T) {
	RunTestAcrossVmTypes(t, testTestDisputeOutputRootChangeClaimedOutputRoot)
}

func testTestDisputeOutputRootChangeClaimedOutputRoot(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Root claim is dishonest
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0xaa})
	arena := createOutputGameArena(t, sys, game)
	testDisputeRootChangeClaimedRoot(t, ctx, arena, &game.SplitGameHelper)
}

func TestInvalidateUnsafeProposal(t *testing.T) {
	ctx := context.Background()

	type TestCase struct {
		name     string
		strategy func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", vm, test.name)
	}

	tests := []TestCase{
		{
			name: "Attack",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.AttackClaim(ctx, parent)
			},
		},
		{
			name: "Defend",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.DefendClaim(ctx, parent)
			},
		},
		{
			name: "Counter",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.CounterClaim(ctx, parent)
			},
		},
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		sys, l1Client := StartFaultDisputeSystem(t, WithSequencerWindowSize(100000), WithBatcherStopped(), WithAllocType(allocType))
		t.Cleanup(sys.Close)

		blockNum := uint64(1)
		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
		// Root claim is _dishonest_ because the required data is not available on L1
		game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", blockNum, disputegame.WithUnsafeProposal())

		correctTrace := game.CreateHonestActor(ctx, "sequencer", disputegame.WithPrivKey(sys.Cfg.Secrets.Alice))

		// Start the honest challenger
		game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

		game.DefendClaim(ctx, game.RootClaim(ctx), func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			if parent.IsBottomGameRoot(ctx) {
				return correctTrace.AttackClaim(ctx, parent)
			}
			return test.strategy(correctTrace, parent)
		})

		// Time travel past when the game will be resolvable.
		sys.TimeTravelClock.AdvanceTime(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, l1Client))

		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
		game.LogGameData(ctx)
	}, WithTestName(testName))
}

func TestInvalidateProposalForFutureBlock(t *testing.T) {
	ctx := context.Background()
	type TestCase struct {
		name     string
		strategy func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper
	}
	testName := func(vm string, test TestCase) string {
		return fmt.Sprintf("%v-%v", vm, test.name)
	}

	tests := []TestCase{
		{
			name: "Attack",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.AttackClaim(ctx, parent)
			},
		},
		{
			name: "Defend",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.DefendClaim(ctx, parent)
			},
		},
		{
			name: "Counter",
			strategy: func(correctTrace *disputegame.OutputHonestHelper, parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
				return correctTrace.CounterClaim(ctx, parent)
			},
		},
	}

	RunTestsAcrossVmTypes(t, tests, func(t *testing.T, allocType config.AllocType, test TestCase) {
		sys, l1Client := StartFaultDisputeSystem(t, WithSequencerWindowSize(100000), WithAllocType(allocType))
		t.Cleanup(sys.Close)

		farFutureBlockNum := uint64(10_000_000)
		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
		// Root claim is _dishonest_ because the required data is not available on L1
		game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", farFutureBlockNum, common.Hash{0xaa}, disputegame.WithFutureProposal())

		correctTrace := game.CreateHonestActor(ctx, "sequencer", disputegame.WithPrivKey(sys.Cfg.Secrets.Alice))

		// Start the honest challenger
		game.StartChallenger(ctx, "Honest", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

		game.DefendClaim(ctx, game.RootClaim(ctx), func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			if parent.IsBottomGameRoot(ctx) {
				return correctTrace.AttackClaim(ctx, parent)
			}
			return test.strategy(correctTrace, parent)
		})

		// Time travel past when the game will be resolvable.
		sys.TimeTravelClock.AdvanceTime(game.MaxClockDuration(ctx))
		require.NoError(t, wait.ForNextBlock(ctx, l1Client))

		game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
		game.LogGameData(ctx)
	}, WithTestName(testName))
}

func TestInvalidateCorrectProposalFutureBlock(t *testing.T) {
	RunTestAcrossVmTypes(t, testInvalidateCorrectProposalFutureBlock)
}

func testInvalidateCorrectProposalFutureBlock(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	// Spin up the system without the batcher so the safe head doesn't advance
	sys, l1Client := StartFaultDisputeSystem(t, WithBatcherStopped(), WithSequencerWindowSize(100000), WithAllocType(allocType))
	t.Cleanup(sys.Close)

	// Create a dispute game factory helper.
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)

	// No batches submitted so safe head is genesis
	output, err := sys.RollupClient("sequencer").OutputAtBlock(ctx, 0)
	require.NoError(t, err, "Failed to get output at safe head")
	// Create a dispute game with an output root that is valid at `safeHead`, but that claims to correspond to block
	// `safeHead.Number + 10000`. This is dishonest, because this block does not exist yet.
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 10_000, common.Hash(output.OutputRoot), disputegame.WithFutureProposal())

	// Start the honest challenger.
	game.StartChallenger(ctx, "Honest", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

	game.WaitForL2BlockNumberChallenged(ctx)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	// The game should resolve as `CHALLENGER_WINS` always, because the root claim signifies a claim that does not exist
	// yet in the L2 chain.
	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
	game.LogGameData(ctx)
}

func TestOutputCannonHonestSafeTraceExtension_ValidRoot(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonHonestSafeTraceExtensionValidRoot)
}

func testOutputCannonHonestSafeTraceExtensionValidRoot(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, l1Client := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	// Wait for there to be there are safe L2 blocks past the claimed safe head that have data available on L1 within
	// the commitment stored in the dispute game.
	safeHeadNum := uint64(3)
	require.NoError(t, wait.ForSafeBlock(ctx, sys.RollupClient("sequencer"), safeHeadNum))

	// Create a dispute game with an honest claim
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", safeHeadNum-1)
	require.NotNil(t, game)

	// Create a correct trace actor with an honest trace extending to L2 block #4
	correctTrace := game.CreateHonestActor(ctx, "sequencer", disputegame.WithPrivKey(sys.Cfg.Secrets.Mallory))

	// Create a correct trace actor with an honest trace extending to L2 block #5
	// Notably, L2 block #5 is a valid block within the safe chain, and the data required to reproduce it
	// will be committed to within the L1 head of the dispute game.
	correctTracePlus1 := game.CreateHonestActor(ctx, "sequencer",
		disputegame.WithPrivKey(sys.Cfg.Secrets.Mallory),
		disputegame.WithClaimedL2BlockNumber(safeHeadNum))

	// Start the honest challenger. They will defend the root claim.
	game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	claim := game.RootClaim(ctx)
	game.ChallengeClaim(ctx, claim, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		// Have to disagree with the root claim - we're trying to invalidate a valid output root
		if parent.IsRootClaim() {
			return parent.Attack(ctx, common.Hash{0xdd})
		}
		return correctTracePlus1.CounterClaim(ctx, parent)
	}, func(parentClaimIdx int64) {
		correctTrace.StepFails(ctx, parentClaimIdx, true)
		correctTrace.StepFails(ctx, parentClaimIdx, false)
		correctTracePlus1.StepFails(ctx, parentClaimIdx, true)
		correctTracePlus1.StepFails(ctx, parentClaimIdx, false)
	})
	game.LogGameData(ctx)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, gameTypes.GameStatusDefenderWon)
}

func TestOutputCannonHonestSafeTraceExtension_InvalidRoot(t *testing.T) {
	RunTestAcrossVmTypes(t, testOutputCannonHonestSafeTraceExtensionInvalidRoot)
}

func testOutputCannonHonestSafeTraceExtensionInvalidRoot(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, l1Client := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	// Wait for there to be there are safe L2 blocks past the claimed safe head that have data available on L1 within
	// the commitment stored in the dispute game.
	safeHeadNum := uint64(2)
	require.NoError(t, wait.ForSafeBlock(ctx, sys.RollupClient("sequencer"), safeHeadNum))

	// Create a dispute game with a dishonest claim @ L2 block #4
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", safeHeadNum-1, common.Hash{0xCA, 0xFE})
	require.NotNil(t, game)

	// Create a correct trace actor with an honest trace extending to L2 block #5
	// Notably, L2 block #5 is a valid block within the safe chain, and the data required to reproduce it
	// will be committed to within the L1 head of the dispute game.
	correctTracePlus1 := game.CreateHonestActor(ctx, "sequencer",
		disputegame.WithPrivKey(sys.Cfg.Secrets.Mallory),
		disputegame.WithClaimedL2BlockNumber(safeHeadNum))

	// Start the honest challenger. They will challenge the root claim.
	game.StartChallenger(ctx, "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	claim := game.RootClaim(ctx)
	game.DefendClaim(ctx, claim, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		return correctTracePlus1.CounterClaim(ctx, parent)
	})

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
}

func TestAgreeFirstBlockWithOriginOf1(t *testing.T) {
	RunTestAcrossVmTypes(t, testAgreeFirstBlockWithOriginOf1)
}

func testAgreeFirstBlockWithOriginOf1(t *testing.T, allocType config.AllocType) {
	ctx := context.Background()
	sys, _ := StartFaultDisputeSystem(t, WithAllocType(allocType))
	t.Cleanup(sys.Close)

	rollupClient := sys.RollupClient("sequencer")
	blockNum := uint64(0)
	limit := uint64(100)
	for ; blockNum <= limit; blockNum++ {
		require.NoError(t, wait.ForBlock(ctx, sys.NodeClient("sequencer"), blockNum))
		output, err := rollupClient.OutputAtBlock(ctx, blockNum)
		require.NoError(t, err)
		if output.BlockRef.L1Origin.Number == 1 {
			break
		}
	}
	require.Less(t, blockNum, limit)

	// Create a dispute game with a dishonest claim @ L2 block #4
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	// Make the agreed block the first one with L1 origin of block 1 so the claim is blockNum+1
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", blockNum+1, common.Hash{0xCA, 0xFE})
	require.NotNil(t, game)
	outputRootClaim := game.DisputeLastBlock(ctx)
	game.LogGameData(ctx)

	honestChallenger := game.StartChallenger(ctx, "HonestActor", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	// Wait for the honest challenger to dispute the outputRootClaim. This creates a root of an execution game that we challenge by coercing
	// a step at a preimage trace index.
	outputRootClaim = outputRootClaim.WaitForCounterClaim(ctx)
	game.LogGameData(ctx)

	// Should claim output root is invalid, but actually panics.
	outputRootClaim.RequireInvalidStatusCode()
	// The above method already verified the image was uploaded and step called successfully
	// So we don't waste time resolving the game - that's tested elsewhere.
	require.NoError(t, honestChallenger.Close())
}
