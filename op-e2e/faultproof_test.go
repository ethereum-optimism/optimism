package op_e2e

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	l2oo2 "github.com/ethereum-optimism/optimism/op-e2e/e2eutils/l2oo"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestMultipleAlphabetGames(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	// Start a challenger with the correct alphabet trace
	gameFactory.StartChallenger(ctx, sys.NodeEndpoint("l1"), "TowerDefense",
		challenger.WithAlphabet("abcdefg"),
		challenger.WithPrivKey(sys.cfg.Secrets.Alice),
		challenger.WithAgreeProposedOutput(true),
	)

	game1 := gameFactory.StartAlphabetGame(ctx, "abcxyz")
	// Wait for the challenger to respond to the first game
	game1.WaitForClaimCount(ctx, 2)

	game2 := gameFactory.StartAlphabetGame(ctx, "zyxabc")
	// Wait for the challenger to respond to the second game
	game2.WaitForClaimCount(ctx, 2)

	// Challenger should respond to new claims
	game2.Attack(ctx, 1, common.Hash{0xaa})
	game2.WaitForClaimCount(ctx, 4)
	game1.Defend(ctx, 1, common.Hash{0xaa})
	game1.WaitForClaimCount(ctx, 4)

	gameDuration := game1.GameDuration(ctx)
	sys.TimeTravelClock.AdvanceTime(gameDuration)
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game1.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game2.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
}

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

	game1 := gameFactory.StartCannonGame(ctx, common.Hash{0xaa})
	game2 := gameFactory.StartCannonGame(ctx, common.Hash{0xbb})

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

	game1.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game2.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)

	// Check that the game directories are removed
	challenger.WaitForGameDataDeletion(ctx, game1, game2)
}

func TestResolveDisputeGame(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)

	game := disputeGameFactory.StartAlphabetGame(ctx, "zyxwvut")
	require.NotNil(t, game)
	gameDuration := game.GameDuration(ctx)

	game.WaitForGameStatus(ctx, disputegame.StatusInProgress)

	game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "HonestAlice",
		challenger.WithAgreeProposedOutput(true),
		challenger.WithAlphabet("abcdefg"),
		challenger.WithPrivKey(sys.cfg.Secrets.Alice),
	)

	game.WaitForClaimCount(ctx, 2)

	sys.TimeTravelClock.AdvanceTime(gameDuration)
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	// Challenger should resolve the game now that the clocks have expired.
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
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

			game.WaitForGameStatus(ctx, test.expectedResult)
		})
	}
}

func TestCannonDisputeGame(t *testing.T) {
	InitParallel(t)

	tests := []struct {
		name          string
		defendAtClaim int64
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
			game := disputeGameFactory.StartCannonGame(ctx, common.Hash{0xaa})
			require.NotNil(t, game)
			game.LogGameData(ctx)

			game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, sys.NodeEndpoint("l1"), sys.NodeEndpoint("sequencer"), "Challenger",
				// Agree with the proposed output, so disagree with the root claim
				challenger.WithAgreeProposedOutput(true),
				challenger.WithPrivKey(sys.cfg.Secrets.Alice),
			)

			maxDepth := game.MaxDepth(ctx)
			for claimCount := int64(1); claimCount < maxDepth; {
				game.LogGameData(ctx)
				claimCount++
				// Wait for the challenger to counter
				game.WaitForClaimCount(ctx, claimCount)

				// Post our own counter to the latest challenger claim
				if claimCount == test.defendAtClaim {
					// Defend one claim so we don't wind up executing from the absolute pre-state
					game.Defend(ctx, claimCount-1, common.Hash{byte(claimCount)})
				} else {
					game.Attack(ctx, claimCount-1, common.Hash{byte(claimCount)})
				}
				claimCount++
				game.WaitForClaimCount(ctx, claimCount)
			}

			game.LogGameData(ctx)
			// Wait for the challenger to call step and counter our invalid claim
			game.WaitForClaimAtMaxDepth(ctx, true)

			sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
			require.NoError(t, wait.ForNextBlock(ctx, l1Client))

			game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
			game.LogGameData(ctx)
		})
	}
}

func TestCannonDefendStep(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	game := disputeGameFactory.StartCannonGame(ctx, common.Hash{0xaa})
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

	maxDepth := game.MaxDepth(ctx)
	for claimCount := int64(1); claimCount < maxDepth; {
		game.LogGameData(ctx)
		claimCount++
		// Wait for the challenger to counter
		game.WaitForClaimCount(ctx, claimCount)

		// Post invalid claims for most steps to get down into the early part of the trace
		if claimCount < 28 {
			game.Attack(ctx, claimCount-1, common.Hash{byte(claimCount)})
		} else {
			// Post our own counter but using the correct hash in low levels to force a defense step
			correctTrace.Attack(ctx, claimCount-1)
		}
		claimCount++
		game.LogGameData(ctx)
		game.WaitForClaimCount(ctx, claimCount)
	}

	game.LogGameData(ctx)
	// Wait for the challenger to call step and counter our invalid claim
	game.WaitForClaimAtMaxDepth(ctx, true)

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameData(ctx)
}

func TestCannonProposedOutputRootInvalid(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client, game, correctTrace := setupDisputeGameForInvalidOutputRoot(t, common.Hash{0xab})
	t.Cleanup(sys.Close)

	maxDepth := game.MaxDepth(ctx)

	// Now maliciously play the game and it should be impossible to win

	for claimCount := int64(1); claimCount < maxDepth; {
		// Attack everything but oddly using the correct hash.
		correctTrace.Attack(ctx, claimCount-1)
		claimCount++
		game.LogGameData(ctx)
		game.WaitForClaimCount(ctx, claimCount)

		game.LogGameData(ctx)
		// Wait for the challenger to counter
		claimCount++
		game.WaitForClaimCount(ctx, claimCount)
	}

	game.LogGameData(ctx)
	// Wait for the challenger to call step and counter our invalid claim
	game.WaitForClaimAtMaxDepth(ctx, false)

	// It's on us to call step if we want to win but shouldn't be possible
	// Need to add support for this to the helper

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, disputegame.StatusDefenderWins)
	game.LogGameData(ctx)
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

	// Submit an invalid output rooot
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
	t.Skip("Not currently handling this case as the correct approach will change when output root bisection is added")
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

	maxDepth := game.MaxDepth(ctx)
	for claimCount := int64(1); claimCount < maxDepth; {
		game.LogGameData(ctx)
		claimCount++
		// Wait for the challenger to counter
		game.WaitForClaimCount(ctx, claimCount)

		// Defend everything because we have the same trace as the honest proposer
		correctTrace.Defend(ctx, claimCount-1)
		claimCount++
		game.LogGameData(ctx)
		game.WaitForClaimCount(ctx, claimCount)
	}

	game.LogGameData(ctx)
	// Wait for the challenger to call step and counter our invalid claim
	game.WaitForClaimAtMaxDepth(ctx, true)

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameData(ctx)
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
