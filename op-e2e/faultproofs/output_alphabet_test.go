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
	game := disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 3, common.Hash{0xff})
	correctTrace := game.CreateHonestActor(ctx, "sequencer")
	game.LogGameData(ctx)

	opts := challenger.WithPrivKey(sys.Cfg.Secrets.Alice)
	game.StartChallenger(ctx, "sequencer", "Challenger", opts)
	game.LogGameData(ctx)

	// Challenger should post an output root to counter claims down to the leaf level of the top game
	claim := game.RootClaim(ctx)
	for claim.IsOutputRoot(ctx) && !claim.IsOutputRootLeaf(ctx) {
		if claim.AgreesWithOutputRoot() {
			// If the latest claim agrees with the output root, expect the honest challenger to counter it
			claim = claim.WaitForCounterClaim(ctx)
			game.LogGameData(ctx)
			claim.RequireCorrectOutputRoot(ctx)
		} else {
			// Otherwise we should counter
			claim = claim.Attack(ctx, common.Hash{0xaa})
			game.LogGameData(ctx)
		}
	}

	// Wait for the challenger to post the first claim in the cannon trace
	claim = claim.WaitForCounterClaim(ctx)
	game.LogGameData(ctx)

	// Attack the root of the alphabet trace subgame
	claim = correctTrace.AttackClaim(ctx, claim)
	for !claim.IsMaxDepth(ctx) {
		if claim.AgreesWithOutputRoot() {
			// If the latest claim supports the output root, wait for the honest challenger to respond
			claim = claim.WaitForCounterClaim(ctx)
			game.LogGameData(ctx)
		} else {
			// Otherwise we need to counter the honest claim
			claim = correctTrace.AttackClaim(ctx, claim)
			game.LogGameData(ctx)
		}
	}
	// Challenger should be able to call step and counter the leaf claim.
	claim.WaitForCountered(ctx)
	game.LogGameData(ctx)

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	game.LogGameData(ctx)
}

func TestOutputAlphabetGame_ValidOutputRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UseExecutor(1))
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputAlphabetGameWithCorrectRoot(ctx, "sequencer", 2)
	correctTrace := game.CreateHonestActor(ctx, "sequencer")
	game.LogGameData(ctx)
	claim := game.DisputeLastBlock(ctx)
	// Invalid root claim of the alphabet game
	claim = claim.Attack(ctx, common.Hash{0x01})

	opts := challenger.WithPrivKey(sys.Cfg.Secrets.Alice)
	game.StartChallenger(ctx, "sequencer", "Challenger", opts)

	claim = claim.WaitForCounterClaim(ctx)
	game.LogGameData(ctx)
	for !claim.IsMaxDepth(ctx) {
		// Dishonest actor always attacks with the correct trace
		claim = correctTrace.AttackClaim(ctx, claim)
		claim = claim.WaitForCounterClaim(ctx)
		game.LogGameData(ctx)
	}

	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))
	game.WaitForGameStatus(ctx, disputegame.StatusDefenderWins)
}

func TestChallengerCompleteExhaustiveDisputeGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UseExecutor(1))

	testCase := func(t *testing.T, isRootCorrect bool) {
		ctx := context.Background()
		sys, l1Client := startFaultDisputeSystem(t)
		t.Cleanup(sys.Close)

		disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
		var game *disputegame.OutputAlphabetGameHelper
		if isRootCorrect {
			game = disputeGameFactory.StartOutputAlphabetGameWithCorrectRoot(ctx, "sequencer", 1)
		} else {
			game = disputeGameFactory.StartOutputAlphabetGame(ctx, "sequencer", 1, common.Hash{0xaa, 0xbb, 0xcc})
		}
		claim := game.DisputeLastBlock(ctx)

		game.LogGameData(ctx)

		// Start honest challenger
		game.StartChallenger(ctx, "sequencer", "Challenger",
			challenger.WithAlphabet(sys.RollupEndpoint("sequencer")),
			challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
			// Ensures the challenger responds to all claims before test timeout
			challenger.WithPollInterval(time.Millisecond*400),
		)

		if isRootCorrect {
			// Attack the correct output root with an invalid alphabet trace
			claim = claim.Attack(ctx, common.Hash{0x01})
		} else {
			// Wait for the challenger to counter the invalid output root
			claim = claim.WaitForCounterClaim(ctx)
		}

		// Start dishonest challenger
		dishonestHelper := game.CreateDishonestHelper(ctx, "sequencer", !isRootCorrect)
		dishonestHelper.ExhaustDishonestClaims(ctx, claim)

		// Wait until we've reached max depth before checking for inactivity
		game.WaitForClaimAtDepth(ctx, game.MaxDepth(ctx))

		// Wait for 4 blocks of no challenger responses. The challenger may still be stepping on invalid claims at max depth
		game.WaitForInactivity(ctx, 4, false)

		gameDuration := game.GameDuration(ctx)
		sys.TimeTravelClock.AdvanceTime(gameDuration)
		require.NoError(t, wait.ForNextBlock(ctx, l1Client))

		expectedStatus := disputegame.StatusChallengerWins
		if isRootCorrect {
			expectedStatus = disputegame.StatusDefenderWins
		}
		game.WaitForGameStatus(ctx, expectedStatus)
		game.LogGameData(ctx)
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
