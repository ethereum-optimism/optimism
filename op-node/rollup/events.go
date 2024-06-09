package rollup

import "github.com/ethereum/go-ethereum/log"

type Event interface {
	String() string
}

type Deriver interface {
	OnEvent(ev Event)
}

type EventEmitter interface {
	Emit(ev Event)
}

type EmitterFunc func(ev Event)

func (fn EmitterFunc) Emit(ev Event) {
	fn(ev)
}

type EngineTemporaryErrorEvent struct {
	Err error
}

var _ Event = EngineTemporaryErrorEvent{}

func (ev EngineTemporaryErrorEvent) String() string {
	return "engine-temporary-error"
}

type ResetEvent struct {
	Err error
}

var _ Event = ResetEvent{}

func (ev ResetEvent) String() string {
	return "reset-event"
}

type CriticalErrorEvent struct {
	Err error
}

var _ Event = CriticalErrorEvent{}

func (ev CriticalErrorEvent) String() string {
	return "critical-error"
}

type SynchronousDerivers []Deriver

func (s *SynchronousDerivers) OnEvent(ev Event) {
	for _, d := range *s {
		d.OnEvent(ev)
	}
}

var _ Deriver = (*SynchronousDerivers)(nil)

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
type DeriverFunc func(ev Event)

func (fn DeriverFunc) OnEvent(ev Event) {
	fn(ev)
}
