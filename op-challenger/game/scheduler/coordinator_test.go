package scheduler

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-challenger/metrics"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"
)

func TestScheduleNewGames(t *testing.T) {
	c, workQueue, _, games, disk := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	ctx := context.Background()
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2, gameAddr3)))

	require.Len(t, workQueue, 3, "should schedule job for each game")
	require.Len(t, games.created, 3, "should have created players")
	var players []GamePlayer
	for i := 0; i < len(games.created); i++ {
		j := <-workQueue
		players = append(players, j.player)
	}
	for addr, player := range games.created {
		require.Equal(t, disk.DirForGame(addr), player.Dir, "should use allocated directory")
		require.Containsf(t, players, player, "should have created a job for player %v", addr)
	}
}

func TestSkipSchedulingInflightGames(t *testing.T) {
	c, workQueue, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	// Schedule the game once
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1)))
	require.Len(t, workQueue, 1, "should schedule game")

	// And then attempt to schedule again
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1)))
	require.Len(t, workQueue, 1, "should not reschedule in-flight game")
}

func TestExitWhenContextDoneWhileSchedulingJob(t *testing.T) {
	// No space in buffer to schedule a job
	c, workQueue, _, _, _ := setupCoordinatorTest(t, 0)
	gameAddr1 := common.Address{0xaa}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Context is cancelled

	// Should not block because the context is done.
	err := c.schedule(ctx, asGames(gameAddr1))
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, workQueue, "should not have been able to schedule game")
}

func TestSchedule_PrestateValidationErrors(t *testing.T) {
	c, _, _, games, _ := setupCoordinatorTest(t, 10)
	games.PrestateErr = fmt.Errorf("prestate error")
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	err := c.schedule(ctx, asGames(gameAddr1))
	require.Error(t, err)
}

func TestScheduleGameAgainAfterCompletion(t *testing.T) {
	c, workQueue, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	// Schedule the game once
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1)))
	require.Len(t, workQueue, 1, "should schedule game")

	// Read the job
	j := <-workQueue
	require.Len(t, workQueue, 0)

	// Process the result
	require.NoError(t, c.processResult(j))

	// And then attempt to schedule again
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1)))
	require.Len(t, workQueue, 1, "should reschedule completed game")
}

func TestResultForUnknownGame(t *testing.T) {
	c, _, _, _, _ := setupCoordinatorTest(t, 10)
	err := c.processResult(job{addr: common.Address{0xaa}})
	require.ErrorIs(t, err, errUnknownGame)
}

func TestProcessResultsWhileJobQueueFull(t *testing.T) {
	c, workQueue, resultQueue, games, disk := setupCoordinatorTest(t, 0)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	ctx := context.Background()

	// Create pre-existing data for all three games
	disk.DirForGame(gameAddr1)
	disk.DirForGame(gameAddr2)
	disk.DirForGame(gameAddr3)

	resultsSent := make(chan any)
	go func() {
		defer close(resultsSent)
		// Process three jobs then exit
		for i := 0; i < 3; i++ {
			j := <-workQueue
			resultQueue <- j
		}
	}()

	// Even though work queue length is only 1, should be able to schedule all three games
	// by reading and processing results
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2, gameAddr3)))
	require.Len(t, games.created, 3, "should have created 3 games")

loop:
	for {
		select {
		case <-resultQueue:
			// Drain any remaining results
		case <-resultsSent:
			break loop
		}
	}

	// Check that pre-existing directories weren't deleted.
	// This would fail if we start processing results before we've added all the required games to the state
	require.Empty(t, disk.deletedDirs, "should not have deleted any directories")
}

func TestDeleteDataForResolvedGames(t *testing.T) {
	c, workQueue, _, _, disk := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	ctx := context.Background()

	// First get game 3 marked as resolved
	require.NoError(t, c.schedule(ctx, asGames(gameAddr3)))
	require.Len(t, workQueue, 1)
	j := <-workQueue
	j.status = types.GameStatusDefenderWon
	require.NoError(t, c.processResult(j))
	// But ensure its data directory is marked as existing
	disk.DirForGame(gameAddr3)

	games := asGames(gameAddr1, gameAddr2, gameAddr3)
	require.NoError(t, c.schedule(ctx, games))

	// The work queue should only contain jobs for games 1 and 2
	// A resolved game should not be scheduled for an update.
	// This makes the inflight game metric more robust.
	require.Len(t, workQueue, 2, "should schedule all games")

	// Game 1 progresses and is still in progress
	// Game 2 progresses and is now resolved
	// Game 3 hasn't yet progressed (update is still in flight)
	for i := 0; i < len(games)-1; i++ {
		j := <-workQueue
		if j.addr == gameAddr2 {
			j.status = types.GameStatusDefenderWon
		}
		require.NoError(t, c.processResult(j))
	}

	require.True(t, disk.gameDirExists[gameAddr1], "game 1 data should be preserved (not resolved)")
	require.False(t, disk.gameDirExists[gameAddr2], "game 2 data should be deleted")
	require.True(t, disk.gameDirExists[gameAddr3], "game 3 data should be preserved (inflight)")
}

