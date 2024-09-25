package event

import "github.com/ethereum/go-ethereum/log"

type Event interface {
	// String returns the name of the event.
	// The name must be simple and identify the event type, not the event content.
	// This name is used for metric-labeling.
	String() string
}

type Deriver interface {
	OnEvent(ev Event) bool
}

type Emitter interface {
	Emit(ev Event)
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

type EmitterFunc func(ev Event)

func (fn EmitterFunc) Emit(ev Event) {
	fn(ev)
}

// DeriverMux takes an event-signal as deriver, and synchronously fans it out to all contained Deriver ends.
// Technically this is a DeMux: single input to multi output.
type DeriverMux []Deriver

func (s *DeriverMux) OnEvent(ev Event) bool {
	out := false
	for _, d := range *s {
		out = d.OnEvent(ev) || out
	}
	return out
}

var _ Deriver = (*DeriverMux)(nil)

type DebugDeriver struct {
	Log log.Logger
}

func (d DebugDeriver) OnEvent(ev Event) {
	d.Log.Debug("on-event", "event", ev)
}

type NoopDeriver struct{}

func (d NoopDeriver) OnEvent(ev Event) {}

// DeriverFunc implements the Deriver interface as a function,
// similar to how the std-lib http HandlerFunc implements a Handler.
// This can be used for small in-place derivers, test helpers, etc.
type DeriverFunc func(ev Event) bool

func (fn DeriverFunc) OnEvent(ev Event) bool {
	return fn(ev)
}

type NoopEmitter struct{}

func (e NoopEmitter) Emit(ev Event) {}

type CriticalErrorEvent struct {
	Err error
}

var _ Event = CriticalErrorEvent{}

func (ev CriticalErrorEvent) String() string {
	return "critical-error"
}
