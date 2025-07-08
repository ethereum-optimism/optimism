package event

import (
	"context"
	"fmt"
	"reflect"
)

// Handler receives events. Handler can have extension interfaces to be more specific
type Handler interface {
	// Serve receives an event from the event system, to be processed.
	// The event system will automatically produce a global event for a context timeout.
	Serve(ctx context.Context, ev Event)

	// EventType returns the type of event to handle.
	// If the handler is not type-specific to one event,
	// then the general Event interface type itself is returned as type.
	// This interface is safe to use as map-key
	// (reflect package type-for/of functions return global type pointers specific per type).
	EventType() reflect.Type
}

var _ Handler = (HandlerFn[Event])(nil)

// HandlerFn is a typed handler, best used to specify a method,
// a function bound to a receiver, as handler to process a specific type of event.
// Event can be used as type, to make the HandlerFn applicable to any event.
type HandlerFn[E Event] func(ctx context.Context, ev E)

func (fn HandlerFn[E]) Serve(ctx context.Context, ev Event) {
	v, ok := ev.(E)
	if !ok {
		panic(fmt.Errorf("typed handler called with unexpected type event %T", ev))
	}
	fn(ctx, v)
}

func (fn HandlerFn[E]) EventType() reflect.Type {
	return reflect.TypeFor[E]()
}

// HandlerConfig represents the configuration that is
// accumulated by combining the HandlerOption arguments.
// This configuration configures when and how a handler is used in the event system.
type HandlerConfig struct {
	Filter Filter

	// We might add an option for parallel handling later
}

// HandlerOption customizes when and how a handler is used in the event system.
type HandlerOption func(cfg *HandlerConfig)
