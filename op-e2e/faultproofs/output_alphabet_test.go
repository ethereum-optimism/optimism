package faultproofs

import (
	"context"
	"testing"
	"time"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestOutputAlphabetGame_ChallengerWins(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UseExecutor(1))
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 3, "abcdexyz")
	game.LogGameData(ctx)

	opts := challenger.WithPrivKey(sys.Cfg.Secrets.Alice)
	game.StartChallenger(ctx, "sequencer", "Challenger", opts)

	game.LogGameData(ctx)
	// Challenger should post an output root to counter claims down to the leaf level of the top game
	splitDepth := game.SplitDepth(ctx)
	for i := int64(1); i < splitDepth; i += 2 {
		game.WaitForCorrectOutputRoot(ctx, i)
		game.Attack(ctx, i, common.Hash{0xaa})
		game.LogGameData(ctx)
	}

	// Wait for the challenger to post the first claim in the alphabet trace
	game.WaitForClaimAtDepth(ctx, int(splitDepth+1))
	game.LogGameData(ctx)

	game.Attack(ctx, splitDepth+1, common.Hash{0x00, 0xcc})
	gameDepth := game.MaxDepth(ctx)
	for i := splitDepth + 3; i < gameDepth; i += 2 {
		// Wait for challenger to respond
		game.WaitForClaimAtDepth(ctx, int(i))
		game.LogGameData(ctx)

		// Respond to push the game down to the max depth
		game.Defend(ctx, i, common.Hash{0x00, 0xdd})
		game.LogGameData(ctx)
	}
	game.LogGameData(ctx)

	// Challenger should be able to call step and counter the leaf claim.
	game.WaitForClaimAtMaxDepth(ctx, true)
	game.LogGameData(ctx)

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
}

func TestOutputAlphabetGame_ExhaustiveDisputeGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UseExecutor(1))

	testCase := func(t *testing.T, isRootCorrect bool) {
		ctx := context.Background()
		sys, l1Client := startFaultDisputeSystem(t)
		t.Cleanup(sys.Close)

		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
		rootClaimedAlphabet := disputegame.CorrectAlphabet
		if !isRootCorrect {
			rootClaimedAlphabet = "abcdexyz"
		}
		game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 3, rootClaimedAlphabet)
		require.NotNil(t, game)
		game.LogGameData(ctx)
		gameDepth := uint64(game.MaxDepth(ctx))
		gameDuration := game.GameDuration(ctx)

		// Start honest challenger
		game.StartChallenger(ctx, "sequencer", "Challenger",
			challenger.WithOutputAlphabet(disputegame.CorrectAlphabet, sys.RollupEndpoint("sequencer")),
			challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
			// Ensures the challenger responds to all claims before test timeout
			challenger.WithPollInterval(time.Millisecond*400),
		)

		// Start dishonest challenger
		dishonestHelper := game.CreateDishonestHelper(
			ctx,
			disputegame.CorrectAlphabet,
			gameDepth,
			"sequencer",
			!isRootCorrect,
		)
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
		op_e2e.InitParallel(t, op_e2e.UseExecutor(1))
		testCase(t, true)
	})
	t.Run("RootIncorrect", func(t *testing.T) {
		op_e2e.InitParallel(t, op_e2e.UseExecutor(1))
		testCase(t, false)
	})
}

func TestChallengerCompleteOutputDisputeGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UseExecutor(1))

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
			op_e2e.InitParallel(t, op_e2e.UseExecutor(1))

			ctx := context.Background()
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 1, test.rootClaimAlphabet)
			require.NotNil(t, game)
			gameDuration := game.GameDuration(ctx)

			game.StartChallenger(ctx, "sequencer", "Defender",
				challenger.WithPrivKey(sys.Cfg.Secrets.Mallory),
			)

			game.StartChallenger(ctx, "sequencer", "Challenger",
				challenger.WithOutputAlphabet(test.otherAlphabet, sys.RollupEndpoint("sequencer")),
				challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
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
