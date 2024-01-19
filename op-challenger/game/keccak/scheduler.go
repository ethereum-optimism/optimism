package keccak

import (
	"context"
	"sync"

	"github.com/ethereum-optimism/optimism/op-challenger/game/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Verifier interface {
	Verify(ctx context.Context, oracle types.LargePreimageOracle, preimage types.LargePreimageMetaData)
}

type LargePreimageScheduler struct {
	log      log.Logger
	ch       chan common.Hash
	oracles  []types.LargePreimageOracle
	verifier Verifier
	cancel   func()
	wg       sync.WaitGroup
}

func NewLargePreimageScheduler(logger log.Logger, oracles []types.LargePreimageOracle, verifier Verifier) *LargePreimageScheduler {
	return &LargePreimageScheduler{
		log:      logger,
		ch:       make(chan common.Hash, 1),
		oracles:  oracles,
		verifier: verifier,
	}
}

func (s *LargePreimageScheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.run(ctx)
}

func (s *LargePreimageScheduler) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

func (s *LargePreimageScheduler) run(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case blockHash := <-s.ch:
			if err := s.verifyPreimages(ctx, blockHash); err != nil {
				s.log.Error("Failed to verify large preimages", "err", err)
			}
		}
	}
}

func (s *LargePreimageScheduler) Schedule(blockHash common.Hash, _ uint64) error {
	select {
	case s.ch <- blockHash:
	default:
		// Already busy processing, skip this update
	}
	return nil
}

func (s *LargePreimageScheduler) verifyPreimages(ctx context.Context, blockHash common.Hash) error {
	for _, oracle := range s.oracles {
		if err := s.verifyOraclePreimages(ctx, oracle, blockHash); err != nil {
			s.log.Error("Failed to verify preimages in oracle %v: %w", oracle.Addr(), err)
		}
	}
	return nil
}

func (s *LargePreimageScheduler) verifyOraclePreimages(ctx context.Context, oracle types.LargePreimageOracle, blockHash common.Hash) error {
	preimages, err := oracle.GetActivePreimages(ctx, blockHash)
	for _, preimage := range preimages {
		if preimage.ShouldVerify() {
			s.verifier.Verify(ctx, oracle, preimage)
		}
	}
	return err
}
