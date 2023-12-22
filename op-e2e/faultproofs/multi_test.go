package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestMultipleGameTypes(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(0))

	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys)

	game1 := gameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
	game2 := gameFactory.StartOutputAlphabetGame(ctx, "sequencer", 1, "xyzabc")
	game1.DisputeLastBlock(ctx)
	game2.DisputeLastBlock(ctx)

	nextClaimIdx1 := game1.SplitDepth(ctx) + 1
	nextClaimIdx2 := game2.SplitDepth(ctx) + 1

	// Start a challenger with both cannon and alphabet support
	gameFactory.StartChallenger(ctx, "TowerDefense",
		challenger.WithOutputCannon(t, sys.RollupConfig, sys.L2GenesisCfg, sys.RollupEndpoint("sequencer"), sys.NodeEndpoint("sequencer")),
		challenger.WithOutputAlphabet(disputegame.CorrectAlphabet, sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
	)

	// Wait for the challenger to respond to both games
	game1.WaitForClaimCount(ctx, nextClaimIdx1)
	game2.WaitForClaimCount(ctx, nextClaimIdx2)
	game1Response := game1.GetClaimValue(ctx, nextClaimIdx1)
	game2Response := game2.GetClaimValue(ctx, nextClaimIdx2)
	// The alphabet game always posts the same traces, so if they're different they can't both be from the alphabet.
	// We're contesting the same block with different VMs, so if the challenger was just playing two cannon or alphabet
	// games the responses would be equal.
	require.NotEqual(t, game1Response, game2Response, "should have posted different claims")
}
