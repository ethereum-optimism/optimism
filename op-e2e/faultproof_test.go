package op_e2e

import (
	"context"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
)

func TestCannonMultipleGames(t *testing.T) {
	t.Skip("Challenger doesn't yet support multiple games")
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	// Start a challenger with the correct alphabet trace
	gameFactory.StartChallenger(ctx, sys.NodeEndpoint("l1"), "TowerDefense", func(c *config.Config) {
		c.AgreeWithProposedOutput = true
		c.AlphabetTrace = "abcdefg"
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Alice)
	})

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

	game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "HonestAlice", func(c *config.Config) {
		c.AgreeWithProposedOutput = true // Agree with the proposed output, so disagree with the root claim
		c.AlphabetTrace = "abcdefg"
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Alice)
	})

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

			game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "Defender", func(c *config.Config) {
				c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Mallory)
			})

			game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "Challenger", func(c *config.Config) {
				c.AgreeWithProposedOutput = true // Agree with the proposed output, so disagree with the root claim
				c.AlphabetTrace = test.otherAlphabet
				c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Alice)
			})

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

			game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, sys.NodeEndpoint("l1"), sys.NodeEndpoint("sequencer"), "Challenger", func(c *config.Config) {
				c.AgreeWithProposedOutput = true // Agree with the proposed output, so disagree with the root claim
				c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Alice)
			})

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
	game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, "Challenger", func(c *config.Config) {
		c.AgreeWithProposedOutput = true // Agree with the proposed output, so disagree with the root claim
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Alice)
	})

	correctTrace := game.CreateHonestActor(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Client, l1Endpoint, l2Endpoint, func(c *config.Config) {
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Mallory)
	})

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

func TestCannonChallengeWithCorrectRoot(t *testing.T) {
	t.Skip("Not currently handling this case as the correct approach will change when output root bisection is added")
	InitParallel(t)

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	l1Endpoint := sys.NodeEndpoint("l1")
	l2Endpoint := sys.NodeEndpoint("sequencer")

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys.cfg.L1Deployments, l1Client)
	game, correctTrace := disputeGameFactory.StartCannonGameWithCorrectRoot(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, func(c *config.Config) {
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Mallory)
	})
	require.NotNil(t, game)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, sys.RollupConfig, sys.L2GenesisCfg, l1Endpoint, l2Endpoint, "Challenger", func(c *config.Config) {
		c.AgreeWithProposedOutput = true // Agree with the proposed output, so disagree with the root claim
		c.TxMgrConfig.PrivateKey = e2eutils.EncodePrivKeyToString(sys.cfg.Secrets.Alice)
	})

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
	sys, err := cfg.Start()
	require.NoError(t, err, "Error starting up system")
	return sys, sys.Clients["l1"]
}
