package event

// Actor represents an independent module in the event system.
//
// Actors are registered with System.Register().
//
// The actor is provided with a Bus to interact with the event system.
//
// The event executor should run the events of the bus synchronously,
// in original order (no order guarantees between different emitters however).
type Actor interface {
	// Events sets up an actor to interact with the event system.
	Events(bus Bus)
}
