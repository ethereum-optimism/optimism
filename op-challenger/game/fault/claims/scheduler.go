package claims

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/log"
)

type BondClaimerScheduler struct {
	log     log.Logger
	ch      chan schedulerMessage
	claimer BondClaimer
	cancel  func()
	wg      sync.WaitGroup
}

type schedulerMessage struct {
	blockNumber uint64
	games       []types.GameMetadata
}

func NewBondClaimerScheduler(logger log.Logger, claimer BondClaimer) *BondClaimerScheduler {
	return &BondClaimerScheduler{
		log:     logger,
		ch:      make(chan schedulerMessage, 1),
		claimer: claimer,
	}
}

func (s *BondClaimerScheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *BondClaimerScheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *BondClaimerScheduler) run(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-s.ch:
			if err := s.claimer.ClaimBonds(ctx, msg.games); err != nil {
				s.log.Error("Failed to claim bonds", "blockNumber", msg.blockNumber, "err", err)
			}
		}
	}
}

func (s *BondClaimerScheduler) Schedule(blockNumber uint64, games []types.GameMetadata) error {
	select {
	case s.ch <- schedulerMessage{blockNumber, games}:
	default:
		s.log.Trace("Skipping game bond claim while claiming in progress")
	}
	return nil
}
