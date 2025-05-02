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
	for !claim.IsMaxDepth(ctx) {
		if claim.AgreesWithOutputRoot() {
			// If the latest claim supports the output root, wait for the honest challenger to respond
			claim = claim.WaitForCounterClaim(ctx)
			game.LogGameData(ctx)
		} else {
			// Otherwise we need to counter the honest claim
			claim = claim.Defend(ctx, common.Hash{0x00, 0xdd})
			game.LogGameData(ctx)
		}
	}

	verifyFinalStepExecution(t, ctx, arena, game, claim)
	game.LogGameData(ctx)

	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
}

func testCannonChallengeAllZeroClaim(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, !rootIsCorrect(t, ctx, arena, game), "This test must be run with an incorrect proposal root")
	game.LogGameData(ctx)

	claim := game.DisputeLastBlock(ctx)
	arena.CreateChallenger(ctx)

	game.SupportClaim(ctx, claim, func(parent *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		if parent.IsBottomGameRoot(ctx) {
			return parent.Attack(ctx, common.Hash{})
		}
		return parent.Defend(ctx, common.Hash{})
	}, verifyFinalStepper(t, ctx, arena, game))
	game.LogGameData(ctx)

	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
	game.LogGameData(ctx)
}

func testCannonDefendStep(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, !rootIsCorrect(t, ctx, arena, game), "This test must be run with an incorrect proposal root")
	outputRootClaim := game.DisputeLastBlock(ctx)
	game.LogGameData(ctx)
	arena.CreateChallenger(ctx)

	correctTrace := arena.CreateHonestActor(ctx)

	maxDepth := game.MaxDepth(ctx)
	game.SupportClaim(ctx, outputRootClaim, func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		// Post invalid claims for most steps to get down into the early part of the trace
		if claim.Depth() < maxDepth-3 {
			return claim.Attack(ctx, common.Hash{0xaa})
		} else {
			// Post our own counter but using the correct hash in low levels to force a defense step
			return correctTrace.AttackClaim(ctx, claim)
		}
	}, verifyFinalStepper(t, ctx, arena, game))

	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
}

func testCannonPoisonedPostState(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, !rootIsCorrect(t, ctx, arena, game), "This test must be run with an incorrect proposal root")
	correctTrace := arena.CreateHonestActor(ctx)

	// Honest first attack at "honest" level
	claim := correctTrace.AttackClaim(ctx, game.RootClaim(ctx))

	// Honest defense at "dishonest" level
	claim = correctTrace.DefendClaim(ctx, claim)

	// Dishonest attack at "honest" level - honest move would be to ignore
	claimToIgnore1 := claim.Attack(ctx, common.Hash{0x03, 0xaa})

	// Honest attack at "dishonest" level - honest move would be to ignore
	claimToIgnore2 := correctTrace.AttackClaim(ctx, claimToIgnore1)
	game.LogGameData(ctx)

	// Start the honest challenger
	arena.CreateChallenger(ctx)

	// Start dishonest challenger that posts correct claims
	for {
		game.LogGameData(ctx)
		// Wait for the challenger to counter
		// Note that we need to ignore claimToIgnore1 which already counters this...
		claim = claim.WaitForCounterClaim(ctx, claimToIgnore1)
		if claim.IsMaxDepth(ctx) {
			break
		}

		// Respond with our own move
		if claim.IsBottomGameRoot(ctx) {
			// Root of the cannon game must have the right VM status code (so it can't be honest).
			// Note this occurs when there are splitDepth + 4 claims because there are multiple forks in this game.
			claim = claim.Attack(ctx, common.Hash{0x01})
		} else {
			claim = correctTrace.DefendClaim(ctx, claim)
		}
		if claim.IsMaxDepth(ctx) {
			break
		}
	}
	verifyFinalStepExecution(t, ctx, arena, game, claim)

	// Verify that the challenger didn't challenge our poisoned claims
	claimToIgnore1.RequireOnlyCounteredBy(ctx, claimToIgnore2)
	claimToIgnore2.RequireOnlyCounteredBy(ctx /* nothing */)

	// Time travel past when the game will be resolvable.
	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))

	game.LogGameData(ctx)
	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
}

