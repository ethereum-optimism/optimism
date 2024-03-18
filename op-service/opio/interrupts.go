package opio

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// DefaultInterruptSignals is a set of default interrupt signals.
var DefaultInterruptSignals = []os.Signal{
	os.Interrupt,
	os.Kill,
	syscall.SIGTERM,
	syscall.SIGQUIT,
}

// BlockOnInterrupts blocks until a SIGTERM is received.
// Passing in signals will override the default signals.
func BlockOnInterrupts(signals ...os.Signal) {
	if len(signals) == 0 {
		signals = DefaultInterruptSignals
	}
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, signals...)
	<-interruptChannel
}

// BlockOnInterruptsContext blocks until a SIGTERM is received.
// Passing in signals will override the default signals.
// The function will stop blocking if the context is closed.
func BlockOnInterruptsContext(ctx context.Context, signals ...os.Signal) {
	if len(signals) == 0 {
		signals = DefaultInterruptSignals
	}
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, signals...)
	select {
	case <-interruptChannel:
	case <-ctx.Done():
		signal.Stop(interruptChannel)
	}
}

type interruptContextKeyType struct{}

var blockerContextKey = interruptContextKeyType{}

type interruptCatcher struct {
	incoming chan os.Signal
}

// Block blocks until either an interrupt signal is received, or the context is cancelled.
// No error is returned on interrupt.
func (c *interruptCatcher) Block(ctx context.Context) {
	select {
	case <-c.incoming:
	case <-ctx.Done():
	}
}

// WithInterruptBlocker attaches an interrupt handler to the context,
// which continues to receive signals after every block.
// This helps functions block on individual consecutive interrupts.
func WithInterruptBlocker(ctx context.Context) context.Context {
	if ctx.Value(blockerContextKey) != nil { // already has an interrupt handler
		return ctx
	}
	catcher := &interruptCatcher{
		incoming: make(chan os.Signal, 10),
	}
	signal.Notify(catcher.incoming, DefaultInterruptSignals...)

	return context.WithValue(ctx, blockerContextKey, BlockFn(catcher.Block))
}

// WithBlocker overrides the interrupt blocker value,
// e.g. to insert a block-function for testing CLI shutdown without actual process signals.
func WithBlocker(ctx context.Context, fn BlockFn) context.Context {
	return context.WithValue(ctx, blockerContextKey, fn)
}

// BlockFn simply blocks until the implementation of the blocker interrupts it, or till the given context is cancelled.
type BlockFn func(ctx context.Context)

// BlockerFromContext returns a BlockFn that blocks on interrupts when called.
func BlockerFromContext(ctx context.Context) BlockFn {
	v := ctx.Value(blockerContextKey)
	if v == nil {
		return nil
	}
	return v.(BlockFn)
}

// CancelOnInterrupt cancels the given context on interrupt.
// If a BlockFn is attached to the context, this is used as interrupt-blocking.
// If not, then the context blocks on a manually handled interrupt signal.
func CancelOnInterrupt(ctx context.Context) context.Context {
	inner, cancel := context.WithCancel(ctx)

	blockOnInterrupt := BlockerFromContext(ctx)
	if blockOnInterrupt == nil {
		blockOnInterrupt = func(ctx context.Context) {
			BlockOnInterruptsContext(ctx) // default signals
		}
	}

	go func() {
		blockOnInterrupt(ctx)
		cancel()
	}()

	return inner
}
