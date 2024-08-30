package ctxinterrupt

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// defaultSignals is a set of default interrupt signals.
var defaultSignals = []os.Signal{
	// Let's not catch SIGQUIT as it's expected to terminate with a stack trace in Go. os.Kill
	// should not/cannot be caught on most systems.
	os.Interrupt,
	syscall.SIGTERM,
}

type signalWaiter struct {
	incoming chan os.Signal
}

func newSignalWaiter() signalWaiter {
	catcher := signalWaiter{
		// Buffer, in case we are slow to act on older signals,
		// but still want to handle repeat-signals as special case (e.g. to force shutdown)
		incoming: make(chan os.Signal, 10),
	}
	signal.Notify(catcher.incoming, defaultSignals...)
	return catcher
}

func (me signalWaiter) Stop() {
	signal.Stop(me.incoming)
}

// Block blocks until either an interrupt signal is received, or the context is cancelled.
// No error is returned on interrupt.
func (me signalWaiter) waitForInterrupt(ctx context.Context) waitResult {
	select {
	case signalValue, ok := <-me.incoming:
		if !ok {
			// Signal channels are not closed.
			panic("signal channel closed")
		}
		return waitResult{Interrupt: fmt.Errorf("received interrupt signal %v", signalValue)}
	case <-ctx.Done():
		return waitResult{CtxError: context.Cause(ctx)}
	}
}
