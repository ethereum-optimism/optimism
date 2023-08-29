package scheduler

import (
	"context"
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

var ErrBusy = errors.New("busy scheduling previous update")

type Scheduler struct {
	logger         log.Logger
	coordinator    *coordinator
	maxConcurrency uint
	scheduleQueue  chan []common.Address
	jobQueue       chan job
	resultQueue    chan job
	wg             sync.WaitGroup
	cancel         func()
}

func NewScheduler(logger log.Logger, disk DiskManager, maxConcurrency uint, createPlayer PlayerCreator) *Scheduler {
	// Size job and results queues to be fairly small so backpressure is applied early
	// but with enough capacity to keep the workers busy
	jobQueue := make(chan job, maxConcurrency*2)
	resultQueue := make(chan job, maxConcurrency*2)

	// scheduleQueue has a size of 1 so backpressure quickly propagates to the caller
	// allowing them to potentially skip update cycles.
	scheduleQueue := make(chan []common.Address, 1)

	return &Scheduler{
		logger:         logger,
		coordinator:    newCoordinator(logger, jobQueue, resultQueue, createPlayer, disk),
		maxConcurrency: maxConcurrency,
		scheduleQueue:  scheduleQueue,
		jobQueue:       jobQueue,
		resultQueue:    resultQueue,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	for i := uint(0); i < s.maxConcurrency; i++ {
		s.wg.Add(1)
		go progressGames(ctx, s.jobQueue, s.resultQueue, &s.wg)
	}

	s.wg.Add(1)
	go s.loop(ctx)
}

func (s *Scheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *Scheduler) Schedule(games []common.Address) error {
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
				s.logger.Error("Failed to schedule game updates", "games", games, "err", err)
			}
		case j := <-s.resultQueue:
			if err := s.coordinator.processResult(j); err != nil {
				s.logger.Error("Error while processing game result", "game", j.addr, "err", err)
			}
		}
	}
}
