package op_e2e

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	l2oo2 "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/l2oo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestMultipleCannonGames(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	// Start a challenger with the correct alphabet trace
	challenger := gameFactory.StartChallenger(ctx, sys.NodeEndpoint("l1"), "TowerDefense",
		challenger.WithCannon(t, sys.RollupConfig, sys.L2GenesisCfg, sys.NodeEndpoint("sequencer")),
		challenger.WithPrivKey(sys.cfg.Secrets.Alice),
		challenger.WithAgreeProposedOutput(true),
	)

	game1 := gameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
	game2 := gameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xbb})

	game1.WaitForClaimCount(ctx, 2)
	game2.WaitForClaimCount(ctx, 2)

	game1Claim := game1.GetClaimValue(ctx, 1)
	game2Claim := game2.GetClaimValue(ctx, 1)
	require.NotEqual(t, game1Claim, game2Claim, "games should have different cannon traces")

	// Check that the helper finds the game directories correctly
	challenger.VerifyGameDataExists(game1, game2)

	// Push both games down to the step function
	maxDepth := game1.MaxDepth(ctx)
	for claimCount := int64(1); claimCount <= maxDepth; {
		// Challenger should respond to both games
		claimCount++
		game1.WaitForClaimCount(ctx, claimCount)
		game2.WaitForClaimCount(ctx, claimCount)

		// Progress both games
		game1.Defend(ctx, claimCount-1, common.Hash{0xaa})
		game2.Defend(ctx, claimCount-1, common.Hash{0xaa})
		claimCount++
	}

	game1.WaitForClaimAtMaxDepth(ctx, true)
	game2.WaitForClaimAtMaxDepth(ctx, true)

	gameDuration := game1.GameDuration(ctx)
	sys.TimeTravelClock.AdvanceTime(gameDuration)
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game1.WaitForInactivity(ctx, 10, true)
	game2.WaitForInactivity(ctx, 10, true)
	game1.LogGameData(ctx)
	game2.LogGameData(ctx)
	require.EqualValues(t, disputegame.StatusChallengerWins, game1.Status(ctx))
	require.EqualValues(t, disputegame.StatusChallengerWins, game2.Status(ctx))

	// Check that the game directories are removed
	challenger.WaitForGameDataDeletion(ctx, game1, game2)
}

func TestMultipleGameTypes(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	// Start a challenger with both cannon and alphabet support
	gameFactory.StartChallenger(ctx, sys.NodeEndpoint("l1"), "TowerDefense",
		challenger.WithCannon(t, sys.RollupConfig, sys.L2GenesisCfg, sys.NodeEndpoint("sequencer")),
		challenger.WithAlphabet(disputegame.CorrectAlphabet),
		challenger.WithPrivKey(sys.cfg.Secrets.Alice),
		challenger.WithAgreeProposedOutput(true),
	)

	game1 := gameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
	game2 := gameFactory.StartAlphabetGame(ctx, "xyzabc")

	// Wait for the challenger to respond to both games
	game1.WaitForClaimCount(ctx, 2)
	game2.WaitForClaimCount(ctx, 2)
	game1Response := game1.GetClaimValue(ctx, 1)
	game2Response := game2.GetClaimValue(ctx, 1)
	// The alphabet game always posts the same traces, so if they're different they can't both be from the alphabet.
	require.NotEqual(t, game1Response, game2Response, "should have posted different claims")
	// Now check they aren't both just from different cannon games by confirming the alphabet value.
	correctAlphabet := alphabet.NewTraceProvider(disputegame.CorrectAlphabet, uint64(game2.MaxDepth(ctx)))
	expectedClaim, err := correctAlphabet.Get(ctx, types.NewPositionFromGIndex(big.NewInt(1)).Attack())
	require.NoError(t, err)
	require.Equal(t, expectedClaim, game2Response)
	// We don't confirm the cannon value because generating the correct claim is expensive
	// Just being different is enough to confirm the challenger isn't just playing two alphabet games incorrectly
}

