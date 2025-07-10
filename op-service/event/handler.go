package event

import (
	"context"
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/log"
)

// HandlerKey identifies what a handler processes.
// A handler-key may be catch-all; all applicable keys are yielded by iterating the Variants.
type HandlerKey struct {
	// EventType returns the type of event to handle.
	// If the handler is not type-specific to one event,
	// then the general Event interface type itself is returned as type.
	//
	// The EventType is used as compile-time scope for a handler.
	//
	// The reflect.Type interface is safe to use as map-key
	// (reflect package type-for/of functions return global type pointers specific per type).
	EventType reflect.Type

	// Domain identifies the runtime scope for a handler.
	// If the handler is not domain-specific, UndefinedDomain is used.
	Domain Domain
}

var genericEventType = reflect.TypeFor[Event]()

// Variants iterates over aver all versions of the key: from most-specific, to most-generic.
func (k HandlerKey) Variants(yield func(key HandlerKey) bool) {
	if k.EventType != genericEventType {
		if k.Domain != UndefinedDomain {
			if !yield(k) {
				return
			}
		}
		if !yield(HandlerKey{
			EventType: k.EventType,
			Domain:    UndefinedDomain,
		}) {
			return
		}
	}
	if k.Domain != UndefinedDomain {
		if !yield(HandlerKey{
			EventType: genericEventType,
			Domain:    k.Domain,
		}) {
			return
		}
	}
	if !yield(HandlerKey{
		EventType: genericEventType,
		Domain:    UndefinedDomain,
	}) {
		return
	}
}

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

func DebugDeriver[E Event](logger log.Logger) Fn[E] {
	return func(ctx context.Context, ev E) {
		logger.Debug("on-event", "event", ev)
	}
}

var NoopHandler = Fn[Event](func(ctx context.Context, ev Event) {})
