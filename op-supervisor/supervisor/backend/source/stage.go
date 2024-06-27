package source

import (
	"context"
	"errors"
	"sync/atomic"
)

type PipelineEvent any

type PipelineEventHandler interface {
	Handle(ctx context.Context, event PipelineEvent, out chan<- PipelineEvent)
}

type PipelineEventHandlerFn func(ctx context.Context, event PipelineEvent, out chan<- PipelineEvent)

func (f PipelineEventHandlerFn) Handle(ctx context.Context, event PipelineEvent, out chan<- PipelineEvent) {
	f(ctx, event, out)
}

type PipelineStage struct {
	started atomic.Bool
	stopped chan interface{}
	cancel  context.CancelFunc
	in      <-chan PipelineEvent
	out     chan<- PipelineEvent
	handler PipelineEventHandler
}

func NewPipelineStage(in <-chan PipelineEvent, out chan<- PipelineEvent, handler PipelineEventHandler) *PipelineStage {
	return &PipelineStage{
		in:      in,
		out:     out,
		handler: handler,
	}
}

func (s *PipelineStage) Start(ctx context.Context) error {
	if !s.started.CompareAndSwap(false, true) {
		return errors.New("stage already started")
	}
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.stopped = make(chan interface{})
	go s.loop(ctx)
	return nil
}

func (s *PipelineStage) Stop(ctx context.Context) error {
	if !s.started.CompareAndSwap(true, false) {
		return errors.New("stage not started")
	}
	s.cancel()
	// Wait for the event loop to actually exit
	<-s.stopped
	return nil
}

func (s *PipelineStage) loop(ctx context.Context) {
	defer close(s.stopped)
	for {
		select {
		case <-ctx.Done():
			return
		case evt := <-s.in:
			s.handler.Handle(ctx, evt, s.out)
		}
	}
}
