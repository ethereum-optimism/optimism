package sync

import (
	"context"
	"errors"
	"sync"
)

// ErrChannelFull is returned when the scheduler's processing channel is full
// and a new item cannot be scheduled.
var ErrChannelFull = errors.New("channel full")

// SchedulerRunner is a function that processes items received by the [Scheduler].
type SchedulerRunner[T any] func(ctx context.Context, item T)

// Scheduler processes generic [T any] items using a provided runner function.
// Processing may be buffered with the [NewSchedulerFromBufferSize] constructor.
type Scheduler[T any] struct {
	receiver chan T
	runner   SchedulerRunner[T]
	cancel   func()
	wg       sync.WaitGroup
}

func NewSchedulerFromBufferSize[T any](runner SchedulerRunner[T], bufferSize int) *Scheduler[T] {
	return &Scheduler[T]{
		receiver: make(chan T, bufferSize),
		runner:   runner,
	}
}

// Start starts the scheduler.
func (s *Scheduler[T]) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.wg.Add(1)
	go s.run(ctx)
}

// Close stops the scheduler and waits for all in-flight processing to finish.
func (s *Scheduler[T]) Close() error {
	s.cancel()
	s.wg.Wait()
	return nil
}

// Drain drains the scheduler's processing channel.
func (s *Scheduler[T]) Drain() {
	for {
		select {
		case <-s.receiver:
		default:
			return
		}
	}
}

func (s *Scheduler[T]) run(ctx context.Context) {
	defer s.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case item := <-s.receiver:
			s.runner(ctx, item)
		}
	}
}

// Schedule sends an item to the scheduler for processing.
func (s *Scheduler[T]) Schedule(item T) error {
	select {
	case s.receiver <- item:
	default:
		return ErrChannelFull
	}
	return nil
}
