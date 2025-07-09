package event

import (
	"context"
	"slices"
	"sync"
	"time"
)

type AnnotatedEvent struct {
	Ctx          context.Context // Ctx passed in via Emit, and provided via executor to OnEvent handlers
	Event        Event
	EmitContext  uint64   // uniquely identifies the emission of the event, useful for debugging and creating diagrams
	EmitPriority Priority // how important the emitter is, higher is more important
}

func (e AnnotatedEvent) Equals(other AnnotatedEvent) bool {
	return e.Event == other.Event && e.EmitContext == other.EmitContext && e.EmitPriority == other.EmitPriority
}

type Tracer interface {
	OnDeriveStart(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time)
	OnDeriveEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool)
	OnEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time)
	OnAfterProcessed(evtype string)
}

type Tracers struct {
	tracers     []Tracer
	tracersLock sync.RWMutex
}

var _ Tracer = (*Tracers)(nil)

func (s *Tracers) AddTracer(t Tracer) {
	s.tracersLock.Lock()
	defer s.tracersLock.Unlock()
	s.tracers = append(s.tracers, t)
}

func (s *Tracers) RemoveTracer(t Tracer) {
	s.tracersLock.Lock()
	defer s.tracersLock.Unlock()
	// We are not removing tracers often enough to optimize the deletion;
	// instead we prefer fast and simple tracer iteration during regular operation.
	s.tracers = slices.DeleteFunc(s.tracers, func(v Tracer) bool {
		return v == t
	})
}

// OnDeriveStart records that the deriver by name is processing the event.
func (s *Tracers) OnDeriveStart(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time) {
	if s == nil {
		return
	}
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnDeriveStart(name, ev, derivContext, startTime)
	}
}

func (s *Tracers) OnDeriveEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool) {
	if s == nil {
		return
	}
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnDeriveEnd(name, ev, derivContext, startTime, duration, effect)
	}
}

func (s *Tracers) OnAfterProcessed(evtype string) {
	if s == nil {
		return
	}
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnAfterProcessed(evtype)
	}
}

func (s *Tracers) OnEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time) {
	if s == nil {
		return
	}
	s.tracersLock.RLock()
	defer s.tracersLock.RUnlock()
	for _, t := range s.tracers {
		t.OnEmit(name, ev, derivContext, emitTime)
	}
}
