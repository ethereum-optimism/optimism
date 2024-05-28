package scheduler

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/ethereum-optimism/optimism/op-challenger/game/scheduler/test"
	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestScheduleNewGames(t *testing.T) {
	c, workQueue, _, games, disk, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	ctx := context.Background()
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2, gameAddr3), 0))

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
	c, workQueue, _, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	// Schedule the game once
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1), 0))
	require.Len(t, workQueue, 1, "should schedule game")

	// And then attempt to schedule again
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1), 0))
	require.Len(t, workQueue, 1, "should not reschedule in-flight game")
}

func TestExitWhenContextDoneWhileSchedulingJob(t *testing.T) {
	// No space in buffer to schedule a job
	c, workQueue, _, _, _, _ := setupCoordinatorTest(t, 0)
	gameAddr1 := common.Address{0xaa}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Context is cancelled

	// Should not block because the context is done.
	err := c.schedule(ctx, asGames(gameAddr1), 0)
	require.ErrorIs(t, err, context.Canceled)
	require.Empty(t, workQueue, "should not have been able to schedule game")
}

func TestSchedule_PrestateValidationErrors(t *testing.T) {
	c, _, _, games, _, _ := setupCoordinatorTest(t, 10)
	games.PrestateErr = types.ErrInvalidPrestate
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	err := c.schedule(ctx, asGames(gameAddr1), 0)
	require.Error(t, err)
}

func TestSchedule_SkipPrestateValidationErrors(t *testing.T) {
	c, _, _, games, _, logs := setupCoordinatorTest(t, 10)
	c.allowInvalidPrestate = true
	games.PrestateErr = types.ErrInvalidPrestate
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	err := c.schedule(ctx, asGames(gameAddr1), 0)
	require.NoError(t, err)
	errLog := logs.FindLog(testlog.NewLevelFilter(log.LevelError), testlog.NewMessageFilter("Invalid prestate"))
	require.NotNil(t, errLog)
	require.Equal(t, errLog.AttrValue("game"), gameAddr1)
	require.Equal(t, errLog.AttrValue("err"), games.PrestateErr)
}

func TestSchedule_PrestateValidationFailure(t *testing.T) {
	c, _, _, games, _, _ := setupCoordinatorTest(t, 10)
	c.allowInvalidPrestate = true
	games.PrestateErr = fmt.Errorf("failed to fetch prestate")
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	err := c.schedule(ctx, asGames(gameAddr1), 0)
	require.ErrorIs(t, err, games.PrestateErr)
}

func TestScheduleGameAgainAfterCompletion(t *testing.T) {
	c, workQueue, _, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	ctx := context.Background()

	// Schedule the game once
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1), 0))
	require.Len(t, workQueue, 1, "should schedule game")

	// Read the job
	j := <-workQueue
	require.Len(t, workQueue, 0)

	// Process the result
	require.NoError(t, c.processResult(j))

	// And then attempt to schedule again
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1), 0))
	require.Len(t, workQueue, 1, "should reschedule completed game")
}

func TestResultForUnknownGame(t *testing.T) {
	c, _, _, _, _, _ := setupCoordinatorTest(t, 10)
	err := c.processResult(job{addr: common.Address{0xaa}})
	require.ErrorIs(t, err, errUnknownGame)
}

func TestProcessResultsWhileJobQueueFull(t *testing.T) {
	c, workQueue, resultQueue, games, disk, _ := setupCoordinatorTest(t, 0)
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
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2, gameAddr3), 0))
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
	c, workQueue, _, _, disk, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	ctx := context.Background()

	// First get game 3 marked as resolved
	require.NoError(t, c.schedule(ctx, asGames(gameAddr3), 0))
	require.Len(t, workQueue, 1)
	j := <-workQueue
	j.status = types.GameStatusDefenderWon
	require.NoError(t, c.processResult(j))
	// But ensure its data directory is marked as existing
	disk.DirForGame(gameAddr3)

	games := asGames(gameAddr1, gameAddr2, gameAddr3)
	require.NoError(t, c.schedule(ctx, games, 0))

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
	// Game 3 never got marked as in-flight because it was already resolved so got skipped.
	// We shouldn't be able to have a known-resolved game that is also in-flight because we always skip processing it.
	require.False(t, disk.gameDirExists[gameAddr3], "game 3 data should be deleted")
}

