package clock

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdvancingClock_AdvancesByTimeBetweenTicks(t *testing.T) {
	clock, realTime := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()
	eventTicker := clock.NewTicker(1 * time.Second)

	start := clock.Now()
	realTime.AdvanceTime(1 * time.Second)
	require.Equal(t, start.Add(1*time.Second), <-eventTicker.Ch(), "should trigger events when advancing")
	require.Equal(t, start.Add(1*time.Second), clock.Now(), "Should advance on single tick")

	start = clock.Now()
	realTime.AdvanceTime(15 * time.Second)
	require.Equal(t, start.Add(15*time.Second), <-eventTicker.Ch(), "should trigger events when advancing")
	require.Equal(t, start.Add(15*time.Second), clock.Now(), "Should advance by time between ticks")
}

func TestAdvancingClock_Stop(t *testing.T) {
	clock, realTime := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()
	eventTicker := clock.NewTicker(1 * time.Second)

	clock.Stop()

	start := clock.Now()
	// Wall time still advances while stopped (Now() is lazy), but the
	// ticker must not fire.
	realTime.AdvanceTime(15 * time.Second)
	require.Equal(t, start.Add(15*time.Second), clock.Now(),
		"Now() should keep tracking wall time even while the scheduler is stopped")
	select {
	case <-eventTicker.Ch():
		t.Fatal("ticker fired while scheduler was stopped")
	default:
	}

	clock.Start()
	realTime.AdvanceTime(1 * time.Second)

	tick := <-eventTicker.Ch()
	// Reported time depends on whether the scheduler saw the second
	// AdvanceTime before firing; just bound it.
	require.GreaterOrEqual(t, tick.UnixNano(), start.Add(15*time.Second).UnixNano(),
		"ticker should report a time at or after the restart point")
	require.LessOrEqual(t, tick.UnixNano(), start.Add(16*time.Second).UnixNano(),
		"ticker should not report a time beyond Now()")
	require.Equal(t, start.Add(16*time.Second), clock.Now(),
		"Now() should reflect the total wall progression after restart")
}

// Now() returns a fresh time on every call, with no tick quantisation.
func TestAdvancingClock_NowIsContinuous(t *testing.T) {
	clock, realTime := newTestAdvancingClock()
	defer clock.Stop()
	start := clock.Now()

	realTime.AdvanceTime(123 * time.Microsecond)
	require.Equal(t, start.Add(123*time.Microsecond), clock.Now(),
		"Now() should reflect sub-tick wall-time progression immediately")

	realTime.AdvanceTime(7 * time.Nanosecond)
	require.Equal(t, start.Add(123*time.Microsecond+7*time.Nanosecond), clock.Now(),
		"Now() should not be quantized to any tick boundary")
}

// AdvanceTime jumps Now() and fires jumped-past actions without waiting.
func TestAdvancingClock_AdvanceTimeFiresImmediately(t *testing.T) {
	clock, _ := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	ch := clock.After(1 * time.Hour)
	start := clock.Now()
	clock.AdvanceTime(2 * time.Hour)

	select {
	case got := <-ch:
		require.False(t, got.Before(start.Add(2*time.Hour)),
			"After channel should fire with the new (jumped) logical time, got %v", got)
	case <-time.After(2 * time.Second):
		t.Fatal("After channel did not fire after AdvanceTime jumped past its due")
	}
	require.Equal(t, start.Add(2*time.Hour), clock.Now(),
		"Now() should reflect the manual jump immediately")
}

// Ticker fires when its period is jumped past.
func TestAdvancingClock_TickerFiresAfterJump(t *testing.T) {
	clock, _ := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	ticker := clock.NewTicker(1 * time.Second)
	defer ticker.Stop()

	clock.AdvanceTime(10 * time.Second)
	// Channel buffer is 1; extra ticks coalesce, per stdlib Ticker semantics.
	select {
	case <-ticker.Ch():
	case <-time.After(2 * time.Second):
		t.Fatal("ticker did not fire after 10s jump")
	}
}

