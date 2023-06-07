// Package clock provides an abstraction for time to enable testing of functionality that uses time as an input.
package clock

import "time"

// Clock represents time in a way that can be provided by varying implementations.
// Methods are designed to be direct replacements for methods in the time package.
type Clock interface {
	// Now provides the current local time. Equivalent to time.Now
	Now() time.Time

	// After waits for the duration to elapse and then sends the current time on the returned channel.
	// It is equivalent to time.After
	After(d time.Duration) <-chan time.Time

	// NewTicker returns a new Ticker containing a channel that will send
	// the current time on the channel after each tick. The period of the
	// ticks is specified by the duration argument. The ticker will adjust
	// the time interval or drop ticks to make up for slow receivers.
	// The duration d must be greater than zero; if not, NewTicker will
	// panic. Stop the ticker to release associated resources.
	NewTicker(d time.Duration) Ticker
}

// A Ticker holds a channel that delivers "ticks" of a clock at intervals
type Ticker interface {
	// Ch returns the channel for the ticker. Equivalent to time.Ticker.C
	Ch() <-chan time.Time

	// Stop turns off a ticker. After Stop, no more ticks will be sent.
	// Stop does not close the channel, to prevent a concurrent goroutine
	// reading from the channel from seeing an erroneous "tick".
	Stop()

	// Reset stops a ticker and resets its period to the specified duration.
	// The next tick will arrive after the new period elapses. The duration d
	// must be greater than zero; if not, Reset will panic.
	Reset(d time.Duration)
}

// SystemClock provides an instance of Clock that uses the system clock via methods in the time package.
var SystemClock Clock = systemClock{}

type systemClock struct {
}

func (s systemClock) Now() time.Time {
	return time.Now()
}

func (s systemClock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}

type SystemTicker struct {
	*time.Ticker
}

func (t *SystemTicker) Ch() <-chan time.Time {
	return t.C
}

func (s systemClock) NewTicker(d time.Duration) Ticker {
	return &SystemTicker{time.NewTicker(d)}
}
