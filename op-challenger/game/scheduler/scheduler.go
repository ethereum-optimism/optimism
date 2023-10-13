package scheduler

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/log"
)

var ErrBusy = errors.New("busy scheduling previous update")

type SchedulerMetricer interface {
	RecordGamesStatus(inProgress, defenderWon, challengerWon int)
	RecordGameUpdateScheduled()
	RecordGameUpdateCompleted()
	IncActiveExecutors()
	DecActiveExecutors()
	IncIdleExecutors()
	DecIdleExecutors()
}

type Scheduler struct {
	logger         log.Logger
	coordinator    *coordinator
	m              SchedulerMetricer
	maxConcurrency uint
	scheduleQueue  chan []types.GameMetadata
	jobQueue       chan job
	resultQueue    chan job
	wg             sync.WaitGroup
	cancel         func()
}

func NewScheduler(logger log.Logger, m SchedulerMetricer, disk DiskManager, maxConcurrency uint, createPlayer PlayerCreator) *Scheduler {
	// Size job and results queues to be fairly small so backpressure is applied early
	// but with enough capacity to keep the workers busy
	jobQueue := make(chan job, maxConcurrency*2)
	resultQueue := make(chan job, maxConcurrency*2)

	// scheduleQueue has a size of 1 so backpressure quickly propagates to the caller
	// allowing them to potentially skip update cycles.
	scheduleQueue := make(chan []types.GameMetadata, 1)

	return &Scheduler{
		logger:         logger,
		m:              m,
		coordinator:    newCoordinator(logger, m, jobQueue, resultQueue, createPlayer, disk),
		maxConcurrency: maxConcurrency,
		scheduleQueue:  scheduleQueue,
		jobQueue:       jobQueue,
		resultQueue:    resultQueue,
	}
}

func (s *Scheduler) ThreadActive() {
	s.m.IncActiveExecutors()
	s.m.DecIdleExecutors()
}

func (s *Scheduler) ThreadIdle() {
	s.m.IncIdleExecutors()
	s.m.DecActiveExecutors()
}

func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	for i := uint(0); i < s.maxConcurrency; i++ {
		s.m.IncIdleExecutors()
		s.wg.Add(1)
		go progressGames(ctx, s.jobQueue, s.resultQueue, &s.wg, s.ThreadActive, s.ThreadIdle)
	}

	s.wg.Add(1)
	go s.loop(ctx)
}

func (s *Scheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *Scheduler) Schedule(games []types.GameMetadata) error {
	select {
	case s.scheduleQueue <- games:
		return nil
	default:
		return ErrBusy
	}
}

func (s *Scheduler) loop(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case games := <-s.scheduleQueue:
			if err := s.coordinator.schedule(ctx, games); err != nil {
				s.logger.Error("Failed to schedule game updates", "err", err)
			}
		case j := <-s.resultQueue:
			if err := s.coordinator.processResult(j); err != nil {
				s.logger.Error("Error while processing game result", "game", j.addr, "err", err)
			}
		}
	}
}
