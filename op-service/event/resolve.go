package event

import (
	"context"
	"fmt"
)

// TODO: attach to ctx: the task / watching state

func CompleteTask(ctx context.Context) {
	// if task: resolve the task
	// if watching: warn
}

// Resolve emits an event as task resolution.
// Resolve will reject instead if the resolution failed to emit.
func Resolve[E Event](ctx context.Context, ev E) {
	// TODO
	// if task: resolve the task
	// if watching: warn, but emit
	err := Emit(ctx, ev)
	if err != nil {
		Reject(ctx, fmt.Errorf("failed to resolve: %w", err))
	} else {
		CompleteTask(ctx)
	}
}

// Reject rejects a task, or signals an error for a watched event.
// Reject itself does not create new events, and thus does not return an error like Emit.
func Reject(ctx context.Context, err error) {
	// TODO
	// if handling: report the error back
	// if watching: just log the error
}
