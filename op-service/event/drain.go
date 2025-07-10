package event

// Drainer is an extension interface of an Executor,
// to make the executor process events.
type Drainer interface {
	// Drain processes all events.
	Drain() error
	// DrainUntil processes all events until a condition is hit.
	// If excl, the event that matches the condition is not processed yet.
	// If not excl, the event that matches is processed.
	DrainUntil(fn func(ev Event) bool, excl bool) error
}
