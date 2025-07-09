package event

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/log"
)

// Handler receives events, and processes them as task or observed event.
// Handler is over the general Event interface,
// so handlers for different types can be enumerated in bundles.
// Use HandlerFn to target a specific event type.
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

// HandlerErrFn is a convience version of HandlerFn,
// to make an event handler with default error handling.
// Errors are handled with Reject.
type HandlerErrFn[E Event] func(ctx context.Context, ev E) error

func (fn HandlerErrFn[E]) Serve(ctx context.Context, ev Event) {
	v, ok := ev.(E)
	if !ok {
		panic(fmt.Errorf("typed handler called with unexpected type event %T", ev))
	}
	err := fn(ctx, v)
	Reject(ctx, err)
}

func (fn HandlerErrFn[E]) EventType() reflect.Type {
	return reflect.TypeFor[E]()
}

// HandlerConfig represents the configuration that is
// accumulated by combining the HandlerOption arguments.
// This configuration configures when and how a handler is used in the event system.
type HandlerConfig struct {
	Filter Filter

	// We might add an option for parallel handling later
}

func (cfg *HandlerConfig) ApplyOpts(opts ...HandlerOption) {
	for _, opt := range opts {
		opt(cfg)
	}
}

// HandlerOption customizes when and how a handler is used in the event system.
type HandlerOption func(cfg *HandlerConfig)

func DebugDeriver[E Event](logger log.Logger) HandlerFn[E] {
	return func(ctx context.Context, ev E) {
		logger.Debug("on-event", "event", ev)
	}
}

var NoopHandler = HandlerFn[Event](func(ctx context.Context, ev Event) {})
