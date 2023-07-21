package op_e2e

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-challenger/config"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/disputegame"
	"github.com/ethereum-optimism/optimism/op-service/client/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	game := disputeGameFactory.StartAlphabetGame(ctx, "zyxwvut")
	require.NotNil(t, game)

	game.WaitForGameStatus(ctx, disputegame.StatusInProgress)

	honest := game.StartChallenger(ctx, sys.NodeEndpoint("l1"), "honestAlice", func(c *config.Config) {
		c.AgreeWithProposedOutput = true // Agree with the proposed output, so disagree with the root claim
		c.AlphabetTrace = "abcdefg"
		c.TxMgrConfig.PrivateKey = hexutil.Encode(e2eutils.EncodePrivKey(cfg.Secrets.Alice))
	})
	defer honest.Close()

	game.WaitForClaimCount(ctx, 2)

	sys.TimeTravelClock.AdvanceTime(gameDuration)
	require.NoError(t, utils.WaitNextBlock(ctx, l1Client))

	// Challenger should resolve the game now that the clocks have expired.
	game.WaitForGameStatus(ctx, disputegame.StatusChallengerWins)
	require.NoError(t, honest.Close())
}