func TestChallengerCompleteDisputeGame(t *testing.T) {
	InitParallel(t)

	tests := []struct {
		name              string
		rootClaimAlphabet string
		otherAlphabet     string
		expectedResult    disputegame.Status
		expectStep        bool
	}{
		{
			name:              "ChallengerWins_DefenseStep",
			rootClaimAlphabet: "abcdexyz",
			otherAlphabet:     disputegame.CorrectAlphabet,
			expectedResult:    disputegame.StatusChallengerWins,
			expectStep:        true,
		},
		{
			name:              "DefenderWins_DefenseStep",
			rootClaimAlphabet: disputegame.CorrectAlphabet,
			otherAlphabet:     "abcdexyz",
			expectedResult:    disputegame.StatusDefenderWins,
			expectStep:        false,
		},
		{
			name:              "ChallengerWins_AttackStep",
			rootClaimAlphabet: "abcdefghzyx",
			otherAlphabet:     disputegame.CorrectAlphabet,
			expectedResult:    disputegame.StatusChallengerWins,
			expectStep:        true,
		},
		{
			name:              "DefenderWins_AttackStep",
			rootClaimAlphabet: disputegame.CorrectAlphabet,
			otherAlphabet:     "abcdexyz",
			expectedResult:    disputegame.StatusDefenderWins,
			expectStep:        false,
		},
		{
			name:              "DefenderIncorrectAtTraceZero",
			rootClaimAlphabet: "zyxwvut",
			otherAlphabet:     disputegame.CorrectAlphabet,
			expectedResult:    disputegame.StatusChallengerWins,
			expectStep:        true,
		},
		{
			name:              "ChallengerIncorrectAtTraceZero",
			rootClaimAlphabet: disputegame.CorrectAlphabet,
			otherAlphabet:     "zyxwvut",
			expectedResult:    disputegame.StatusDefenderWins,
			expectStep:        false,
		},
		{
			name:              "DefenderIncorrectAtLastTraceIndex",
			rootClaimAlphabet: "abcdefghijklmnoz",
			otherAlphabet:     disputegame.CorrectAlphabet,
			expectedResult:    disputegame.StatusChallengerWins,
			expectStep:        true,
		},
		{
			name:              "ChallengerIncorrectAtLastTraceIndex",
			rootClaimAlphabet: disputegame.CorrectAlphabet,
			otherAlphabet:     "abcdefghijklmnoz",
			expectedResult:    disputegame.StatusDefenderWins,
			expectStep:        false,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			InitParallel(t)

			ctx := context.Background()
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
			game := disputeGameFactory.StartAlphabetGame(ctx, test.rootClaimAlphabet)
			require.NotNil(t, game)
			gameDuration := game.GameDuration(ctx)

			game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "Defender",
				challenger.WithAgreeProposedOutput(false),
				challenger.WithPrivKey(sys.cfg.Secrets.Mallory),
			)

			game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "Challenger",
				// Agree with the proposed output, so disagree with the root claim
				challenger.WithAgreeProposedOutput(true),
				challenger.WithAlphabet(test.otherAlphabet),
				challenger.WithPrivKey(sys.cfg.Secrets.Alice),
			)

			// Wait for a claim at the maximum depth that has been countered to indicate we're ready to resolve the game
			game.WaitForClaimAtMaxDepth(ctx, test.expectStep)

			sys.TimeTravelClock.AdvanceTime(gameDuration)
			require.NoError(t, wait.ForNextBlock(ctx, l1Client))

			game.WaitForInactivity(ctx, 10, true)
			game.LogGameData(ctx)
			require.EqualValues(t, test.expectedResult, game.Status(ctx))
		})
	}
}

