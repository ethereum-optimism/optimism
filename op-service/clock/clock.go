// Package clock provides an abstraction for time to enable testing of functionality that uses time as an input.
package clock

import (
	"context"
	"time"
)

// Clock represents time in a way that can be provided by varying implementations.
// Methods are designed to be direct replacements for methods in the time package,
// with some new additions to make common patterns simple.
type Clock interface {
	// Now provides the current local time. Equivalent to time.Now
	Now() time.Time

	// After waits for the duration to elapse and then sends the current time on the returned channel.
	// It is equivalent to time.After
	After(d time.Duration) <-chan time.Time

	AfterFunc(d time.Duration, f func()) Timer

	// NewTicker returns a new Ticker containing a channel that will send
	// the current time on the channel after each tick. The period of the
	// ticks is specified by the duration argument. The ticker will adjust
	// the time interval or drop ticks to make up for slow receivers.
	// The duration d must be greater than zero; if not, NewTicker will
	// panic. Stop the ticker to release associated resources.
	NewTicker(d time.Duration) Ticker

	// NewTimer creates a new Timer that will send
	// the current time on its channel after at least duration d.
	NewTimer(d time.Duration) Timer

	// SleepCtx sleeps until either ctx is done or the specified duration has elapsed.
	// Returns the ctx.Err if it returns because the context is done.
	SleepCtx(ctx context.Context, d time.Duration) error
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

// Timer represents a single event.
type Timer interface {
	// Ch returns the channel for the ticker. Equivalent to time.Timer.C
	Ch() <-chan time.Time

	// Stop prevents the Timer from firing.
	// It returns true if the call stops the timer, false if the timer has already
	// expired or been stopped.
	// Stop does not close the channel, to prevent a read from the channel succeeding
	// incorrectly.
	//
	// For a timer created with AfterFunc(d, f), if t.Stop returns false, then the timer
	// has already expired and the function f has been started in its own goroutine;
	// Stop does not wait for f to complete before returning.
	// If the caller needs to know whether f is completed, it must coordinate
	// with f explicitly.
	Stop() bool
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

func (s systemClock) NewTimer(d time.Duration) Timer {
	return &SystemTimer{time.NewTimer(d)}
}

type SystemTimer struct {
	*time.Timer
}

func (t *SystemTimer) Ch() <-chan time.Time {
	return t.C
}

func (s systemClock) AfterFunc(d time.Duration, f func()) Timer {
	return &SystemTimer{time.AfterFunc(d, f)}
}

func (s systemClock) SleepCtx(ctx context.Context, d time.Duration) error {
	timer := s.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.Ch():
		return nil
	}
}