func testDisputeRootBeyondProposedBlockValidOutputRoot(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, rootIsCorrect(t, ctx, arena, game), "This test must be run with a correct proposal root")
	correctTrace := arena.CreateHonestActor(ctx)
	// Start the honest challenger
	arena.CreateChallenger(ctx)

	claim := game.RootClaim(ctx)
	// Attack the output root
	claim = correctTrace.AttackClaim(ctx, claim)
	// Wait for the challenger to respond
	claim = claim.WaitForCounterClaim(ctx)
	// Then defend until the split depth to force the game into the extension part of the output root bisection
	// ie. the output root we wind up disputing is theoretically for a block after block number 1
	for !claim.IsOutputRootLeaf(ctx) {
		claim = correctTrace.DefendClaim(ctx, claim)
		claim = claim.WaitForCounterClaim(ctx)
	}
	game.LogGameData(ctx)
	// At this point we've reached the bottom of the output root bisection and every claim
	// will have the same, valid, output root. We now need to post a cannon trace root that claims its invalid.
	claim = claim.Defend(ctx, common.Hash{0x01, 0xaa})
	// Now defend with the correct trace
	for {
		game.LogGameData(ctx)
		claim = claim.WaitForCounterClaim(ctx)
		if claim.IsMaxDepth(ctx) {
			break
		}
		claim = correctTrace.DefendClaim(ctx, claim)
		if claim.IsMaxDepth(ctx) {
			break
		}
	}
	verifyFinalStepExecution(t, ctx, arena, game, claim)

	// Time travel past when the game will be resolvable.
	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))

	game.WaitForGameStatus(ctx, gameTypes.GameStatusDefenderWon)
	game.LogGameData(ctx)
}

func testDisputeRootBeyondProposedBlockInvalidOutputRoot(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, !rootIsCorrect(t, ctx, arena, game), "This test must be run with an incorrect proposal root")
	correctTrace := arena.CreateHonestActor(ctx)

	// Start the honest challenger
	arena.CreateChallenger(ctx)

	claim := game.RootClaim(ctx)
	// Wait for the honest challenger to counter the root
	claim = claim.WaitForCounterClaim(ctx)
	// Then defend until the split depth to force the game into the extension part of the output root bisection
	// ie. the output root we wind up disputing is theoretically for a block after block number 1
	// The dishonest actor challenges with the correct roots
	for claim.IsOutputRoot(ctx) {
		claim = correctTrace.DefendClaim(ctx, claim)
		claim = claim.WaitForCounterClaim(ctx)
	}
	game.LogGameData(ctx)
	// Now defend with the correct trace
	for !claim.IsMaxDepth(ctx) {
		game.LogGameData(ctx)
		if claim.IsBottomGameRoot(ctx) {
			claim = correctTrace.AttackClaim(ctx, claim)
		} else {
			claim = correctTrace.DefendClaim(ctx, claim)
		}
		if !claim.IsMaxDepth(ctx) {
			// Have to attack the root of the cannon trace
			claim = claim.WaitForCounterClaim(ctx)
		}
	}
	verifyFinalStepExecution(t, ctx, arena, game, claim)

	// Time travel past when the game will be resolvable.
	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))

	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
	game.LogGameData(ctx)
}

func testDisputeRootChangeClaimedRoot(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, !rootIsCorrect(t, ctx, arena, game), "This test must be run with an incorrect proposal root")
	correctTrace := arena.CreateHonestActor(ctx)

	// Start the honest challenger
	arena.CreateChallenger(ctx)

	claim := game.RootClaim(ctx)
	// Wait for the honest challenger to counter the root
	claim = claim.WaitForCounterClaim(ctx)

	// Then attack every claim until the leaf of output root bisection
	for {
		claim = claim.Attack(ctx, common.Hash{0xbb})
		claim = claim.WaitForCounterClaim(ctx)
		if claim.Depth() == game.SplitDepth(ctx)-1 {
			// Post the correct output root as the leaf.
			// This is for epoch 1 which is what the original root proposal was for too
			claim = correctTrace.AttackClaim(ctx, claim)
			// Challenger should post the first cannon trace
			claim = claim.WaitForCounterClaim(ctx)
			break
		}
	}

	game.LogGameData(ctx)

	// Now defend with the correct trace
	for !claim.IsMaxDepth(ctx) {
		game.LogGameData(ctx)
		if claim.IsBottomGameRoot(ctx) {
			claim = correctTrace.AttackClaim(ctx, claim)
		} else {
			claim = correctTrace.DefendClaim(ctx, claim)
		}
		if !claim.IsMaxDepth(ctx) {
			// Have to attack the root of the cannon trace
			claim = claim.WaitForCounterClaim(ctx)
		}
	}
	verifyFinalStepExecution(t, ctx, arena, game, claim)

	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))

	game.WaitForGameStatus(ctx, gameTypes.GameStatusChallengerWon)
	game.LogGameData(ctx)
}

