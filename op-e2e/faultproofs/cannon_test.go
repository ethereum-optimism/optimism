package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestCannonDisputeGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(1))

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
			op_e2e.InitParallel(t, op_e2e.UseExecutor(1))

			ctx := context.Background()
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
			require.NotNil(t, game)
			game.LogGameData(ctx)

			game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

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
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(1))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
	require.NotNil(t, game)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))

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
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(0))
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
			op_e2e.InitParallel(t, op_e2e.UseExecutor(0))

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

func TestCannonChallengeWithCorrectRoot(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(0))
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game, correctTrace := disputeGameFactory.StartCannonGameWithCorrectRoot(ctx, "sequencer",
		challenger.WithPrivKey(sys.Cfg.Secrets.Mallory),
	)
	require.NotNil(t, game)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "sequencer", "Challenger",
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
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
