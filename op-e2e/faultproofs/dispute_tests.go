package faultproofs

import (
	"context"
	"testing"

	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func testCannonGame(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	game.LogGameData(ctx)
	arena.CreateChallenger(ctx)

	require.True(t, !rootIsCorrect(t, ctx, arena, game), "This test must be run with an incorrect proposal root")

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

	// Attack the root of the cannon trace subgame
	claim = claim.Attack(ctx, common.Hash{0x00, 0xcc})
	lastDishonestClaim := claim
	for !claim.IsMaxDepth(ctx) {
		if claim.AgreesWithOutputRoot() {
			// If the latest claim supports the output root, wait for the honest challenger to respond
			claim = claim.WaitForCounterClaim(ctx)
			game.LogGameData(ctx)
		} else {
			// Otherwise we need to counter the honest claim
			claim = claim.Defend(ctx, common.Hash{0x00, 0xdd})
			lastDishonestClaim = claim
			game.LogGameData(ctx)
		}
	}

	if expectChallengerToStep := game.MaxDepth(ctx)%2 == 0; expectChallengerToStep {
		t.Log("Waiting for challenger to step...")
		claim.WaitForCountered(ctx)
	} else {
		t.Log("Calling step on challenger's claim...")
		require.NotEqual(t, lastDishonestClaim.Position, claim.Position, "sanity checking challenger claim failed")
		honestActor := arena.CreateHonestActor(ctx)
		honestActor.StepFails(ctx, claim.Index, false)
		honestActor.StepFails(ctx, claim.Index, true)
	}
	game.LogGameData(ctx)

	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
}

func rootIsCorrect(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) bool {
	root, err := game.Game.GetClaim(ctx, 0)
	require.NoError(t, err)
	_, l2SequenceNumber, err := game.Game.GetGameRange(ctx)
	require.NoError(t, err)
	output := arena.GetProposalRoot(ctx, l2SequenceNumber)
	return output == root.Value
}
