package source

import (
	"context"
	"errors"
	"sync/atomic"
)

type PipelineEventHandler[E any] interface {
	Handle(ctx context.Context, event E)
}

type PipelineEventHandlerFn[E any] func(ctx context.Context, event E)

func (f PipelineEventHandlerFn[E]) Handle(ctx context.Context, event E) {
	f(ctx, event)
}

type PipelineStage[E any] struct {
	started atomic.Bool
	stopped chan interface{}
	cancel  context.CancelFunc
	in      <-chan E
	handler PipelineEventHandler[E]
}

func NewPipelineStage[E any](in <-chan E, handler PipelineEventHandler[E]) *PipelineStage[E] {
	return &PipelineStage[E]{
		in:      in,
		handler: handler,
	}
}

func (s *PipelineStage[E]) Start(ctx context.Context) error {
	if !s.started.CompareAndSwap(false, true) {
		return errors.New("stage already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.stopped = make(chan interface{})
	go s.loop(ctx)
	return nil
}

func (s *PipelineStage[E]) Stop(ctx context.Context) error {
	if !s.started.CompareAndSwap(true, false) {
		return errors.New("stage not started")
	}
	s.cancel()
	// Wait for the event loop to actually exit
	<-s.stopped
	return nil
}

func (s *PipelineStage[E]) loop(ctx context.Context) {
	defer close(s.stopped)
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-s.in:
			s.handler.Handle(ctx, evt)
		}
	}
}
