package faultproofs

import (
	"context"
	"testing"

	op_e2e "github.com/ethereum-optimism/optimism/op-e2e"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/challenger"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum/go-ethereum/common"
)

func TestMultipleGameTypes(t *testing.T) {
	op_e2e.InitParallel(t, op_e2e.UsesCannon, op_e2e.UseExecutor(0))

	ctx := context.Background()
	sys, _ := startFaultDisputeSystem(t)
	t.Cleanup(sys.Close)

	gameFactory := disputegame.NewFactoryHelper(t, ctx, sys)

	game1 := gameFactory.StartOutputCannonGame(ctx, "sequencer", 1, common.Hash{0x01, 0xaa})
	game2 := gameFactory.StartOutputAlphabetGame(ctx, "sequencer", 1, "xyzabc")
	latestClaim1 := game1.DisputeLastBlock(ctx)
	latestClaim2 := game2.DisputeLastBlock(ctx)

	// Start a challenger with both cannon and alphabet support
	gameFactory.StartChallenger(ctx, "TowerDefense",
		challenger.WithOutputCannon(t, sys.RollupConfig, sys.L2GenesisCfg, sys.RollupEndpoint("sequencer"), sys.NodeEndpoint("sequencer")),
		challenger.WithOutputAlphabet(disputegame.CorrectAlphabet, sys.RollupEndpoint("sequencer")),
		challenger.WithPrivKey(sys.Cfg.Secrets.Alice),
	)

	// Wait for the challenger to respond to both games
	counter1 := latestClaim1.WaitForCounterClaim(ctx)
	counter2 := latestClaim2.WaitForCounterClaim(ctx)
	// The alphabet game always posts the same traces, so if they're different they can't both be from the alphabet.
	// We're contesting the same block with different VMs, so if the challenger was just playing two cannon or alphabet
	// games the responses would be equal.
	counter1.RequireDifferentClaimValue(counter2)
}
