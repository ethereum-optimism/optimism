package fault

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/exp/slices"
)

var ErrBusy = errors.New("scheduler busy processing previous updates")

type gamePlayer interface {
	ProgressGame(ctx context.Context) bool
}

type playerCreator func(address common.Address, dir string) (gamePlayer, error)

type job struct {
	gameAddr     common.Address
	player       gamePlayer
	gameResolved bool
}

type gameState struct {
	scheduled bool
	resolved  bool
	player    gamePlayer
}

type GameScheduler struct {
	wg             sync.WaitGroup
	cancel         func()
	logger         log.Logger
	maxConcurrency int
	scheduleQueue  chan []common.Address
	runQueue       chan *job
	resultQueue    chan *job

	// Only safe to access from coordinatorLoop
	gameState    map[common.Address]*gameState
	disk         *diskManager
	createPlayer playerCreator
}

func NewGameScheduler(logger log.Logger, disk *diskManager, createPlayer playerCreator, maxConcurrency int) *GameScheduler {
	return &GameScheduler{
		logger:         logger,
		maxConcurrency: maxConcurrency,
		// Only one pending list of games to schedule.
		// When full, we just skip a new chain head and will schedule new games on the next block.
		scheduleQueue: make(chan []common.Address, 1),

		// TODO: Work out sizing for these queues
		// Currently keeping them relatively small. Big enough to keep the workers busy
		// but small enough to create backpressure so we don't fetch more games if we haven't
		// finished processing the previous batch as we'll just skip them all since the previous update is pending.
		runQueue:     make(chan *job, maxConcurrency*2),
		resultQueue:  make(chan *job, maxConcurrency*2),
		gameState:    make(map[common.Address]*gameState),
		disk:         disk,
		createPlayer: createPlayer,
	}
}

func (s *GameScheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	// Start coordinator
	s.wg.Add(1)
	go s.coordinatorLoop(ctx)

	// Start workers
	for i := 0; i < s.maxConcurrency; i++ {
		s.wg.Add(1)
		go s.workerLoop(ctx)
	}
}

func (s *GameScheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *GameScheduler) ProgressCurrentGames(ctx context.Context, games []common.Address) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case s.scheduleQueue <- games:
		return nil
	default:
		// If the queue is full, skip this head update
		// TODO: Might want to remove this so the monitor is blocked and doesn't move on to poll the factory again
		// On the other hand, skipping this update might mean we get new games from a later update...
		return ErrBusy
	}
}

func (s *GameScheduler) coordinatorLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case games := <-s.scheduleQueue:
			s.coordinateScheduling(ctx, games)
		case j := <-s.resultQueue:
			s.coordinateResult(j)
		}
	}
}

func (s *GameScheduler) coordinateScheduling(ctx context.Context, games []common.Address) {
	// Remove existing game states that aren't in the list of addresses as those games have expired
	for addr, state := range s.gameState {
		if !slices.Contains(games, addr) && !state.scheduled {
			delete(s.gameState, addr)
		}
	}

	// Update the game state for all games and create jobs for those that need to be played
	// We need to ensure all required games are tracked before we start executing any or we might delete
	// some files on disk right after startup because we get a result before all current games are added to the state
	var jobs []*job
	for _, addr := range games {
		state, ok := s.gameState[addr]
		if !ok {

			state = &gameState{}
			s.gameState[addr] = state
		}
		if state.scheduled {
			s.logger.Debug("Game update already scheduled, skipping", "game", addr)
			continue
		}
		if state.resolved {
			s.logger.Debug("Game already resolved, skipping update", "game", addr)
			continue
		}
		// Create the player separate to the gameState to ensure that the game state is present
		// even if creating the player fails. Otherwise, existing game data may be deleted because
		// of temporary problems requesting data
		if state.player == nil {
			player, err := s.createPlayer(addr, s.disk.DirForGame(addr))
			if err != nil {
				s.logger.Error("Could not create player for game", "game", addr, "err", err)
				continue
			}
			state.player = player
		}
		state.scheduled = true
		jobs = append(jobs, &job{
			gameAddr: addr,
			player:   state.player,
		})
	}
	for _, j := range jobs {
		select {
		case <-ctx.Done():
			return

		case s.runQueue <- j:

		// Process incoming results to avoid deadlock because the runQueue is full.
		case j := <-s.resultQueue:
			s.coordinateResult(j)
		}
	}
}

func (s *GameScheduler) coordinateResult(j *job) {
	state, ok := s.gameState[j.gameAddr]
	if !ok {
		s.logger.Error("Got result for untracked game", "game", j.gameAddr)
		return
	}
	state.resolved = j.gameResolved
	state.scheduled = false

	// Delete resources for any games that aren't currently scheduled and are resolved
	var keepData []common.Address
	for addr, state := range s.gameState {
		if !state.scheduled && state.resolved {
			// We can delete games that have been resolved and aren't scheduled to run.
			continue
		}
		keepData = append(keepData, addr)
	}
	if err := s.disk.RemoveAllExcept(keepData); err != nil {
		s.logger.Error("Unable to cleanup game data", "err", err)
	}
}

func (s *GameScheduler) workerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-s.runQueue:
			j.gameResolved = j.player.ProgressGame(ctx)
			select {
			case <-ctx.Done():
				return
			case s.resultQueue <- j:
			}
		}
	}
}
