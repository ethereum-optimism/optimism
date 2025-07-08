package event

import "context"

type Deriver interface {
	// OnEvent runs the event with the context that was used to emit the event.
	// The context is managed by the emitter,
	// and may be used to continue work after OnEvent if compatible with the emitter.
	// OnEvent returns true if it recognizes the event as "processed",
	// for tracing/metrics purposes primarily.
	OnEvent(ctx context.Context, ev Event) bool
}
