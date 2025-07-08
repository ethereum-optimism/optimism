package event

import (
	"context"
	"errors"
)

// Emitter represents an entrypoint to insert new events into the event system.
// Default emitters are available when attached to the context.
// See global Emit function.
type Emitter interface {
	// Emit emits an event.
	//
	// An error is returned if the event could not be accepted by the event system.
	// The event system may choose to not accept an event
	// if the context is canceled before the event is passed on to a handler,
	// and return the context error.
	//
	// The event system may block until ready to accept the event,
	// to provide back-pressure to emitters of events.
	//
	// The context is passed on to the processors of the event.
	//
	// Events emitted by the same actor will arrive in the same order as they were sent.
	// Across different emitters there’s no guarantee.
	//
	// After the event is accepted into the system (but not necessarily processed yet)
	// emit returns nil: the event processing is not awaited.
	// To await completion of an event, see Await functionality instead.
	Emit(ctx context.Context, ev Event) error
}

// EmitterFunc implements Emitter
type EmitterFunc func(ctx context.Context, ev Event) error

func (fn EmitterFunc) Emit(ctx context.Context, ev Event) error {
	return fn(ctx, ev)
}

var _ Emitter = (EmitterFunc)(nil)

type emitterCtxKeyType struct{}

var emitterCtxKey = emitterCtxKeyType{}

// CtxWithEmitter attaches an emitter to the context, overriding any previously attached emitter.
func CtxWithEmitter(ctx context.Context, em Emitter) context.Context {
	return context.WithValue(ctx, emitterCtxKey, em)
}

// EmitterFromCtx returns the Emitter that is attached to the context with CtxWithEmitter.
// If no emitter was attached this returns nil.
func EmitterFromCtx(ctx context.Context) Emitter {
	v := ctx.Value(emitterCtxKey)
	if v == nil {
		return nil
	}
	return v.(Emitter)
}

// Emit emits the event to the emitter that is attached to the current context.
// If no emitter was set, the Emit will error.
func Emit(ctx context.Context, ev Event) error {
	em := EmitterFromCtx(ctx)
	if em == nil {
		return errors.New("no emitter in ctx")
	}
	return em.Emit(ctx, ev)
}
