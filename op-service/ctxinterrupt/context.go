package ctxinterrupt

import (
	"context"
)

// Newtyping empty struct prevents collision with other empty struct keys in the Context.
type interruptWaiterContextKeyType struct{}

var waiterContextKey = interruptWaiterContextKeyType{}

// WithInterruptWaiter overrides the interrupt waiter value, e.g. to insert a function that mocks
// interrupt signals for testing CLI shutdown without actual process signals.
func WithWaiterFunc(ctx context.Context, fn WaiterFunc) context.Context {
	return withInterruptWaiter(ctx, fn)
}

func withInterruptWaiter(ctx context.Context, value waiter) context.Context {
	return context.WithValue(ctx, waiterContextKey, value)
}

// contextInterruptWaiter returns a interruptWaiter that blocks on interrupts when called.
func contextInterruptWaiter(ctx context.Context) waiter {
	v := ctx.Value(waiterContextKey)
	if v == nil {
		return nil
	}
	return v.(waiter)
}
