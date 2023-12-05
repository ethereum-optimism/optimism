package faultproofs

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/trace/alphabet"
	"github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMultipleCannonGames(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(0))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys.Cfg.L1Deployments, l1Client)
	// Start a challenger with the correct alphabet trace
	challenger := gameFactory.StartChallenger(ctx, sys.NodeEndpoint("l1"), "TowerDefense",
		challenger.WithCannon(t, sys.RollupConfig, sys.L2GenesisCfg, sys.NodeEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
	)

	game1 := gameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
	game2 := gameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xbb})

	game1.WaitForClaimCount(ctx, 2)
	game2.WaitForClaimCount(ctx, 2)

	game1Claim := game1.GetClaimValue(ctx, 1)
	game2Claim := game2.GetClaimValue(ctx, 1)
	require.NotEqual(t, game1Claim, game2Claim, "games should have different cannon traces")

	// Check that the helper finds the game directories correctly
	challenger.VerifyGameDataExists(game1, game2)

	// Push both games down to the step function
	maxDepth := game1.MaxDepth(ctx)
	for claimCount := int64(1); claimCount <= maxDepth; {
		// Challenger should respond to both games
		claimCount++
		game1.WaitForClaimCount(ctx, claimCount)
		game2.WaitForClaimCount(ctx, claimCount)

		// Progress both games
		game1.Defend(ctx, claimCount-1, common.Hash{0xaa})
		game2.Defend(ctx, claimCount-1, common.Hash{0xaa})
		claimCount++
	}

	game1.WaitForClaimAtMaxDepth(ctx, true)
	game2.WaitForClaimAtMaxDepth(ctx, true)

	gameDuration := game1.GameDuration(ctx)
	sys.TimeTravelClock.AdvanceTime(gameDuration)
	require.NoError(t, wait.ForNextBlock(ctx, l1Client))

	game1.WaitForInactivity(ctx, 10, true)
	game2.WaitForInactivity(ctx, 10, true)
	game1.LogGameData(ctx)
	game2.LogGameData(ctx)
	require.EqualValues(t, disputegame.StatusChallengerWins, game1.Status(ctx))
	require.EqualValues(t, disputegame.StatusChallengerWins, game2.Status(ctx))

	// Check that the game directories are removed
	challenger.WaitForGameDataDeletion(ctx, game1, game2)
}

func TestMultipleGameTypes(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(0))

	ctx := context.Background()
	sys, l1Client := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys.Cfg.L1Deployments, l1Client)
	// Start a challenger with both cannon and alphabet support
	gameFactory.StartChallenger(ctx, sys.NodeEndpoint("l1"), "TowerDefense",
		challenger.WithCannon(t, sys.RollupConfig, sys.L2GenesisCfg, sys.NodeEndpoint("sequencer")),
		challenger.WithAlphabet(disputegame.CorrectAlphabet),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
	)

	game1 := gameFactory.StartCannonGame(ctx, common.Hash{0x01, 0xaa})
	game2 := gameFactory.StartAlphabetGame(ctx, "xyzabc")

	// Wait for the challenger to respond to both games
	game1.WaitForClaimCount(ctx, 2)
	game2.WaitForClaimCount(ctx, 2)
	game1Response := game1.GetClaimValue(ctx, 1)
	game2Response := game2.GetClaimValue(ctx, 1)
	// The alphabet game always posts the same traces, so if they're different they can't both be from the alphabet.
	require.NotEqual(t, game1Response, game2Response, "should have posted different claims")
	// Now check they aren't both just from different cannon games by confirming the alphabet value.
	correctAlphabet := alphabet.NewTraceProvider(disputegame.CorrectAlphabet, uint64(game2.MaxDepth(ctx)))
	expectedClaim, err := correctAlphabet.Get(ctx, types.NewPositionFromGIndex(big.NewInt(1)).Attack())
	require.NoError(t, err)
	require.Equal(t, expectedClaim, game2Response)
	// We don't confirm the cannon value because generating the correct claim is expensive
	// Just being different is enough to confirm the challenger isn't just playing two alphabet games incorrectly
}