func TestSchedule_RecordActedL1Block(t *testing.T) {
	c, workQueue, _, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xcc}
	ctx := context.Background()

	// The first game should be tracked
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2), 1))

	// Process the result
	require.Len(t, workQueue, 2)
	j := <-workQueue
	require.Equal(t, gameAddr1, j.addr)
	j.status = types.GameStatusDefenderWon
	require.NoError(t, c.processResult(j))
	j = <-workQueue
	require.Equal(t, gameAddr2, j.addr)
	j.status = types.GameStatusInProgress
	require.NoError(t, c.processResult(j))

	// Schedule another block
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2), 2))

	// Process the result (only the in-progress game gets rescheduled)
	require.Len(t, workQueue, 1)
	j = <-workQueue
	require.Equal(t, gameAddr2, j.addr)
	require.Equal(t, uint64(2), j.block)
	j.status = types.GameStatusInProgress
	require.NoError(t, c.processResult(j))

	// Schedule a third block
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2), 3))

	// Process the result (only the in-progress game gets rescheduled)
	// This is deliberately done a third time, because there was actually a bug where it worked for the first two
	// cycles and failed on the third. This was because the first cycle the game status was unknown so it was processed
	// the second cycle was the first time the game was known to be complete so was skipped but crucially it left it
	// marked as in-flight.  On the third update the was incorrectly skipped as in-flight and the l1 block number
	// wasn't updated. From then on the block number would never be updated.
	require.Len(t, workQueue, 1)
	j = <-workQueue
	require.Equal(t, gameAddr2, j.addr)
	require.Equal(t, uint64(3), j.block)
	j.status = types.GameStatusInProgress
	require.NoError(t, c.processResult(j))

	// Schedule so that the metric is updated
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2), 4))

	// Verify that the block number is recorded by the metricer as acted upon
	require.Equal(t, uint64(3), c.m.(*stubSchedulerMetrics).actedL1Blocks)
}

func TestSchedule_RecordActedL1BlockMultipleGames(t *testing.T) {
	c, workQueue, _, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	ctx := context.Background()

	games := asGames(gameAddr1, gameAddr2, gameAddr3)
	require.NoError(t, c.schedule(ctx, games, 1))
	require.Len(t, workQueue, 3)

	// Game 1 progresses and is still in progress
	// Game 2 progresses and is now resolved
	// Game 3 hasn't yet progressed (update is still in flight)
	var game3Job job
	for i := 0; i < len(games); i++ {
		require.Equal(t, uint64(0), c.m.(*stubSchedulerMetrics).actedL1Blocks)
		j := <-workQueue
		if j.addr == gameAddr2 {
			j.status = types.GameStatusDefenderWon
		}
		if j.addr != gameAddr3 {
			require.NoError(t, c.processResult(j))
		} else {
			game3Job = j
		}
	}

	// Schedule so that the metric is updated
	require.NoError(t, c.schedule(ctx, games, 2))

	// Verify that block 1 isn't yet complete
	require.Equal(t, uint64(0), c.m.(*stubSchedulerMetrics).actedL1Blocks)

	// Complete processing game 3
	require.NoError(t, c.processResult(game3Job))

	// Schedule so that the metric is updated
	require.NoError(t, c.schedule(ctx, games, 3))

	// Verify that block 1 is now complete
	require.Equal(t, uint64(1), c.m.(*stubSchedulerMetrics).actedL1Blocks)
}

