package backpressure

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
)

type Backpressure interface {
	Start(context.Context)
	Ready() <-chan struct{}
	Adjust(bool)
}

type Task interface {
	Do(context.Context) error
}

type TaskFunc func(context.Context) error

var _ Task = (TaskFunc)(nil)

func (f TaskFunc) Do(ctx context.Context) error {
	return f(ctx)
}

type Schedule interface {
	Run(context.Context, Backpressure, Task)
}

type Burst struct {
	logger log.Logger
}

var _ Schedule = (*Burst)(nil)

func NewBurst(logger log.Logger) *Burst {
	return &Burst{
		logger: logger,
	}
}

func (b *Burst) Run(ctx context.Context, backpressure Backpressure, task Task) {
	var wg sync.WaitGroup
	defer wg.Wait()
	run(ctx, &wg, backpressure, task, func(err error) {
		backpressure.Adjust(err == nil)
		logOnError(b.logger, err)
	})
}

type InfoByLabel interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
}

type Steady struct {
	el                   InfoByLabel
	elasticityMultiplier uint64
	blockTime            time.Duration
	logger               log.Logger
}

var _ Schedule = (*Steady)(nil)

func NewSteady(logger log.Logger, blockTime time.Duration, el InfoByLabel, elasticityMultiplier uint64) *Steady {
	return &Steady{
		blockTime:            blockTime,
		el:                   el,
		elasticityMultiplier: elasticityMultiplier,
		logger:               logger,
	}
}

func (s *Steady) Run(ctx context.Context, backpressure Backpressure, task Task) {
	var wg sync.WaitGroup
	defer wg.Wait()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(s.blockTime)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				unsafe, err := s.el.InfoByLabel(ctx, eth.Unsafe)
				if err != nil {
					s.logger.Warn("Failed to read unsafe head", "err", err)
					continue
				}
				gasTarget := unsafe.GasLimit() / s.elasticityMultiplier
				// Apply backpressure when we meet or exceed the gas target.
				backpressure.Adjust(unsafe.GasUsed() < gasTarget)
			}
		}
	}()

	run(ctx, &wg, backpressure, task, func(err error) {
		logOnError(s.logger, err)
	})
}

type Constant struct {
	logger log.Logger
}

var _ Schedule = (*Constant)(nil)

func NewConstant(logger log.Logger) *Constant {
	return &Constant{
		logger: logger,
	}
}

func (c *Constant) Run(ctx context.Context, backpressure Backpressure, task Task) {
	var wg sync.WaitGroup
	defer wg.Wait()
	run(ctx, &wg, backpressure, task, func(err error) {
		logOnError(c.logger, err)
	})
}

func run(ctx context.Context, wg *sync.WaitGroup, backpressure Backpressure, task Task, onErr func(error)) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		backpressure.Start(ctx)
	}()

	for range backpressure.Ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			onErr(task.Do(ctx))
		}()
	}
}

func logOnError(logger log.Logger, err error) {
	if errors.Is(err, context.Canceled) {
		// Context cancelation is typically caused by the test ending, which is not really a
		// spammer error. Don't spam warnings in that case.
		logger.Debug("Task error", "err", err)
	} else if err != nil {
		logger.Warn("Task error", "err", err)
	}
}