// One AdvanceTime fires every action it jumped past.
func TestAdvancingClock_MultiplePending(t *testing.T) {
	clock, _ := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	a := clock.After(1 * time.Second)
	b := clock.After(2 * time.Second)
	c := clock.After(3 * time.Second)

	clock.AdvanceTime(5 * time.Second)

	deadline := time.After(2 * time.Second)
	for i, ch := range []<-chan time.Time{a, b, c} {
		select {
		case <-ch:
		case <-deadline:
			t.Fatalf("After channel %d did not fire", i)
		}
	}
}

// AfterFunc callback runs when AdvanceTime jumps past its due.
func TestAdvancingClock_AfterFuncCallback(t *testing.T) {
	clock, _ := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	done := make(chan struct{})
	clock.AfterFunc(1*time.Hour, func() { close(done) })
	clock.AdvanceTime(1 * time.Hour)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("AfterFunc callback did not run after AdvanceTime jumped past due")
	}
}

// A stopped timer never fires.
func TestAdvancingClock_TimerStop(t *testing.T) {
	clock, _ := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	timer := clock.NewTimer(1 * time.Hour)
	require.True(t, timer.Stop(), "Stop should report active")
	require.False(t, timer.Stop(), "Stop is idempotent")

	clock.AdvanceTime(2 * time.Hour)
	select {
	case <-timer.Ch():
		t.Fatal("stopped timer must not fire")
	case <-time.After(200 * time.Millisecond):
	}
}

// SleepCtx wakes when wall time advances past its duration.
func TestAdvancingClock_SleepCtx(t *testing.T) {
	clock, realTime := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	done := make(chan error, 1)
	go func() { done <- clock.SleepCtx(context.Background(), 5*time.Second) }()

	waitCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	require.True(t, clock.WaitForNewPendingAction(waitCtx),
		"SleepCtx should register a pending timer")

	realTime.AdvanceTime(5 * time.Second)

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("SleepCtx did not return after wall time advanced")
	}
}

// Concurrent AdvanceTime calls accumulate correctly.
func TestAdvancingClock_AdvanceTimeConcurrent(t *testing.T) {
	clock, _ := newTestAdvancingClock()
	defer clock.Stop()

	start := clock.Now()
	const goroutines = 32
	const perGoroutine = 100

	done := make(chan struct{}, goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < perGoroutine; j++ {
				clock.AdvanceTime(time.Millisecond)
			}
			done <- struct{}{}
		}()
	}
	for i := 0; i < goroutines; i++ {
		<-done
	}
	require.Equal(t, start.Add(goroutines*perGoroutine*time.Millisecond), clock.Now())
}

// Ticker.Reset(d) must update the period used for subsequent re-arms, not
// just the next due time. Matches stdlib time.Ticker.Reset.
func TestAdvancingClock_TickerResetUpdatesPeriod(t *testing.T) {
	clock, realTime := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	ticker := clock.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	ticker.Reset(1 * time.Second)

	// First tick at the new period.
	realTime.AdvanceTime(1 * time.Second)
	select {
	case <-ticker.Ch():
	case <-time.After(2 * time.Second):
		t.Fatal("ticker did not fire at new period after Reset")
	}

	// Re-arm must also use the new period, not the original 1h.
	realTime.AdvanceTime(1 * time.Second)
	select {
	case <-ticker.Ch():
	case <-time.After(2 * time.Second):
		t.Fatal("ticker did not re-arm at new period — Reset failed to update period")
	}
}

// AfterFunc must run f in its own goroutine so a slow/blocking callback does
// not stall other timers on the same clock. Matches stdlib time.AfterFunc.
func TestAdvancingClock_AfterFuncDoesNotBlockScheduler(t *testing.T) {
	clock, _ := newTestAdvancingClock()
	clock.Start()
	defer clock.Stop()

	block := make(chan struct{})
	defer close(block)

	clock.AfterFunc(10*time.Millisecond, func() { <-block })

	second := make(chan struct{})
	clock.AfterFunc(20*time.Millisecond, func() { close(second) })

	clock.AdvanceTime(1 * time.Second)

	select {
	case <-second:
	case <-time.After(2 * time.Second):
		t.Fatal("second AfterFunc did not fire — first callback blocked the scheduler")
	}
}

func newTestAdvancingClock() (*AdvancingClock, *DeterministicClock) {
	systemTime := NewDeterministicClock(time.UnixMilli(1000))
	return newAdvancingClock(systemTime), systemTime
}
