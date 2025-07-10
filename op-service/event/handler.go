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
// Use Fn to target a specific event type.
type Handler struct {
	// Serve receives an event from the event system, to be processed.
	// The event system will automatically produce a global event for a context timeout.
	Fn func(ctx context.Context, ev Event)

	Key HandlerKey
}

func (h *Handler) Apply(opts ...HandlerOption) {
	for _, opt := range opts {
		opt.Apply(h)
	}
}

type HandlerOption interface {
	Apply(h *Handler)
}

// Fn is a typed handler, best used to specify a method,
// a function bound to a receiver, as handler to process a specific type of event.
// Event can be used as type, to make the Fn applicable to any event.
type Fn[E Event] func(ctx context.Context, ev E)

var _ HandlerOption = (Fn[Event])(nil)

func (fn Fn[E]) Apply(h *Handler) {
	h.Fn = fn.serve
	h.Key.EventType = reflect.TypeFor[E]()
}

func (fn Fn[E]) serve(ctx context.Context, ev Event) {
	v, ok := ev.(E)
	if !ok {
		panic(fmt.Errorf("typed handler called with unexpected type event %T", ev))
	}
	fn(ctx, v)
}

// ErrFn is a convenience version of Fn,
// to make an event handler with default error handling.
// Errors are handled with Reject.
type ErrFn[E Event] func(ctx context.Context, ev E) error

var _ HandlerOption = (ErrFn[Event])(nil)

func (fn ErrFn[E]) Apply(h *Handler) {
	h.Fn = fn.serve
	h.Key.EventType = reflect.TypeFor[E]()
}

func (fn ErrFn[E]) serve(ctx context.Context, ev Event) {
	v, ok := ev.(E)
	if !ok {
		panic(fmt.Errorf("typed handler called with unexpected type event %T", ev))
	}
	err := fn(ctx, v)
	if err != nil {
		Reject(ctx, err)
	}
}

func DebugFn[E Event](logger log.Logger) Fn[E] {
	return func(ctx context.Context, ev E) {
		logger.Debug("on-event", "event", ev)
	}
}

var NoopFn = Fn[Event](func(ctx context.Context, ev Event) {})
