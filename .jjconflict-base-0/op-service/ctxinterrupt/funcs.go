package ctxinterrupt

import (
	"context"
)

// Wait blocks until an interrupt is received, defaulting to interrupting on the default
// signals if no interrupt blocker is present in the Context. Returns nil if an interrupt occurs,
// else the Context error when it's done.
func Wait(ctx context.Context) error {
	iw := contextInterruptWaiter(ctx)
	if iw == nil {
		catcher := newSignalWaiter()
		defer catcher.Stop()
		iw = catcher
	}
	return iw.waitForInterrupt(ctx).CtxError
}

// WithSignalWaiter attaches an interrupt signal handler to the context which continues to receive
// signals after every wait, and also prevents the interrupt signals being handled before we're
// ready to wait for them. This helps functions wait on individual consecutive interrupts.
func WithSignalWaiter(ctx context.Context) (_ context.Context, stop func()) {
	if ctx.Value(waiterContextKey) != nil { // already has an interrupt waiter
		return ctx, func() {}
	}
	catcher := newSignalWaiter()
	return withInterruptWaiter(ctx, catcher), catcher.Stop
}

// WithSignalWaiterMain returns a Context with a signal interrupt blocker and leaks the destructor. Intended for use in
// main functions where we exit right after using the returned context anyway.
func WithSignalWaiterMain(ctx context.Context) context.Context {
	ctx, _ = WithSignalWaiter(ctx)
	return ctx
}

// WithCancelOnInterrupt returns a Context that is cancelled when Wait returns on the waiter in ctx.
// If there's no waiter, the default interrupt signals are used: In this case the signal hooking is
// not stopped until the original ctx is cancelled.
func WithCancelOnInterrupt(ctx context.Context) context.Context {
	interruptWaiter := contextInterruptWaiter(ctx)
	ctx, cancel := context.WithCancelCause(ctx)
	stop := func() {}
	if interruptWaiter == nil {
		catcher := newSignalWaiter()
		stop = catcher.Stop
		interruptWaiter = catcher
	}
	go func() {
		defer stop()
		cancel(interruptWaiter.waitForInterrupt(ctx).Cause())
	}()
	return ctx
}