func TestChallengerCompleteExhaustiveDisputeGame(t *testing.T) {
	InitParallel(t)

	testCase := func(t *testing.T, isRootCorrect bool) {
		ctx := context.Background()
		sys, l1Client := startFaultDisputeSystem(t)
		t.Cleanup(sys.Close)

		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
		rootClaimedAlphabet := disputegame.CorrectAlphabet
		if !isRootCorrect {
			rootClaimedAlphabet = "abcdexyz"
		}
		game := disputeGameFactory.StartAlphabetGame(ctx, rootClaimedAlphabet)
		require.NotNil(t, game)
		gameDuration := game.GameDuration(ctx)

		// Start honest challenger
		game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "Challenger",
			challenger.WithAgreeProposedOutput(!isRootCorrect),
			challenger.WithAlphabet(disputegame.CorrectAlphabet),
			challenger.WithPrivKey(sys.cfg.Secrets.Alice),
			// Ensures the challenger responds to all claims before test timeout
			challenger.WithPollInterval(time.Millisecond*400),
		)

		// Start dishonest challenger
		dishonestHelper := game.CreateDishonestHelper(disputegame.CorrectAlphabet, 4, !isRootCorrect)
		dishonestHelper.ExhaustDishonestClaims(ctx)

		// Wait until we've reached max depth before checking for inactivity
		game.WaitForClaimAtDepth(ctx, int(game.MaxDepth(ctx)))

		// Wait for 4 blocks of no challenger responses. The challenger may still be stepping on invalid claims at max depth
		game.WaitForInactivity(ctx, 4, false)

		sys.TimeTravelClock.AdvanceTime(gameDuration)
		require.NoError(t, wait.ForNextBlock(ctx, l1Client))

		expectedStatus := disputegame.StatusChallengerWins
		if isRootCorrect {
			expectedStatus = disputegame.StatusDefenderWins
		}
		game.WaitForInactivity(ctx, 10, true)
		game.LogGameData(ctx)
		require.EqualValues(t, expectedStatus, game.Status(ctx))
	}

	t.Run("RootCorrect", func(t *testing.T) {
		InitParallel(t)
		testCase(t, true)
	})
	t.Run("RootIncorrect", func(t *testing.T) {
		InitParallel(t)
		testCase(t, false)
	})
}

func TestCannonDisputeGame(t *testing.T) {
	InitParallel(t)

	tests := []struct {
		name             string
		defendClaimCount int64
	}{
		{"StepFirst", 0},
		{"StepMiddle", 28},
		{"StepInExtension", 2},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			InitParallel(t)

			ctx := context.Background()
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
			game := disputeGameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
			require.NotNil(t, game)
			game.LogGameData(ctx)

			game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, sys.NodeEndpoint("l1"), sys.NodeEndpoint("sequencer"), "Challenger",
				// Agree with the proposed output, so disagree with the root claim
				challenger.WithAgreeProposedOutput(true),
				challenger.WithPrivKey(sys.cfg.Secrets.Alice),
			)

			game.DefendRootClaim(
				ctx,
				func(parentClaimIdx int64) {
					if parentClaimIdx+1 == test.defendClaimCount {
						game.Defend(ctx, parentClaimIdx, common.Hash{byte(parentClaimIdx)})
					} else {
						game.Attack(ctx, parentClaimIdx, common.Hash{byte(parentClaimIdx)})
					}
				})

			sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
			require.NoError(t, wait.ForNextBlock(ctx, l1Client))

			game.WaitForInactivity(ctx, 10, true)
			game.LogGameData(ctx)
			require.EqualValues(t, disputegame.StatusChallengerWins, game.Status(ctx))
		})
	}
}

func TestCannonDefendStep(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	game := disputeGameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
	require.NotNil(t, game)
	game.LogGameData(ctx)

	l1Endpoint := sys.NodeEndpoint("l1")
	l2Endpoint := sys.NodeEndpoint("sequencer")
	game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, "Challenger",
		// Agree with the proposed output, so disagree with the root claim
		challenger.WithAgreeProposedOutput(true),
		challenger.WithPrivKey(sys.cfg.Secrets.Alice),
	)

	correctTrace := game.CreateHonestActor(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Client, l1Endpoint, l2Endpoint,
		challenger.WithPrivKey(sys.cfg.Secrets.Mallory),
	)

	game.DefendRootClaim(ctx, func(parentClaimIdx int64) {
		// Post invalid claims for most steps to get down into the early part of the trace
		if parentClaimIdx < 27 {
			game.Attack(ctx, parentClaimIdx, common.Hash{byte(parentClaimIdx)})
		} else {
			// Post our own counter but using the correct hash in low levels to force a defense step
			correctTrace.Attack(ctx, parentClaimIdx)
		}
	})

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForInactivity(ctx, 10, true)
	game.LogGameData(ctx)
	require.EqualValues(t, disputegame.StatusChallengerWins, game.Status(ctx))
}

