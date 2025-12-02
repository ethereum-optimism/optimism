package event

import (
	"context"

	"github.com/ethereum/go-ethereum/log"
)

type Event interface {
	// String returns the name of the event.
	// The name must be simple and identify the event type, not the event content.
	// This name is used for metric-labeling.
	String() string
}

type Deriver interface {
	// OnEvent runs the event with the context that was used to emit the event.
	// The context is managed by the emitter,
	// and may be used to continue work after OnEvent if compatible with the emitter.
	// OnEvent returns true if it recognizes the event as "processed",
	// for tracing/metrics purposes primarily.
	OnEvent(ctx context.Context, ev Event) bool
}

type Emitter interface {
	// Emit emits an event, broadcasting it to all derivers (including the emitter itself).
	// The context is provided to the deriver OnEvent function.
	//
	// Events emitted by the same module will arrive in the same order as they were sent.
	// Across different emitters there’s no guarantee.
	Emit(ctx context.Context, ev Event)
}

type Drainer interface {
	// Drain processes all events.
	Drain() error
	// DrainUntil processes all events until a condition is hit.
	// If excl, the event that matches the condition is not processed yet.
	// If not excl, the event that matches is processed.
	DrainUntil(fn func(ev Event) bool, excl bool) error
}

type EmitterDrainer interface {
	Emitter
	Drainer
}

type EmitterFunc func(ctx context.Context, ev Event)

func (fn EmitterFunc) Emit(ctx context.Context, ev Event) {
	fn(ctx, ev)
}

// DeriverMux takes an event-signal as deriver, and synchronously fans it out to all contained Deriver ends.
// Technically this is a DeMux: single input to multi output.
type DeriverMux []Deriver

func (s *DeriverMux) OnEvent(ctx context.Context, ev Event) bool {
	out := false
	for _, d := range *s {
		out = d.OnEvent(ctx, ev) || out
	}
	return out
}

var _ Deriver = (*DeriverMux)(nil)

type DebugDeriver struct {
	Log log.Logger
}

func (d DebugDeriver) OnEvent(ctx context.Context, ev Event) {
	d.Log.Debug("on-event", "event", ev)
}

type NoopDeriver struct{}

func (d NoopDeriver) OnEvent(ctx context.Context, ev Event) {}

// DeriverFunc implements the Deriver interface as a function,
// similar to how the std-lib http HandlerFunc implements a Handler.
// This can be used for small in-place derivers, test helpers, etc.
type DeriverFunc func(ctx context.Context, ev Event) bool

func (fn DeriverFunc) OnEvent(ctx context.Context, ev Event) bool {
	return fn(ctx, ev)
}

type NoopEmitter struct{}

func (e NoopEmitter) Emit(ctx context.Context, ev Event) {}

type CriticalErrorEvent struct {
	Err error
}

var _ Event = CriticalErrorEvent{}

func (ev CriticalErrorEvent) String() string {
	return "critical-error"
}
