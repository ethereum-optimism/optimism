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

// L1TemporaryErrorEvent identifies a temporary issue with the L1 data.
type L1TemporaryErrorEvent struct {
	Err error
}

var _ Event = L1TemporaryErrorEvent{}

func (ev L1TemporaryErrorEvent) String() string {
	return "l1-temporary-error"
}

// EngineTemporaryErrorEvent identifies a temporary processing issue.
// It applies to both L1 and L2 data, often inter-related.
// This scope will be reduced over time, to only capture L2-engine specific temporary errors.
// See L1TemporaryErrorEvent for L1 related temporary errors.
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

type NoopEmitter struct{}

func (e NoopEmitter) Emit(ev Event) {}