func TestCannonProposedOutputRootInvalid(t *testing.T) {
	InitParallel(t)
	// honestStepsFail attempts to perform both an attack and defend step using the correct trace.
	honestStepsFail := func(ctx context.Context, game *disputegame.CannonGameHelper, correctTrace *disputegame.HonestHelper, parentClaimIdx int64) {
		// Attack step should fail
		correctTrace.StepFails(ctx, parentClaimIdx, true)
		// Defending should fail too
		correctTrace.StepFails(ctx, parentClaimIdx, false)
	}
	tests := []struct {
		// name is the name of the test
		name string

		// outputRoot is the invalid output root to propose
		outputRoot common.Hash

		// performMove is called to respond to each claim posted by the honest op-challenger.
		// It should either attack or defend the claim at parentClaimIdx
		performMove func(ctx context.Context, game *disputegame.CannonGameHelper, correctTrace *disputegame.HonestHelper, parentClaimIdx int64)

		// performStep is called once the maximum game depth is reached. It should perform a step to counter the
		// claim at parentClaimIdx. Since the proposed output root is invalid, the step call should always revert.
		performStep func(ctx context.Context, game *disputegame.CannonGameHelper, correctTrace *disputegame.HonestHelper, parentClaimIdx int64)
	}{
		{
			name:       "AttackWithCorrectTrace",
			outputRoot: common.Hash{0xab},
			performMove: func(ctx context.Context, game *disputegame.CannonGameHelper, correctTrace *disputegame.HonestHelper, parentClaimIdx int64) {
				// Attack everything but oddly using the correct hash.
				correctTrace.Attack(ctx, parentClaimIdx)
			},
			performStep: honestStepsFail,
		},
		{
			name:       "DefendWithCorrectTrace",
			outputRoot: common.Hash{0xab},
			performMove: func(ctx context.Context, game *disputegame.CannonGameHelper, correctTrace *disputegame.HonestHelper, parentClaimIdx int64) {
				// Can only attack the root claim
				if parentClaimIdx == 0 {
					correctTrace.Attack(ctx, parentClaimIdx)
					return
				}
				// Otherwise, defend everything using the correct hash.
				correctTrace.Defend(ctx, parentClaimIdx)
			},
			performStep: honestStepsFail,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			InitParallel(t)

			ctx := context.Background()
			sys, l1Client, game, correctTrace := setupDisputeGameForInvalidOutputRoot(t, test.outputRoot)
			t.Cleanup(sys.Close)

			// Now maliciously play the game and it should be impossible to win
			game.ChallengeRootClaim(ctx,
				func(parentClaimIdx int64) {
					test.performMove(ctx, game, correctTrace, parentClaimIdx)
				},
				func(parentClaimIdx int64) {
					test.performStep(ctx, game, correctTrace, parentClaimIdx)
				})

			// Time travel past when the game will be resolvable.
			sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
			require.NoError(t, wait.ForNextBlock(ctx, l1Client))

			game.WaitForInactivity(ctx, 10, true)
			game.LogGameData(ctx)
			require.EqualValues(t, disputegame.StatusDefenderWins, game.Status(ctx))
		})
	}
}

func TestCannonPoisonedPostState(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	l1Endpoint := sys.NodeEndpoint("l1")
	l2Endpoint := sys.NodeEndpoint("sequencer")

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	game, correctTrace := disputeGameFactory.StartCannonGameWithCorrectRoot(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint,
		challenger.WithPrivKey(sys.cfg.Secrets.Mallory),
	)
	require.NotNil(t, game)
	game.LogGameData(ctx)

	// Honest first attack at "honest" level
	correctTrace.Attack(ctx, 0)

	// Honest defense at "dishonest" level
	correctTrace.Defend(ctx, 1)

	// Dishonest attack at "honest" level - honest move would be to ignore
	game.Attack(ctx, 2, common.Hash{0x03, 0xaa})

	// Honest attack at "dishonest" level - honest move would be to ignore
	correctTrace.Attack(ctx, 3)

	// Start the honest challenger
	game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, "Honest",
		// Agree with the proposed output, so disagree with the root claim
		challenger.WithAgreeProposedOutput(true),
		challenger.WithPrivKey(sys.cfg.Secrets.Bob),
	)

	// Start dishonest challenger that posts correct claims
	// It participates in the subgame root the honest claim index 4
	func() {
		claimCount := int64(5)
		depth := game.MaxDepth(ctx)
		for {
			game.LogGameData(ctx)
			claimCount++
			// Wait for the challenger to counter
			game.WaitForClaimCount(ctx, claimCount)

			// Respond with our own move
			correctTrace.Defend(ctx, claimCount-1)
			claimCount++
			game.WaitForClaimCount(ctx, claimCount)

			// Defender moves last. If we're at max depth, then we're done
			pos := game.GetClaimPosition(ctx, claimCount-1)
			if int64(pos.Depth()) == depth {
				break
			}
		}
	}()

	// Wait for the challenger to drive the subgame at 4 to the leaf node, which should be countered
	game.WaitForClaimAtMaxDepth(ctx, true)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForInactivity(ctx, 10, true)
	game.LogGameData(ctx)
	require.EqualValues(t, disputegame.StatusChallengerWins, game.Status(ctx))
}

