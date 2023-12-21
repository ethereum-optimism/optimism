package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/stretchr/testify/require"
)

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
