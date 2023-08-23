package fault

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum-optimism/optimism/op-service/clock"
)

func TestMonitorMinGameTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("zero game window returns zero", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Duration(0)
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("non-zero game window with zero clock", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		monitor.clock = clock.NewDeterministicClock(time.Unix(0, 0))
		require.Equal(t, monitor.minGameTimestamp(), uint64(0))
	})

	t.Run("minimum computed correctly", func(t *testing.T) {
		monitor, _, _, _, _ := setupMonitorTest(t, []common.Address{})
		monitor.gameWindow = time.Minute
		frozen := time.Unix(int64(time.Hour.Seconds()), 0)
		monitor.clock = clock.NewDeterministicClock(frozen)
		expected := uint64(frozen.Add(-time.Minute).Unix())
		require.Equal(t, monitor.minGameTimestamp(), expected)
	})
}

func TestMonitorCreateAndProgressGameAgents(t *testing.T) {
	monitor, source, games, _, _ := setupMonitorTest(t, []common.Address{})

	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	source.games = []FaultDisputeGame{newFDG(addr1, 9999), newFDG(addr2, 9999)}

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
	monitor, source, games, _, _ := setupMonitorTest(t, []common.Address{addr2})

	source.games = []FaultDisputeGame{newFDG(addr1, 9999), newFDG(addr2, 9999)}

	require.NoError(t, monitor.progressGames(context.Background(), uint64(1)))

	require.Len(t, games.created, 1, "should only create allowed game")
	require.Contains(t, games.created, addr2)
	require.NotContains(t, games.created, addr1)
	require.Equal(t, 1, games.created[addr2].progressCount)
}

// TestMonitorGames tests that the monitor can handle a new head event
// and resubscribe to new heads if the subscription errors.
func TestMonitorGames(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	monitor, source, games, _, mockHeadSource := setupMonitorTest(t, []common.Address{})

	source.games = []FaultDisputeGame{newFDG(addr1, 9999), newFDG(addr2, 9999)}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	monitor.MonitorGames(ctx)

	// Wait for a new header to be received
	waitErr := wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		if len(games.created) >= 2 {
			return true, nil
		}
		if mockHeadSource.sub == nil {
			return false, nil
		}
		mockHeadSource.sub.headers <- &ethtypes.Header{
			Number: big.NewInt(1),
		}
		return false, nil
	})
	require.NoError(t, waitErr)

	// Manually zero out the game players
	games.created = make(map[common.Address]*stubGame)
	monitor.players = make(map[common.Address]gamePlayer)

	// Send a subscription error
	require.NotNil(t, mockHeadSource.sub, "subscription should exist")
	mockHeadSource.sub.errChan <- ethereum.NotFound

	// Wait for a resubscription
	waitErr = wait.For(context.Background(), 500*time.Millisecond, func() (bool, error) {
		if len(games.created) >= 2 {
			return true, nil
		}
		if mockHeadSource.sub == nil {
			return false, nil
		}
		mockHeadSource.sub.headers <- &ethtypes.Header{
			Number: big.NewInt(1),
		}
		return false, nil
	})
	require.NoError(t, waitErr)
	require.Len(t, games.created, 2, "should create game agents")
	require.Contains(t, games.created, addr1)
	require.Contains(t, games.created, addr2)
	require.Equal(t, 1, games.created[addr1].progressCount)
	require.Equal(t, 1, games.created[addr2].progressCount)
}

func TestDeletePlayersWhenNoLongerInListOfGames(t *testing.T) {
	addr1 := common.Address{0xaa}
	addr2 := common.Address{0xbb}
	monitor, source, games, _, _ := setupMonitorTest(t, nil)

	allGames := []FaultDisputeGame{newFDG(addr1, 9999), newFDG(addr2, 9999)}
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

	monitor, source, games, disk, _ := setupMonitorTest(t, []common.Address{})
	games.createCompleted = addr1

	source.games = []FaultDisputeGame{newFDG(addr1, 9999), newFDG(addr2, 9999)}

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

func newFDG(proxy common.Address, timestamp uint64) FaultDisputeGame {
	return FaultDisputeGame{
		Proxy:     proxy,
		Timestamp: timestamp,
	}
}

func setupMonitorTest(t *testing.T, allowedGames []common.Address) (*gameMonitor, *stubGameSource, *createdGames, *stubDiskManager, *mockNewHeadSource) {
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

	mockHeadSource := &mockNewHeadSource{}
	monitor := newGameMonitor(logger, time.Duration(0), clock.SystemClock, disk, fetchBlockNum, allowedGames, source, games.CreateGame, mockHeadSource)
	return monitor, source, games, disk, mockHeadSource
}

type mockNewHeadSource struct {
	sub *mockSubscription
}

func (m *mockNewHeadSource) SubscribeNewHead(ctx context.Context, ch chan<- *ethtypes.Header) (ethereum.Subscription, error) {
	errChan := make(chan error)
	m.sub = &mockSubscription{errChan, ch}
	return m.sub, nil
}

type mockSubscription struct {
	errChan chan error
	headers chan<- *ethtypes.Header
}

func (m *mockSubscription) Unsubscribe() {}

func (m *mockSubscription) Err() <-chan error {
	return m.errChan
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
