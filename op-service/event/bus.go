package event

// Bus represents the communication interface that
// is specific to an actor registered in the event system.
// All incoming and outgoing communication on the bus is synchronous within the scope of the bus,
// unless specified otherwise.
type Bus interface {
	// Handle registers an event-handler to receive events.
	Handle(handler Handler, opts ...HandlerOption)
	// Emitter returns an emitter that corresponds to the outgoing events of this bus,
	// and that is also attached to the call context that is passed into the event handler.
	Emitter() Emitter
}
