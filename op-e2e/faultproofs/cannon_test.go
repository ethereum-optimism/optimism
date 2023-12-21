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

func TestCannonPoisonedPostState(t *testing.T) {
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

	// Honest first attack at "honest" level
	correctTrace.Attack(ctx, 0)

	// Honest defense at "dishonest" level
	correctTrace.Defend(ctx, 1)

	// Dishonest attack at "honest" level - honest move would be to ignore
	game.Attack(ctx, 2, common.Hash{0x03, 0xaa})

	// Honest attack at "dishonest" level - honest move would be to ignore
	correctTrace.Attack(ctx, 3)

	// Start the honest challenger
	game.StartChallenger(ctx, "sequencer", "Honest", challenger.WithPrivKey(sys.Cfg.Secrets.Bob))

	// Start dishonest challenger that posts correct claims
	// It participates in the subgame root the honest claim index 4
	func() {
		claimCount := int64(5)
		depth := game.MaxDepth(ctx)
		for {
			game.LogGameData(ctx)
			claimCount++
			// Wait for the challenger to counter
			game.WaitForClaimCount(ctx, claimCount)

			// Respond with our own move
			correctTrace.Defend(ctx, claimCount-1)
			claimCount++
			game.WaitForClaimCount(ctx, claimCount)

			// Defender moves last. If we're at max depth, then we're done
			pos := game.GetClaimPosition(ctx, claimCount-1)
			if int64(pos.Depth()) == depth {
				break
			}
		}
	}()

	// Wait for the challenger to drive the subgame at 4 to the leaf node, which should be countered
	game.WaitForClaimAtMaxDepth(ctx, true)

	// Time travel past when the game will be resolvable.
	sys.TimeTravelClock.AdvanceTime(game.GameDuration(ctx))
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game.WaitForInactivity(ctx, 10, true)
	game.LogGameData(ctx)
	require.EqualValues(t, disputegame.StatusChallengerWins, game.Status(ctx))
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
