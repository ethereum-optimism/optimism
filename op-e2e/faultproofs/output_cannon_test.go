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
	// TODO(client-pod#336) Reduce the number of cases and enable this tests
	t.Skip("Contracts always require VM status to indicate the post-state output root is invalid")
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(outputCannonTestExecutor))
	tests := []struct {
		disputeL2BlockNumber uint64
	}{
		{1},
		{2},
		{3},
		{4},
		{5},
		{6},
		{7},
		{8},
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
	// TODO(client-pod#247): Fix and enable this.
	t.Skip("Currently failing because of invalid pre-state")
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
	// TODO(client-pod#247): Fix and enable this.
	t.Skip("Currently failing because of invalid pre-state")
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