func TestSchedule_RecordActedL1BlockNewGame(t *testing.T) {
	c, workQueue, _, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	ctx := context.Background()

	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2), 1))
	require.Len(t, workQueue, 2)

	// Game 1 progresses and is still in progress
	// Game 2 progresses and is now resolved
	// Game 3 doesn't exist yet
	for i := 0; i < 2; i++ {
		require.Equal(t, uint64(0), c.m.(*stubSchedulerMetrics).actedL1Blocks)
		j := <-workQueue
		if j.addr == gameAddr2 {
			j.status = types.GameStatusDefenderWon
		}
		require.NoError(t, c.processResult(j))
	}

	// Schedule next block with game 3 now created
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2, gameAddr3), 2))

	// Verify that block 1 is now complete
	require.Equal(t, uint64(1), c.m.(*stubSchedulerMetrics).actedL1Blocks)
}

func TestDoNotDeleteDataForGameThatFailedToCreatePlayer(t *testing.T) {
	c, workQueue, _, games, disk, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	ctx := context.Background()

	games.creationFails = gameAddr1

	gameList := asGames(gameAddr1, gameAddr2)
	err := c.schedule(ctx, gameList, 0)
	require.Error(t, err)

	// Game 1 won't be scheduled because the player failed to be created
	require.Len(t, workQueue, 1, "should schedule game 2")

	// Process game 2 result
	require.NoError(t, c.processResult(<-workQueue))

	require.True(t, disk.gameDirExists[gameAddr1], "game 1 data should be preserved")
	require.True(t, disk.gameDirExists[gameAddr2], "game 2 data should be preserved")

	// Should create player for game 1 next time its scheduled
	games.creationFails = common.Address{}
	require.NoError(t, c.schedule(ctx, gameList, 0))
	require.Len(t, workQueue, len(gameList), "should schedule all games")

	j := <-workQueue
	require.Equal(t, gameAddr1, j.addr, "first job should be for first game")
	require.NotNil(t, j.player, "should have created player for game 1")
}

func TestDropOldGameStates(t *testing.T) {
	c, workQueue, _, _, _, _ := setupCoordinatorTest(t, 10)
	gameAddr1 := common.Address{0xaa}
	gameAddr2 := common.Address{0xbb}
	gameAddr3 := common.Address{0xcc}
	gameAddr4 := common.Address{0xdd}
	ctx := context.Background()

	// Start tracking game 1, 2 and 3
	require.NoError(t, c.schedule(ctx, asGames(gameAddr1, gameAddr2, gameAddr3), 0))
	require.Len(t, workQueue, 3, "should schedule games")

	// Complete processing of games 1 and 2, leaving 3 in flight
	require.NoError(t, c.processResult(<-workQueue))
	require.NoError(t, c.processResult(<-workQueue))

	// Next update only has games 2 and 4
	require.NoError(t, c.schedule(ctx, asGames(gameAddr2, gameAddr4), 0))

	require.NotContains(t, c.states, gameAddr1, "should drop state for game 1")
	require.Contains(t, c.states, gameAddr2, "should keep state for game 2 (still active)")
	require.Contains(t, c.states, gameAddr3, "should keep state for game 3 (inflight)")
	require.Contains(t, c.states, gameAddr4, "should create state for game 4")
}

func setupCoordinatorTest(t *testing.T, bufferSize int) (*coordinator, <-chan job, chan job, *createdGames, *stubDiskManager, *testlog.CapturingHandler) {
	logger, logs := testlog.CaptureLogger(t, log.LevelInfo)
	workQueue := make(chan job, bufferSize)
	resultQueue := make(chan job, bufferSize)
	games := &createdGames{
		t:       t,
		created: make(map[common.Address]*test.StubGamePlayer),
	}
	disk := &stubDiskManager{gameDirExists: make(map[common.Address]bool)}
	c := newCoordinator(logger, &stubSchedulerMetrics{}, workQueue, resultQueue, games.CreateGame, disk, false)
	return c, workQueue, resultQueue, games, disk, logs
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

type stubSchedulerMetrics struct {
	actedL1Blocks uint64
}

func (s *stubSchedulerMetrics) RecordActedL1Block(n uint64) {
	s.actedL1Blocks = n
}

func (s *stubSchedulerMetrics) RecordGamesStatus(_, _, _ int) {}
func (s *stubSchedulerMetrics) RecordGameUpdateScheduled()    {}
func (s *stubSchedulerMetrics) RecordGameUpdateCompleted()    {}

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
