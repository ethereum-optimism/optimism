package opio

import (
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
