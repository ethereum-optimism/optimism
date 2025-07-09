package event

type Event interface {
	// String returns the name of the event.
	// The name must be simple and identify the event type, not the event content.
	// This name is used for metric-labeling.
	String() string
}

type Drainer interface {
	// Drain processes all events.
	Drain() error
	// DrainUntil processes all events until a condition is hit.
	// If excl, the event that matches the condition is not processed yet.
	// If not excl, the event that matches is processed.
	DrainUntil(fn func(ev Event) bool, excl bool) error
}

type CriticalErrorEvent struct {
	Err error
}

var _ Event = CriticalErrorEvent{}

func (ev CriticalErrorEvent) String() string {
	return "critical-error"
}
