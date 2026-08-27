package withdrawals

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/log"
)

type InvalidatedWithdrawalDeleter interface {
	DeleteInvalidatedWithdrawals(ctx context.Context, blockNumber uint64, games []types.GameMetadata) error
}

type SchedulerMetrics interface {
	RecordWithdrawalDeletionFailed()
}

type schedulerMessage struct {
	blockNumber uint64
	games       []types.GameMetadata
}

type Scheduler struct {
	log     log.Logger
	metrics SchedulerMetrics
	ch      chan schedulerMessage
	deleter InvalidatedWithdrawalDeleter
	cancel  func()
	wg      sync.WaitGroup
}

func NewScheduler(logger log.Logger, metrics SchedulerMetrics, deleter InvalidatedWithdrawalDeleter) *Scheduler {
	return &Scheduler{
		log:     logger,
		metrics: metrics,
		ch:      make(chan schedulerMessage, 1),
		deleter: deleter,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *Scheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *Scheduler) run(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.ch:
			if err := s.deleter.DeleteInvalidatedWithdrawals(ctx, msg.blockNumber, msg.games); err != nil {
				s.metrics.RecordWithdrawalDeletionFailed()
				s.log.Error("Failed to delete invalidated withdrawals", "blockNumber", msg.blockNumber, "err", err)
			}
		}
	}
}

func (s *Scheduler) Schedule(blockNumber uint64, games []types.GameMetadata) error {
	select {
	case s.ch <- schedulerMessage{blockNumber, games}:
	default:
		s.log.Trace("Skipping withdrawal deletion while previous run in progress")
	}
	return nil
}
