package op_e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/stretchr/testify/require"
)

func TestResolveDisputeGame(t *testing.T) {
	InitParallel(t)

	ctx := context.Background()
	cfg := DefaultSystemConfig(t)
	cfg.DeployConfig.L1BlockTime = 1
	delete(cfg.Nodes, "verifier")
	delete(cfg.Nodes, "sequencer")
	cfg.SupportL1TimeTravel = true
	sys, err := cfg.Start()
	require.Nil(t, err, "Error starting up system")
	defer sys.Close()

	l1Client := sys.Clients["l1"]
	gameDuration := 24 * time.Hour
	disputeGameFactory := disputegame.NewFactoryHelper(t, ctx, l1Client, uint64(gameDuration.Seconds()))
	game := disputeGameFactory.StartAlphabetGame(ctx, "abcdefg")
	require.NotNil(t, game)

	game.AssertStatusEquals(disputegame.StatusInProgress)

	sys.TimeTravelClock.AdvanceTime(gameDuration)
	require.NoError(t, utils.WaitNextBlock(ctx, l1Client))

	game.Resolve(ctx)

	game.AssertStatusEquals(disputegame.StatusDefenderWins)
}