// setupDisputeGameForInvalidOutputRoot sets up an L2 chain with at least one valid output root followed by an invalid output root.
// A cannon dispute game is started to dispute the invalid output root with the correct root claim provided.
// An honest challenger is run to defend the root claim (ie disagree with the invalid output root).
func setupDisputeGameForInvalidOutputRoot(t *testing.T, outputRoot common.Hash) (*System, *ethclient.Client, *disputegame.CannonGameHelper, *disputegame.HonestHelper) {
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)

	l2oo := l2oo2.NewL2OOHelper(t, sys.cfg.L1Deployments, l1Client, sys.cfg.Secrets.Proposer, sys.RollupConfig)

	// Wait for one valid output root to be submitted
	l2oo.WaitForProposals(ctx, 1)

	// Stop the honest output submitter so we can publish invalid outputs
	sys.L2OutputSubmitter.Stop()
	sys.L2OutputSubmitter = nil

	// Submit an invalid output root
	l2oo.PublishNextOutput(ctx, outputRoot)

	l1Endpoint := sys.NodeEndpoint("l1")
	l2Endpoint := sys.NodeEndpoint("sequencer")

	// Dispute the new output root by creating a new game with the correct cannon trace.
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	game, correctTrace := disputeGameFactory.StartCannonGameWithCorrectRoot(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint,
		challenger.WithPrivKey(sys.cfg.Secrets.Mallory),
	)
	require.NotNil(t, game)

	// Start the honest challenger
	game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, "Defender",
		// Disagree with the proposed output, so agree with the (correct) root claim
		challenger.WithAgreeProposedOutput(false),
		challenger.WithPrivKey(sys.cfg.Secrets.Mallory),
	)
	return sys, l1Client, game, correctTrace
}

func TestCannonChallengeWithCorrectRoot(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	l1Endpoint := sys.NodeEndpoint("l1")
	l2Endpoint := sys.NodeEndpoint("sequencer")

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	game, correctTrace := disputeGameFactory.StartCannonGameWithCorrectRoot(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint,
		challenger.WithPrivKey(sys.cfg.Secrets.Mallory),
	)
	require.NotNil(t, game)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, "Challenger",
		// Agree with the proposed output, so disagree with the root claim
		challenger.WithAgreeProposedOutput(true),
		challenger.WithPrivKey(sys.cfg.Secrets.Alice),
	)

	game.DefendRootClaim(ctx, func(parentClaimIdx int64) {
		// Defend everything because we have the same trace as the honest proposer
		correctTrace.Defend(ctx, parentClaimIdx)
	})

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForInactivity(ctx, 10, true)
	game.LogGameData(ctx)
	require.EqualValues(t, disputegame.StatusChallengerWins, game.Status(ctx))
}

func startFaultDisputeSystem(t *testing.T) (*System, *ethclient.Client) {
	cfg := DefaultSystemConfig(t)
	delete(cfg.Nodes, "verifier")
	cfg.DeployConfig.SequencerWindowSize = 4
	cfg.DeployConfig.FinalizationPeriodSeconds = 2
	cfg.SupportL1TimeTravel = true
	cfg.DeployConfig.L2OutputOracleSubmissionInterval = 1
	cfg.NonFinalizedProposals = true // Submit output proposals asap
	sys, err := cfg.Start(t)
	require.Nil(t, err, "Error starting up system")
	return sys, sys.Clients["l1"]
}
