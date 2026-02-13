package keccak

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Challenger interface {
	Challenge(ctx context.Context, blockHash common.Hash, oracle Oracle, preimages []keccakTypes.LargePreimageMetaData) error
}

type OracleSource interface {
	Oracles() []keccakTypes.LargePreimageOracle
}

type Metrics interface {
	RecordLargePreimageCount(count int)
}

type LargePreimageScheduler struct {
	log        log.Logger
	m          Metrics
	cl         faultTypes.ClockReader
	ch         chan common.Hash
	oracles    OracleSource
	challenger Challenger
	cancel     func()
	wg         sync.WaitGroup
}

func NewLargePreimageScheduler(
	logger log.Logger,
	m Metrics,
	cl faultTypes.ClockReader,
	oracleSource OracleSource,
	challenger Challenger) *LargePreimageScheduler {
	return &LargePreimageScheduler{
		log:        logger,
		m:          m,
		cl:         cl,
		ch:         make(chan common.Hash, 1),
		oracles:    oracleSource,
		challenger: challenger,
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
				s.log.Error("Failed to verify large preimages", "blockHash", blockHash, "err", err)
			}
		}
	}
}

func (s *LargePreimageScheduler) Schedule(blockHash common.Hash, _ uint64) error {
	select {
	case s.ch <- blockHash:
	default:
		s.log.Trace("Skipping preimage check while already processing")
	}
	return nil
}

func (s *LargePreimageScheduler) verifyPreimages(ctx context.Context, blockHash common.Hash) error {
	var err error
	for _, oracle := range s.oracles.Oracles() {
		err = errors.Join(err, s.verifyOraclePreimages(ctx, oracle, blockHash))
	}
	return err
}

func (s *LargePreimageScheduler) verifyOraclePreimages(ctx context.Context, oracle keccakTypes.LargePreimageOracle, blockHash common.Hash) error {
	preimages, err := oracle.GetActivePreimages(ctx, blockHash)
	if err != nil {
		return err
	}
	s.m.RecordLargePreimageCount(len(preimages))
	period, err := oracle.ChallengePeriod(ctx)
	if err != nil {
		return fmt.Errorf("failed to load challenge period: %w", err)
	}
	toVerify := make([]keccakTypes.LargePreimageMetaData, 0, len(preimages))
	for _, preimage := range preimages {
		if preimage.ShouldVerify(s.cl.Now(), time.Duration(period)*time.Second) {
			toVerify = append(toVerify, preimage)
		}
	}
	return s.challenger.Challenge(ctx, blockHash, oracle, toVerify)
}
