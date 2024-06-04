package scheduler

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var errUnknownGame = errors.New("unknown game")

type PlayerCreator func(game types.GameMetadata, dir string) (GamePlayer, error)

type CoordinatorMetricer interface {
	RecordActedL1Block(n uint64)
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
	RecordGameUpdateScheduled()
	RecordGameUpdateCompleted()
}

type gameState struct {
	player                GamePlayer
	inflight              bool
	lastProcessedBlockNum uint64
	status                types.GameStatus
}

// coordinator manages the set of current games, queues games to be played (on separate worker threads) and
// cleans up data files once a game is resolved.
// All function calls must be made on the same thread.
type coordinator struct {
	// jobQueue is the outgoing queue for jobs being sent to workers for progression
	jobQueue chan<- job

	// resultQueue is the incoming queue of jobs that have been completed by workers
	resultQueue <-chan job

	logger       log.Logger
	m            CoordinatorMetricer
	createPlayer PlayerCreator
	states       map[common.Address]*gameState
	disk         DiskManager

	allowInvalidPrestate bool

	// lastScheduledBlockNum is the highest block number that the coordinator has seen and scheduled jobs.
	lastScheduledBlockNum uint64
}

// schedule takes the current list of games to attempt to progress, filters out games that have previous
// progressions already in-flight and schedules jobs to progress on the outbound jobQueue.
// To avoid deadlock, it may process results from the inbound resultQueue while adding jobs to the outbound jobQueue.
// Returns an error if a game couldn't be scheduled because of an error. It will continue attempting to progress
// all games even if an error occurs with one game.
func (c *coordinator) schedule(ctx context.Context, games []types.GameMetadata, blockNumber uint64) error {
	// First remove any game states we no longer require
	for addr, state := range c.states {
		if !state.inflight && !slices.ContainsFunc(games, func(candidate types.GameMetadata) bool {
			return candidate.Proxy == addr
		}) {
			delete(c.states, addr)
		}
	}

	var gamesInProgress int
	var gamesChallengerWon int
	var gamesDefenderWon int
	var errs []error
	var jobs []job
	// Next collect all the jobs to schedule and ensure all games are recorded in the states map.
	// Otherwise, results may start being processed before all games are recorded, resulting in existing
	// data directories potentially being deleted for games that are required.
	for _, game := range games {
		if j, err := c.createJob(ctx, game, blockNumber); err != nil {
			errs = append(errs, fmt.Errorf("failed to create job for game %v: %w", game.Proxy, err))
		} else if j != nil {
			jobs = append(jobs, *j)
			c.m.RecordGameUpdateScheduled()
		}
		state, ok := c.states[game.Proxy]
		if ok {
			switch state.status {
			case types.GameStatusInProgress:
				gamesInProgress++
			case types.GameStatusDefenderWon:
				gamesDefenderWon++
			case types.GameStatusChallengerWon:
				gamesChallengerWon++
			}
		} else {
			c.logger.Warn("Game not found in states map", "game", game.Proxy)
		}
	}
	c.m.RecordGamesStatus(gamesInProgress, gamesDefenderWon, gamesChallengerWon)

	lowestProcessedBlockNum := blockNumber
	for _, state := range c.states {
		lowestProcessedBlockNum = min(lowestProcessedBlockNum, state.lastProcessedBlockNum)
	}
	c.lastScheduledBlockNum = blockNumber
	c.m.RecordActedL1Block(lowestProcessedBlockNum)

	// Finally, enqueue the jobs
	for _, j := range jobs {
		if err := c.enqueueJob(ctx, j); err != nil {
			errs = append(errs, fmt.Errorf("failed to enqueue job for game %v: %w", j.addr, err))
		}
	}
	return errors.Join(errs...)
}

// createJob updates the state for the specified game and returns the job to enqueue for it, if any
// Returns (nil, nil) when there is no error and no job to enqueue
func (c *coordinator) createJob(ctx context.Context, game types.GameMetadata, blockNumber uint64) (*job, error) {
	state, ok := c.states[game.Proxy]
	if !ok {
		// This is the first time we're seeing this game, so its last processed block
		// is the last block the coordinator processed (it didn't exist yet).
		state = &gameState{lastProcessedBlockNum: c.lastScheduledBlockNum}
		c.states[game.Proxy] = state
	}
	if state.inflight {
		c.logger.Debug("Not rescheduling already in-flight game", "game", game.Proxy)
		return nil, nil
	}
	// Create the player separately to the state so we retry creating it if it fails on the first attempt.
	if state.player == nil {
		player, err := c.createPlayer(game, c.disk.DirForGame(game.Proxy))
		if err != nil {
			return nil, fmt.Errorf("failed to create game player: %w", err)
		}
		if err := player.ValidatePrestate(ctx); err != nil {
			if !c.allowInvalidPrestate || !errors.Is(err, types.ErrInvalidPrestate) {
				return nil, fmt.Errorf("failed to validate prestate: %w", err)
			}
			c.logger.Error("Invalid prestate", "game", game.Proxy, "err", err)
		}
		state.player = player
		state.status = player.Status()
	}
	if state.status != types.GameStatusInProgress {
		c.logger.Debug("Not rescheduling resolved game", "game", game.Proxy, "status", state.status)
		state.lastProcessedBlockNum = blockNumber
		return nil, nil
	}
	state.inflight = true
	return newJob(blockNumber, game.Proxy, state.player, state.status), nil
}

func (c *coordinator) enqueueJob(ctx context.Context, j job) error {
	for {
		select {
		case c.jobQueue <- j:
			return nil
		case result := <-c.resultQueue:
			if err := c.processResult(result); err != nil {
				c.logger.Error("Failed to process result", "err", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (c *coordinator) processResult(j job) error {
	state, ok := c.states[j.addr]
	if !ok {
		return fmt.Errorf("game %v received unexpected result: %w", j.addr, errUnknownGame)
	}
	state.inflight = false
	state.status = j.status
	state.lastProcessedBlockNum = j.block
	c.deleteResolvedGameFiles()
	c.m.RecordGameUpdateCompleted()
	return nil
}

func (c *coordinator) deleteResolvedGameFiles() {
	var keepGames []common.Address
	for addr, state := range c.states {
		if state.status == types.GameStatusInProgress || state.inflight {
			keepGames = append(keepGames, addr)
		}
	}
	if err := c.disk.RemoveAllExcept(keepGames); err != nil {
		c.logger.Error("Unable to cleanup game data", "err", err)
	}
}

func newCoordinator(logger log.Logger, m CoordinatorMetricer, jobQueue chan<- job, resultQueue <-chan job, createPlayer PlayerCreator, disk DiskManager, allowInvalidPrestate bool) *coordinator {
	return &coordinator{
		logger:               logger,
		m:                    m,
		jobQueue:             jobQueue,
		resultQueue:          resultQueue,
		createPlayer:         createPlayer,
		disk:                 disk,
		states:               make(map[common.Address]*gameState),
		allowInvalidPrestate: allowInvalidPrestate,
	}
}
