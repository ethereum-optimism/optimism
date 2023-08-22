package fault

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func TestMonitorExitsWhenContextDone(t *testing.T) {
	monitor, _, _ := setupMonitorTest(t, []common.Address{common.Address{}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := monitor.MonitorGames(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestMonitorCreateAndProgressGameAgents(t *testing.T) {
	monitor, source, games := setupMonitorTest(t, []common.Address{})

	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	source.games = []FaultDisputeGame{
		{
			Proxy:     addr1,
			Timestamp: 9999,
		},
		{
			Proxy:     addr2,
			Timestamp: 9999,
		},
	}

	err := monitor.progressGames(context.Background())
	require.NoError(t, err)

	require.Len(t, games.created, 2, "should create game agents")
	require.Contains(t, games.created, addr1)
	require.Contains(t, games.created, addr2)
	require.Equal(t, 1, games.created[addr1].progressCount)
	require.Equal(t, 1, games.created[addr2].progressCount)

	// The stub will fail the test if a game is created with the same address multiple times
	require.NoError(t, monitor.progressGames(context.Background()), "should only create games once")
	require.Equal(t, 2, games.created[addr1].progressCount)
	require.Equal(t, 2, games.created[addr2].progressCount)
}

func TestMonitorOnlyCreateSpecifiedGame(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	monitor, source, games := setupMonitorTest(t, []common.Address{addr2})

	source.games = []FaultDisputeGame{
		{
			Proxy:     addr1,
			Timestamp: 9999,
		},
		{
			Proxy:     addr2,
			Timestamp: 9999,
		},
	}

	err := monitor.progressGames(context.Background())
	require.NoError(t, err)

	require.Len(t, games.created, 1, "should only create allowed game")
	require.Contains(t, games.created, addr2)
	require.NotContains(t, games.created, addr1)
	require.Equal(t, 1, games.created[addr2].progressCount)
}

func setupMonitorTest(t *testing.T, allowedGames []common.Address) (*gameMonitor, *stubGameSource, *createdGames) {
	logger := testlog.Logger(t, log.LvlDebug)
	source := &stubGameSource{}
	games := &createdGames{
		t:       t,
		created: make(map[common.Address]*stubGame),
	}
	fetchBlockNum := func(ctx context.Context) (uint64, error) {
		return 1234, nil
	}
	monitor := newGameMonitor(logger, clock.SystemClock, fetchBlockNum, allowedGames, source, games.CreateGame)
	return monitor, source, games
}

type stubGameSource struct {
	games []FaultDisputeGame
}

func (s *stubGameSource) FetchAllGamesAtBlock(ctx context.Context, blockNumber *big.Int) ([]FaultDisputeGame, error) {
	return s.games, nil
}

type stubGame struct {
	addr          common.Address
	progressCount int
	done          bool
}

func (g *stubGame) ProgressGame(ctx context.Context) bool {
	g.progressCount++
	return g.done
}

type createdGames struct {
	t       *testing.T
	created map[common.Address]*stubGame
}

func (c *createdGames) CreateGame(addr common.Address) (gamePlayer, error) {
	if _, exists := c.created[addr]; exists {
		c.t.Fatalf("game %v already exists", addr)
	}
	game := &stubGame{addr: addr}
	c.created[addr] = game
	return game, nil
}
