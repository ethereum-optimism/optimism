package event

import (
	"sync"
	"time"
)

type TraceEntryKind int

const (
	TraceDeriveStart TraceEntryKind = iota
	TraceDeriveEnd
	TraceRateLimited
	TraceEmit
)

type TraceEntry struct {
	Kind TraceEntryKind

	Name         string
	DerivContext uint64

	// Not present if Kind == TraceRateLimited
	EmitContext uint64
	// Not present if Kind == TraceRateLimited
	EventName string

	// Set to deriver start-time if derive-start/end, or emit-time if emitted. Not set if Kind == TraceRateLimited
	EventTime time.Time

	// Only present if Kind == TraceDeriveEnd
	DeriveEnd struct {
		Duration time.Duration
		Effect   bool
	}
}

type StructTracer struct {
	l sync.Mutex

	Entries []TraceEntry
}

var _ Tracer = (*StructTracer)(nil)

func NewStructTracer() *StructTracer {
	return &StructTracer{}
}

func (st *StructTracer) OnDeriveStart(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time) {
	st.l.Lock()
	defer st.l.Unlock()
	st.Entries = append(st.Entries, TraceEntry{
		Kind:         TraceDeriveStart,
		Name:         name,
		EventName:    ev.Event.String(),
		EmitContext:  ev.EmitContext,
		DerivContext: derivContext,
		EventTime:    startTime,
	})
}

func (st *StructTracer) OnDeriveEnd(name string, ev AnnotatedEvent, derivContext uint64, startTime time.Time, duration time.Duration, effect bool) {
	st.l.Lock()
	defer st.l.Unlock()
	st.Entries = append(st.Entries, TraceEntry{
		Kind:         TraceDeriveEnd,
		Name:         name,
		EventName:    ev.Event.String(),
		EmitContext:  ev.EmitContext,
		DerivContext: derivContext,
		EventTime:    startTime,
		DeriveEnd: struct {
			Duration time.Duration
			Effect   bool
		}{Duration: duration, Effect: effect},
	})
}

func (st *StructTracer) OnRateLimited(name string, derivContext uint64) {
	st.l.Lock()
	defer st.l.Unlock()
	st.Entries = append(st.Entries, TraceEntry{
		Kind:         TraceRateLimited,
		Name:         name,
		DerivContext: derivContext,
	})
}

func (st *StructTracer) OnEmit(name string, ev AnnotatedEvent, derivContext uint64, emitTime time.Time) {
	st.l.Lock()
	defer st.l.Unlock()
	st.Entries = append(st.Entries, TraceEntry{
		Kind:         TraceEmit,
		Name:         name,
		EventName:    ev.Event.String(),
		EmitContext:  ev.EmitContext,
		DerivContext: derivContext,
		EventTime:    emitTime,
	})
}
