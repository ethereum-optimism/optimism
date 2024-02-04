package claims

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/log"
)

type BondClaimer interface {
	ClaimBonds(ctx context.Context, games []types.GameMetadata) error
}

type BondClaimScheduler struct {
	log     log.Logger
	metrics BondClaimSchedulerMetrics
	ch      chan schedulerMessage
	claimer BondClaimer
	cancel  func()
	wg      sync.WaitGroup
}

type BondClaimSchedulerMetrics interface {
	RecordBondClaimFailed()
}

type schedulerMessage struct {
	blockNumber uint64
	games       []types.GameMetadata
}

func NewBondClaimScheduler(logger log.Logger, metrics BondClaimSchedulerMetrics, claimer BondClaimer) *BondClaimScheduler {
	return &BondClaimScheduler{
		log:     logger,
		metrics: metrics,
		ch:      make(chan schedulerMessage, 1),
		claimer: claimer,
	}
}

func (s *BondClaimScheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *BondClaimScheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *BondClaimScheduler) run(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.ch:
			if err := s.claimer.ClaimBonds(ctx, msg.games); err != nil {
				s.metrics.RecordBondClaimFailed()
				s.log.Error("Failed to claim bonds", "blockNumber", msg.blockNumber, "err", err)
			}
		}
	}
}

func (s *BondClaimScheduler) Schedule(blockNumber uint64, games []types.GameMetadata) error {
	select {
	case s.ch <- schedulerMessage{blockNumber, games}:
	default:
		s.log.Trace("Skipping game bond claim while claiming in progress")
	}
	return nil
}
