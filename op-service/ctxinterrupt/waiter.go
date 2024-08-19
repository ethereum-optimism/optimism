package ctxinterrupt

import (
	"context"
	"fmt"
)

// waiter describes a value that can wait for interrupts and context cancellation at the same time.
type waiter interface {
	waitForInterrupt(ctx context.Context) waitResult
}

// Waits for an interrupt or context cancellation. ctxErr should be the context.Cause of ctx when it
// is done. interrupt is only inspected if ctxErr is nil, and is not required to be set.
type WaiterFunc func(ctx context.Context) (interrupt, ctxErr error)

func (me WaiterFunc) waitForInterrupt(ctx context.Context) (res waitResult) {
	res.Interrupt, res.CtxError = me(ctx)
	return
}

// Either CtxError is not nil and is set to the context error cause, or the wait was interrupted.
type waitResult struct {
	// Not required to be non-nil on an interrupt.
	Interrupt error
	// Maybe set this using context.Cause.
	CtxError error
}

func (me waitResult) Cause() error {
	if me.CtxError != nil {
		return me.CtxError
	}
	if me.Interrupt != nil {
		// Do we really need to wrap the interrupt?
		return fmt.Errorf("interrupted: %w", me.Interrupt)
	}
	return nil
}
