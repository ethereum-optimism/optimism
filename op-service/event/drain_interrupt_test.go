package event

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// CascadeEvent triggers the cascade deriver to emit more events.
type CascadeEvent struct{ Remaining int }

func (ev CascadeEvent) String() string { return "cascade" }

// TestDrainUntilInterruptByCascade tests that DrainUntil's callback fires between
// events in a long cascade (where one event handler emits the next event),
// and that a timer channel is detected via len() peek.
func TestDrainUntilInterruptByCascade(t *testing.T) {
	exec := NewGlobalSynchronous(context.Background())

	eventsProcessed := 0
	// Cascade deriver: each CascadeEvent with Remaining > 0 enqueues the next one.
	cascade := ExecutableFunc(func(ev AnnotatedEvent) {
		eventsProcessed++
		if ce, ok := ev.Event.(CascadeEvent); ok && ce.Remaining > 0 {
			_ = exec.Enqueue(AnnotatedEvent{Event: CascadeEvent{Remaining: ce.Remaining - 1}})
		}
	})
	leave := exec.Add(cascade, &ExecutorConfig{Priority: Normal})
	defer leave()

	// Simulate a sequencer timer channel (buffered size 1, like time.Timer.C)
	timerCh := make(chan time.Time, 1)

	totalEvents := 1000
	fireAt := 500 // fire timer after 500 events

	// Enqueue the first cascade event
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: CascadeEvent{Remaining: totalEvents - 1}}))

	// DrainUntil with the peek check — same pattern as the driver.
	// Fire the timer into the channel when we've processed enough events.
	err := exec.DrainUntil(func(ev Event) bool {
		if eventsProcessed >= fireAt && len(timerCh) == 0 {
			timerCh <- time.Now()
		}
		return len(timerCh) > 0
	}, true)

	require.NoError(t, err, "should stop with nil (not io.EOF) when condition is met")
	require.Less(t, eventsProcessed, totalEvents, "should have stopped before processing all events")
	t.Logf("Processed %d/%d events before timer was detected", eventsProcessed, totalEvents)

	// The remaining events should still be in the queue
	// Drain them normally
	require.NoError(t, exec.Drain())
	require.Equal(t, totalEvents, eventsProcessed, "all events processed after full drain")
}

// TestDrainUntilInterruptExclLeavesEvent verifies excl=true leaves the triggering event in the queue.
func TestDrainUntilInterruptExclLeavesEvent(t *testing.T) {
	exec := NewGlobalSynchronous(context.Background())

	eventsProcessed := 0
	cascade := ExecutableFunc(func(ev AnnotatedEvent) {
		eventsProcessed++
		if ce, ok := ev.Event.(CascadeEvent); ok && ce.Remaining > 0 {
			_ = exec.Enqueue(AnnotatedEvent{Event: CascadeEvent{Remaining: ce.Remaining - 1}})
		}
	})
	leave := exec.Add(cascade, &ExecutorConfig{Priority: Normal})
	defer leave()

	timerCh := make(chan time.Time, 1)
	totalEvents := 100
	fireAt := 50

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: CascadeEvent{Remaining: totalEvents - 1}}))

	err := exec.DrainUntil(func(ev Event) bool {
		if eventsProcessed >= fireAt && len(timerCh) == 0 {
			timerCh <- time.Now()
		}
		return len(timerCh) > 0
	}, true) // excl=true: don't process the event that triggered the stop

	require.NoError(t, err)
	stoppedAt := eventsProcessed
	require.Equal(t, fireAt, stoppedAt, "stopped exactly at the fire point (excl=true means the event is not processed)")

	// The remaining events (50) should still be in the queue.
	// Drain the rest.
	require.NoError(t, exec.Drain())
	require.Equal(t, totalEvents, eventsProcessed, "all events processed after full drain")
}

// TestDrainUntilInterruptNilChannel tests that len(nil) == 0 and never triggers.
func TestDrainUntilInterruptNilChannel(t *testing.T) {
	exec := NewGlobalSynchronous(context.Background())

	eventsProcessed := 0
	cascade := ExecutableFunc(func(ev AnnotatedEvent) {
		eventsProcessed++
		if ce, ok := ev.Event.(CascadeEvent); ok && ce.Remaining > 0 {
			_ = exec.Enqueue(AnnotatedEvent{Event: CascadeEvent{Remaining: ce.Remaining - 1}})
		}
	})
	leave := exec.Add(cascade, &ExecutorConfig{Priority: Normal})
	defer leave()

	var nilCh <-chan time.Time // nil channel, like disabled sequencer

	totalEvents := 100
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: CascadeEvent{Remaining: totalEvents - 1}}))

	err := exec.DrainUntil(func(ev Event) bool {
		return len(nilCh) > 0
	}, true)

	require.ErrorIs(t, err, io.EOF, "should exhaust all events (nil channel never triggers)")
	require.Equal(t, totalEvents, eventsProcessed, "all events processed")
}
