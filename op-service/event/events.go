package event

type Event interface {
	// String returns the name of the event.
	// The name must be simple and identify the event type, not the event content.
	// This name is used for metric-labeling.
	String() string
}

type CriticalErrorEvent struct {
	Err error
}

var _ Event = CriticalErrorEvent{}

func (ev CriticalErrorEvent) String() string {
	return "critical-error"
}