func TestDoNotDeleteDataForGameThatFailedToCreatePlayer(t *testing.T) {
	c, workQueue, _, games, disk := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	ctx := context.Background()

	games.creationFails = gameAddr1

	gameList := asGames(gameAddr1, gameAddr2)
	err := c.schedule(ctx, gameList)
	require.Error(t, err)

	// Game 1 won't be scheduled because the player failed to be created
	require.Len(t, workQueue, 1, "should schedule game 2")

	// Process game 2 result
	require.NoError(t, c.processResult(<-workQueue))

	require.True(t, disk.gameDirExists[gameAddr1], "game 1 data should be preserved")
	require.True(t, disk.gameDirExists[gameAddr2], "game 2 data should be preserved")

	// Should create player for game 1 next time its scheduled
	games.creationFails = common.Address{}
	require.NoError(t, c.schedule(ctx, gameList))
	require.Len(t, workQueue, len(gameList), "should schedule all games")

	j := <-workQueue
	require.Equal(t, gameAddr1, j.addr, "first job should be for first game")
	require.NotNil(t, j.player, "should have created player for game 1")
}

func TestDropOldGameStates(t *testing.T) {
	c, workQueue, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	gameAddr4 := common.Address{0xdd}
	ctx := context.Background()

	// Start tracking game 1, 2 and 3
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2, gameAddr3)))
	require.Len(t, workQueue, 3, "should schedule games")

	// Complete processing of games 1 and 2, leaving 3 in flight
	require.NoError(t, c.processResult(<-workQueue))
	require.NoError(t, c.processResult(<-workQueue))

	// Next update only has games 2 and 4
	require.NoError(t, c.schedule(ctx, asGames(gameAddr2, gameAddr4)))

	require.NotContains(t, c.states, gameAddr1, "should drop state for game 1")
	require.Contains(t, c.states, gameAddr2, "should keep state for game 2 (still active)")
	require.Contains(t, c.states, gameAddr3, "should keep state for game 3 (inflight)")
	require.Contains(t, c.states, gameAddr4, "should create state for game 4")
}

func setupCoordinatorTest(t *testing.T, bufferSize int) (*coordinator, <-chan job, chan job, *createdGames, *stubDiskManager) {
	logger := testlog.Logger(t, log.LvlInfo)
	workQueue := make(chan job, bufferSize)
	resultQueue := make(chan job, bufferSize)
	games := &createdGames{
		t:       t,
		created: make(map[common.Address]*test.StubGamePlayer),
	}
	disk := &stubDiskManager{gameDirExists: make(map[common.Address]bool)}
	c := newCoordinator(logger, metrics.NoopMetrics, workQueue, resultQueue, games.CreateGame, disk)
	return c, workQueue, resultQueue, games, disk
}

type createdGames struct {
	t               *testing.T
	createCompleted common.Address
	creationFails   common.Address
	created         map[common.Address]*test.StubGamePlayer
	PrestateErr     error
}

func (c *createdGames) CreateGame(fdg types.GameMetadata, dir string) (GamePlayer, error) {
	addr := fdg.Proxy
	if c.creationFails == addr {
		return nil, fmt.Errorf("refusing to create player for game: %v", addr)
	}
	if _, exists := c.created[addr]; exists {
		c.t.Fatalf("game %v already exists", addr)
	}
	status := types.GameStatusInProgress
	if addr == c.createCompleted {
		status = types.GameStatusDefenderWon
	}
	game := &test.StubGamePlayer{
		Addr:        addr,
		StatusValue: status,
		Dir:         dir,
	}
	if c.PrestateErr != nil {
		game.PrestateErr = c.PrestateErr
	}
	c.created[addr] = game
	return game, nil
}

type stubDiskManager struct {
	gameDirExists map[common.Address]bool
	deletedDirs   []common.Address
}

func (s *stubDiskManager) DirForGame(addr common.Address) string {
	s.gameDirExists[addr] = true
	return addr.Hex()
}

func (s *stubDiskManager) RemoveAllExcept(addrs []common.Address) error {
	for address := range s.gameDirExists {
		keep := slices.Contains(addrs, address)
		s.gameDirExists[address] = keep
		if !keep {
			s.deletedDirs = append(s.deletedDirs, address)
		}
	}
	return nil
}

func asGames(addrs ...common.Address) []types.GameMetadata {
	var games []types.GameMetadata
	for _, addr := range addrs {
		games = append(games, types.GameMetadata{
			Proxy: addr,
		})
	}
	return games
}
