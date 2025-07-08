package event

// CtxClosedEvent is emitted by the event system
// whenever the context used to emit an event closes prematurely.
// The event itself is empty; the context itself is already passed on.
type CtxClosedEvent struct{}

func (ev CtxClosedEvent) String() string {
	return "ctx-closed"
}
