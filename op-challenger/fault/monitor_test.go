package fault

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
)

func TestMonitorMinGameTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("zero game window returns zero", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Duration(0)
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("non-zero game window with zero clock", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		monitor.clock = clock.NewDeterministicClock(time.Unix(0, 0))
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("minimum computed correctly", func(t *testing.T) {
		monitor, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		frozen := time.Unix(int64(time.Hour.Seconds()), 0)
		monitor.clock = clock.NewDeterministicClock(frozen)
		expected := uint64(frozen.Add(-time.Minute).Unix())
		require.Equal(t, monitor.minGameTimestamp(), expected)
	})
}

func TestMonitorExitsWhenContextDone(t *testing.T) {
	monitor, _, _, _ := setupMonitorTest(t, []common.Address{{}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := monitor.MonitorGames(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestMonitorCreateAndProgressGameAgents(t *testing.T) {
	monitor, source, games, _ := setupMonitorTest(t, []common.Address{})

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

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))

	require.Len(t, games.created, 2, "should create game agents")
	require.Contains(t, games.created, addr1)
	require.Contains(t, games.created, addr2)
	require.Equal(t, 1, games.created[addr1].progressCount)
	require.Equal(t, 1, games.created[addr2].progressCount)

	// The stub will fail the test if a game is created with the same address multiple times
	require.NoError(t, monitor.progressGames(context.Background(), uint64(2)), "should only create games once")
	require.Equal(t, 2, games.created[addr1].progressCount)
	require.Equal(t, 2, games.created[addr2].progressCount)
}

func TestMonitorOnlyCreateSpecifiedGame(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	monitor, source, games, _ := setupMonitorTest(t, []common.Address{addr2})

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

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))

	require.Len(t, games.created, 1, "should only create allowed game")
	require.Contains(t, games.created, addr2)
	require.NotContains(t, games.created, addr1)
	require.Equal(t, 1, games.created[addr2].progressCount)
}

func TestDeletePlayersWhenNoLongerInListOfGames(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	monitor, source, games, _ := setupMonitorTest(t, nil)

	allGames := []FaultDisputeGame{
		{
			Proxy:     addr1,
			Timestamp: 9999,
		},
		{
			Proxy:     addr2,
			Timestamp: 9999,
		},
	}
	source.games = allGames

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))
	require.Len(t, games.created, 2)
	require.Contains(t, games.created, addr1)
	require.Contains(t, games.created, addr2)

	// First game is now old enough it's not returned in the list of active games
	source.games = source.games[1:]
	require.NoError(t, monitor.progressGames(context.Background(), uint64(2)))
	require.Len(t, games.created, 2)
	require.Contains(t, games.created, addr1)
	require.Contains(t, games.created, addr2)

	// Forget that we created the first game so it can be recreated if needed
	delete(games.created, addr1)

	// First game now reappears (inexplicably but usefully for our testing)
	source.games = allGames
	require.NoError(t, monitor.progressGames(context.Background(), uint64(3)))
	// A new player is created for it because the original was deleted
	require.Len(t, games.created, 2)
	require.Contains(t, games.created, addr1)
	require.Contains(t, games.created, addr2)
	require.Equal(t, 1, games.created[addr1].progressCount)
}

func TestCleanupResourcesOfCompletedGames(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}

	monitor, source, games, disk := setupMonitorTest(t, []common.Address{})
	games.createCompleted = addr1

	source.games = []FaultDisputeGame{
		{
			Proxy:     addr1,
			Timestamp: 1999,
		},
		{
			Proxy:     addr2,
			Timestamp: 9999,
		},
	}

	err := monitor.progressGames(context.Background(), uint64(1))
	require.NoError(t, err)

	require.Len(t, games.created, 2, "should create game agents")
	require.Contains(t, games.created, addr1)
	require.Contains(t, games.created, addr2)
	require.Equal(t, 1, games.created[addr1].progressCount)
	require.Equal(t, 1, games.created[addr2].progressCount)
	require.Contains(t, disk.gameDirExists, addr1, "should have allocated a game dir for game 1")
	require.False(t, disk.gameDirExists[addr1], "should have then deleted the game 1 dir")

	require.Contains(t, disk.gameDirExists, addr2, "should have allocated a game dir for game 2")
	require.True(t, disk.gameDirExists[addr2], "should not have deleted the game 2 dir")
}

func setupMonitorTest(t *testing.T, allowedGames []common.Address) (*gameMonitor, *stubGameSource, *createdGames, *stubDiskManager) {
	logger := testlog.Logger(t, log.LvlDebug)
	source := &stubGameSource{}
	games := &createdGames{
		t:       t,
		created: make(map[common.Address]*stubGame),
	}
	i := uint64(1)
	fetchBlockNum := func(ctx context.Context) (uint64, error) {
		i++
		return i, nil
	}
	disk := &stubDiskManager{
		gameDirExists: make(map[common.Address]bool),
	}
	monitor := newGameMonitor(logger, time.Duration(0), clock.SystemClock, disk, fetchBlockNum, allowedGames, source, games.CreateGame)
	return monitor, source, games, disk
}

type stubGameSource struct {
	games []FaultDisputeGame
}

func (s *stubGameSource) FetchAllGamesAtBlock(ctx context.Context, earliest uint64, blockNumber *big.Int) ([]FaultDisputeGame, error) {
	return s.games, nil
}

type stubGame struct {
	addr          common.Address
	progressCount int
	done          bool
	dir           string
}

func (g *stubGame) ProgressGame(ctx context.Context) bool {
	g.progressCount++
	return g.done
}

type createdGames struct {
	t               *testing.T
	createCompleted common.Address
	created         map[common.Address]*stubGame
}

func (c *createdGames) CreateGame(addr common.Address, dir string) (gamePlayer, error) {
	if _, exists := c.created[addr]; exists {
		c.t.Fatalf("game %v already exists", addr)
	}
	game := &stubGame{
		addr: addr,
		done: addr == c.createCompleted,
		dir:  dir,
	}
	c.created[addr] = game
	return game, nil
}

type stubDiskManager struct {
	gameDirExists map[common.Address]bool
}

func (s *stubDiskManager) DirForGame(addr common.Address) string {
	s.gameDirExists[addr] = true
	return addr.Hex()
}

func (s *stubDiskManager) RemoveAllExcept(addrs []common.Address) error {
	for address := range s.gameDirExists {
		s.gameDirExists[address] = slices.Contains(addrs, address)
	}
	return nil
}