func testCannonProposalValid_AttackWithCorrectTrace(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, rootIsCorrect(t, ctx, arena, game), "This test must be run with a correct proposal root")
	correctTrace := arena.CreateHonestActor(ctx)

	performMove := func(ctx context.Context, correctTrace *disputegame.OutputHonestHelper, claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		// Attack everything but oddly using the correct hash.
		// Except the root of the cannon game must have an invalid VM status code.
		if claim.IsOutputRootLeaf(ctx) {
			return claim.Attack(ctx, common.Hash{0x01})
		}
		return correctTrace.AttackClaim(ctx, claim)
	}

	arena.CreateChallenger(ctx)

	rootAttack := performMove(ctx, correctTrace, game.RootClaim(ctx))
	game.SupportClaim(ctx,
		rootAttack,
		func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			return performMove(ctx, correctTrace, claim)
		},
		verifyFinalStepper(t, ctx, arena, game),
	)

	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusDefenderWon)
}

func testCannonProposalValid_DefendWithCorrectTrace(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) {
	require.True(t, rootIsCorrect(t, ctx, arena, game), "This test must be run with a correct proposal root")
	correctTrace := arena.CreateHonestActor(ctx)

	performMove := func(ctx context.Context, correctTrace *disputegame.OutputHonestHelper, claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
		// Can only attack the root claim or the first cannon claim
		if claim.IsRootClaim() {
			return correctTrace.AttackClaim(ctx, claim)
		}
		// The root of the cannon game must have an invalid VM status code
		// Attacking ensure we're running the cannon trace between two different blocks
		// instead of being in the trace extension of the output root bisection
		if claim.IsOutputRootLeaf(ctx) {
			return claim.Attack(ctx, common.Hash{0x01})
		}
		// Otherwise, defend everything using the correct hash.
		return correctTrace.DefendClaim(ctx, claim)
	}

	arena.CreateChallenger(ctx)

	rootAttack := performMove(ctx, correctTrace, game.RootClaim(ctx))
	game.SupportClaim(ctx,
		rootAttack,
		func(claim *disputegame.ClaimHelper) *disputegame.ClaimHelper {
			return performMove(ctx, correctTrace, claim)
		},
		verifyFinalStepper(t, ctx, arena, game),
	)

	arena.AdvanceTime(game.MaxClockDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, arena.L1Client()))
	game.WaitForGameStatus(ctx, gameTypes.GameStatusDefenderWon)
}

func rootIsCorrect(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) bool {
	root, err := game.Game.GetClaim(ctx, 0)
	require.NoError(t, err)
	_, l2SequenceNumber, err := game.Game.GetGameRange(ctx)
	require.NoError(t, err)
	output := arena.GetProposalRoot(ctx, l2SequenceNumber)
	return output == root.Value
}

func verifyFinalStepper(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper) disputegame.Stepper {
	return func(parentClaimIdx int64) {
		validRoot := rootIsCorrect(t, ctx, arena, game)
		maxDepth := game.MaxDepth(ctx)
		expectChallengerToStep := validRoot == (maxDepth%2 == 1)
		if expectChallengerToStep {
			t.Log("Waiting for challenger to step on dishonest claim...")
			game.WaitForCountered(ctx, parentClaimIdx)
		} else {
			t.Log("Calling step on challenger's claim...")
			honestActor := arena.CreateHonestActor(ctx)
			honestActor.StepFails(ctx, parentClaimIdx, false)
			honestActor.StepFails(ctx, parentClaimIdx, true)
		}
	}
}

func verifyFinalStepExecution(t *testing.T, ctx context.Context, arena gameArena, game *disputegame.SplitGameHelper, claim *disputegame.ClaimHelper) {
	validRoot := rootIsCorrect(t, ctx, arena, game)
	maxDepth := game.MaxDepth(ctx)
	expectChallengerToStep := validRoot == (maxDepth%2 == 1)

	if expectChallengerToStep {
		t.Log("Waiting for challenger to step on dishonest claim...")
		claim.WaitForCountered(ctx)
	} else {
		t.Log("Calling step on challenger's claim...")
		honestActor := arena.CreateHonestActor(ctx)
		honestActor.StepFails(ctx, claim.Index, false)
		honestActor.StepFails(ctx, claim.Index, true)
	}
}
