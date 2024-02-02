package keccak

import (
	"context"
	"errors"
	"fmt"
	"time"

	faultTypes "github.com/ethereum-optimism/optimism/op-challenger/game/fault/types"
	keccakTypes "github.com/ethereum-optimism/optimism/op-challenger/game/keccak/types"
	"github.com/ethereum-optimism/optimism/op-service/sync"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type Scheduler interface {
	Start(context.Context)
	Close() error
	Schedule(common.Hash) error
	Drain()
}

type Challenger interface {
	Challenge(ctx context.Context, blockHash common.Hash, oracle Oracle, preimages []keccakTypes.LargePreimageMetaData) error
}

type LargePreimageScheduler struct {
	log       log.Logger
	scheduler Scheduler
}

func verifyOraclePreimages(ctx context.Context, cl faultTypes.ClockReader, oracle keccakTypes.LargePreimageOracle, blockHash common.Hash, challenger Challenger) error {
	preimages, err := oracle.GetActivePreimages(ctx, blockHash)
	if err != nil {
		return err
	}
	period, err := oracle.ChallengePeriod(ctx)
	if err != nil {
		return fmt.Errorf("failed to load challenge period: %w", err)
	}
	toVerify := make([]keccakTypes.LargePreimageMetaData, 0, len(preimages))
	for _, preimage := range preimages {
		if preimage.ShouldVerify(cl.Now(), time.Duration(period)*time.Second) {
			toVerify = append(toVerify, preimage)
		}
	}
	return challenger.Challenge(ctx, blockHash, oracle, toVerify)
}

func newVerifyPreimagesRunner(logger log.Logger, cl faultTypes.ClockReader, oracles []keccakTypes.LargePreimageOracle, challenger Challenger) sync.SchedulerRunner[common.Hash] {
	return func(ctx context.Context, blockHash common.Hash) {
		var err error
		for _, oracle := range oracles {
			err = errors.Join(err, verifyOraclePreimages(ctx, cl, oracle, blockHash, challenger))
		}
		if err != nil {
			logger.Error("Failed to verify large preimages", "blockHash", blockHash, "err", err)
		}
	}
}

func NewLargePreimageScheduler(logger log.Logger, cl faultTypes.ClockReader, oracles []keccakTypes.LargePreimageOracle, challenger Challenger) *LargePreimageScheduler {
	runner := newVerifyPreimagesRunner(logger, cl, oracles, challenger)
	return &LargePreimageScheduler{
		log:       logger,
		scheduler: sync.NewSchedulerFromBufferSize[common.Hash](runner, 1),
	}
}

func (s *LargePreimageScheduler) Start(ctx context.Context) {
	s.scheduler.Start(ctx)
}

func (s *LargePreimageScheduler) Close() error {
	if err := s.scheduler.Close(); err != nil {
		return err
	}
	s.scheduler.Drain()
	return nil
}

func (s *LargePreimageScheduler) Schedule(blockHash common.Hash) error {
	if err := s.scheduler.Schedule(blockHash); errors.Is(err, sync.ErrChannelFull) {
		s.log.Trace("Skipping preimage check while already processing")
	} else if err != nil {
		s.log.Error("Failed to schedule preimage verification", "error", err)
	}
	return nil
}
