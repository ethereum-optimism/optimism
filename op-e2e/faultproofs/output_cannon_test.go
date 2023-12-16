package faultproofs

import (
	"context"
	"fmt"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

const outputCannonTestExecutor = 0

func TestOutputCannonGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 4, common.Hash{0x01})
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	game.LogGameData(ctx)
	// Challenger should post an output root to counter claims down to the leaf level of the top game
	splitDepth := game.SplitDepth(ctx)
	for i := int64(1); i < splitDepth; i += 2 {
		game.WaitForCorrectOutputRoot(ctx, i)
		game.Attack(ctx, i, common.Hash{0xaa})
		game.LogGameData(ctx)
	}

	// Wait for the challenger to post the first claim in the cannon trace
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

func TestOutputCannon_PublishCannonRootClaim(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
	tests := []struct {
		disputeL2BlockNumber uint64
	}{
		{7}, // Post-state output root is invalid
		{8}, // Post-state output root is valid
	}
	for _, test := range tests {
		test := test
		t.Run(fmt.Sprintf("Dispute_%v", test.disputeL2BlockNumber), func(t *testing.T) {
			op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
			ctx := context.Background()
			sys, _ := startFaultDisputeSystem(t)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", test.disputeL2BlockNumber, common.Hash{0x01})
			game.DisputeLastBlock(ctx)
			game.LogGameData(ctx)

			game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

			splitDepth := game.SplitDepth(ctx)
			game.WaitForClaimAtDepth(ctx, int(splitDepth)+1)
		})
	}
}

func TestOutputCannonDisputeGame(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))

	tests := []struct {
		name             string
		defendClaimDepth int64
	}{
		{"StepFirst", 0},
		{"StepMiddle", 28},
		{"StepInExtension", 1},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			op_e2e.InitParallel(t, op_e2e.UseExecutor(outputCannonTestExecutor))

			ctx := context.Background()
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
			require.NotNil(t, game)
			game.LogGameData(ctx)

			game.DisputeLastBlock(ctx)
			splitDepth := game.SplitDepth(ctx)

			game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

			game.DefendRootClaim(
				ctx,
				func(parentClaimIdx int64) {
					if parentClaimIdx+1 == splitDepth+test.defendClaimDepth {
						game.Defend(ctx, parentClaimIdx, common.Hash{byte(parentClaimIdx)})
					} else {
						game.Attack(ctx, parentClaimIdx, common.Hash{byte(parentClaimIdx)})
					}
				})

			sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
			require.NoError(t, wait.ForNextBlock(ctx, l1Client))

			game.LogGameData(ctx)
			game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
		})
	}
}

func TestOutputCannonDefendStep(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
	game := disputeGameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
	require.NotNil(t, game)
	game.DisputeLastBlock(ctx)
	game.LogGameData(ctx)

	game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

	correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))

	splitDepth := game.SplitDepth(ctx)
	game.DefendRootClaim(ctx, func(parentClaimIdx int64) {
		// Post invalid claims for most steps to get down into the early part of the trace
		if parentClaimIdx < splitDepth+27 {
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

func TestOutputCannonProposedOutputRootValid(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
	// honestStepsFail attempts to perform both an attack and defend step using the correct trace.
	honestStepsFail := func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, parentClaimIdx int64) {
		// Attack step should fail
		correctTrace.StepFails(ctx, parentClaimIdx, true)
		// Defending should fail too
		correctTrace.StepFails(ctx, parentClaimIdx, false)
	}
	tests := []struct {
		// name is the name of the test
		name string

		// performMove is called to respond to each claim posted by the honest op-challenger.
		// It should either attack or defend the claim at parentClaimIdx
		performMove func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, parentClaimIdx int64)

		// performStep is called once the maximum game depth is reached. It should perform a step to counter the
		// claim at parentClaimIdx. Since the proposed output root is invalid, the step call should always revert.
		performStep func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, parentClaimIdx int64)
	}{
		{
			name: "AttackWithCorrectTrace",
			performMove: func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, parentClaimIdx int64) {
				// Attack everything but oddly using the correct hash.
				// Except the root of the cannon game must have an invalid VM status code.
				splitDepth := game.SplitDepth(ctx)
				if splitDepth == parentClaimIdx {
					// TODO(client-pod#262): Verify that an attack with a valid status code is rejected
					game.Attack(ctx, parentClaimIdx, common.Hash{0x01})
					return
				}
				correctTrace.Attack(ctx, parentClaimIdx)
			},
			performStep: honestStepsFail,
		},
		{
			name: "DefendWithCorrectTrace",
			performMove: func(ctx context.Context, game *disputegame.OutputCannonGameHelper, correctTrace *disputegame.OutputHonestHelper, parentClaimIdx int64) {
				splitDepth := game.SplitDepth(ctx)
				// Can only attack the root claim or the first cannon claim
				if parentClaimIdx == 0 {
					correctTrace.Attack(ctx, parentClaimIdx)
					return
				}
				// The root of the cannon game must have an invalid VM status code
				// Attacking ensure we're running the cannon trace between two different blocks
				// instead of being in the trace extension of the output root bisection
				if splitDepth == parentClaimIdx {
					// TODO(client-pod#262): Verify that an attack with a valid status code is rejected
					game.Attack(ctx, parentClaimIdx, common.Hash{0x01})
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
			sys, l1Client := startFaultDisputeSystem(t)
			t.Cleanup(sys.Close)

			disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, sys)
			game := disputeGameFactory.StartOutputCannonGameWithCorrectRoot(ctx, "sequencer", 1)
			correctTrace := game.CreateHonestActor(ctx, "sequencer", challenger.WithPrivKey(sys.Cfg.Secrets.Mallory))

			game.StartChallenger(ctx, "sequencer", "Challenger", challenger.WithPrivKey(sys.Cfg.Secrets.Alice))

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
