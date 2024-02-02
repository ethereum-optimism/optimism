package claims

import (
	"context"
	"errors"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum-optimism/optimism/op-service/sync"
	"github.com/ethereum/go-ethereum/log"
)

type schedulerMessage struct {
	blockNumber uint64
	games       []types.GameMetadata
}

type Scheduler interface {
	Start(context.Context)
	Close() error
	Schedule(schedulerMessage) error
	Drain()
}

type BondClaimer interface {
	ClaimBonds(ctx context.Context, games []types.GameMetadata) error
}

type BondClaimScheduler struct {
	log       log.Logger
	metrics   BondClaimSchedulerMetrics
	scheduler Scheduler
}

type BondClaimSchedulerMetrics interface {
	RecordBondClaimFailed()
}

func newBondClaimRunner(claimer BondClaimer, logger log.Logger, metrics BondClaimSchedulerMetrics) sync.SchedulerRunner[schedulerMessage] {
	return func(ctx context.Context, msg schedulerMessage) {
		if err := claimer.ClaimBonds(ctx, msg.games); err != nil {
			metrics.RecordBondClaimFailed()
			logger.Error("Failed to claim bonds", "blockNumber", msg.blockNumber, "err", err)
		}
	}
}

func NewBondClaimScheduler(logger log.Logger, metrics BondClaimSchedulerMetrics, claimer BondClaimer) *BondClaimScheduler {
	runner := newBondClaimRunner(claimer, logger, metrics)
	return &BondClaimScheduler{
		log:       logger,
		metrics:   metrics,
		scheduler: sync.NewSchedulerFromBufferSize[schedulerMessage](runner, 1),
	}
}

func (s *BondClaimScheduler) Start(ctx context.Context) {
	s.scheduler.Start(ctx)
}

func (s *BondClaimScheduler) Close() error {
	if err := s.scheduler.Close(); err != nil {
		return err
	}
	s.scheduler.Drain()
	return nil
}

func (s *BondClaimScheduler) Schedule(blockNumber uint64, games []types.GameMetadata) error {
	if err := s.scheduler.Schedule(schedulerMessage{blockNumber, games}); errors.Is(err, sync.ErrChannelFull) {
		s.log.Trace("Skipping bond claim check while already processing")
	} else if err != nil {
		s.log.Error("Failed to schedule bond claim", "blockNumber", blockNumber, "error", err)
	}
	return nil
}
