package clock

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNowReturnsCurrentTime(t *testing.T) {
	now := time.UnixMilli(23829382)
	clock := NewDeterministicClock(now)
	require.Equal(t, now, clock.Now())
}

func TestAdvanceTime(t *testing.T) {
	start := time.UnixMilli(1000)
	clock := NewDeterministicClock(start)
	clock.AdvanceTime(500 * time.Millisecond)

	require.Equal(t, start.Add(500*time.Millisecond), clock.Now())
}

func TestAfter(t *testing.T) {
	t.Run("ZeroCompletesImmediately", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ch := clock.After(0)
		require.Len(t, ch, 1, "duration should already have been reached")
	})

	t.Run("CompletesWhenTimeAdvances", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ch := clock.After(500 * time.Millisecond)
		require.Len(t, ch, 0, "should not complete immediately")

		clock.AdvanceTime(499 * time.Millisecond)
		require.Len(t, ch, 0, "should not complete before time is due")

		clock.AdvanceTime(1 * time.Millisecond)
		require.Len(t, ch, 1, "should complete when time is reached")
		require.Equal(t, clock.Now(), <-ch)
	})

	t.Run("CompletesWhenTimeAdvancesPastDue", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ch := clock.After(500 * time.Millisecond)
		require.Len(t, ch, 0, "should not complete immediately")

		clock.AdvanceTime(9000 * time.Millisecond)
		require.Len(t, ch, 1, "should complete when time is past")
		require.Equal(t, clock.Now(), <-ch)
	})

	t.Run("RegisterAsPending", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		_ = clock.After(500 * time.Millisecond)

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		require.True(t, clock.WaitForNewPendingTask(ctx), "should have added a new pending task")
	})
}

func TestAfterFunc(t *testing.T) {
	t.Run("ZeroExecutesImmediately", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ran := new(atomic.Bool)
		timer := clock.AfterFunc(0, func() { ran.Store(true) })
		require.True(t, ran.Load(), "duration should already have been reached")
		require.False(t, timer.Stop(), "Stop should return false after executing")
	})

	t.Run("CompletesWhenTimeAdvances", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ran := new(atomic.Bool)
		timer := clock.AfterFunc(500*time.Millisecond, func() { ran.Store(true) })
		require.False(t, ran.Load(), "should not complete immediately")

		clock.AdvanceTime(499 * time.Millisecond)
		require.False(t, ran.Load(), "should not complete before time is due")

		clock.AdvanceTime(1 * time.Millisecond)
		require.True(t, ran.Load(), "should complete when time is reached")
		require.False(t, timer.Stop(), "Stop should return false after executing")
	})

	t.Run("CompletesWhenTimeAdvancesPastDue", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ran := new(atomic.Bool)
		timer := clock.AfterFunc(500*time.Millisecond, func() { ran.Store(true) })
		require.False(t, ran.Load(), "should not complete immediately")

		clock.AdvanceTime(9000 * time.Millisecond)
		require.True(t, ran.Load(), "should complete when time is reached")
		require.False(t, timer.Stop(), "Stop should return false after executing")
	})

	t.Run("RegisterAsPending", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ran := new(atomic.Bool)
		clock.AfterFunc(500*time.Millisecond, func() { ran.Store(true) })

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		require.True(t, clock.WaitForNewPendingTask(ctx), "should have added a new pending task")
	})

	t.Run("DoNotRunIfStopped", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ran := new(atomic.Bool)
		timer := clock.AfterFunc(500*time.Millisecond, func() { ran.Store(true) })
		require.False(t, ran.Load(), "should not complete immediately")

		require.True(t, timer.Stop(), "Stop should return true on first call")
		require.False(t, timer.Stop(), "Stop should return false on subsequent calls")

		clock.AdvanceTime(9000 * time.Millisecond)
		require.False(t, ran.Load(), "should not run when time is reached")
	})
}

func TestNewTicker(t *testing.T) {
	t.Run("FiresAfterEachDuration", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire immediately")

		clock.AdvanceTime(4 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire before due")

		clock.AdvanceTime(1 * time.Second)
		require.Len(t, ticker.Ch(), 1, "should fire when due")
		require.Equal(t, clock.Now(), <-ticker.Ch(), "should post current time")

		clock.AdvanceTime(4 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not re-fire before due")

		clock.AdvanceTime(1 * time.Second)
		require.Len(t, ticker.Ch(), 1, "should fire when due")
		require.Equal(t, clock.Now(), <-ticker.Ch(), "should post current time")
	})

	t.Run("SkipsFiringWhenAdvancedMultipleDurations", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire immediately")

		// Advance more than three periods, but should still only fire once
		clock.AdvanceTime(16 * time.Second)
		require.Len(t, ticker.Ch(), 1, "should fire when due")
		require.Equal(t, clock.Now(), <-ticker.Ch(), "should post current time")

		clock.AdvanceTime(1 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire until due again")
	})

	t.Run("SkipsFiringWhenProcessingIsSlow", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)

		// Fire once to fill the channel queue
		clock.AdvanceTime(5 * time.Second)
		firstEventTime := clock.Now()

		var startProcessing sync.WaitGroup
		startProcessing.Add(1)
		processedTicks := make(chan time.Time)
		go func() {
			startProcessing.Wait()
			// Read two events then exit
			for i := 0; i < 2; i++ {
				event := <-ticker.Ch()
				processedTicks <- event
			}
		}()

		// Advance time further before processing of events has started
		// Can't publish any further events to the channel but shouldn't block
		clock.AdvanceTime(30 * time.Second)

		// Allow processing to start
		startProcessing.Done()
		require.Equal(t, firstEventTime, <-processedTicks, "Should process first event")

		clock.AdvanceTime(5 * time.Second)
		require.Equal(t, clock.Now(), <-processedTicks, "Should skip to latest time")
	})

	t.Run("StopFiring", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)

		ticker.Stop()

		clock.AdvanceTime(10 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire after stop")
	})

	t.Run("ResetPanicWhenLessNotPositive", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)
		require.Panics(t, func() {
			ticker.Reset(0)
		}, "reset to 0 should panic")
		require.Panics(t, func() {
			ticker.Reset(-1)
		}, "reset to negative duration should panic")
	})

	t.Run("ResetWithShorterPeriod", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire immediately")

		ticker.Reset(1 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire immediately after reset")

		clock.AdvanceTime(1 * time.Second)
		require.Len(t, ticker.Ch(), 1, "should fire when new duration reached")
		require.Equal(t, clock.Now(), <-ticker.Ch(), "should post current time")
	})

	t.Run("ResetWithLongerPeriod", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire immediately")

		ticker.Reset(7 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire immediately after reset")

		clock.AdvanceTime(5 * time.Second)
		require.Len(t, ticker.Ch(), 0, "should not fire when old duration reached")

		clock.AdvanceTime(2 * time.Second)
		require.Len(t, ticker.Ch(), 1, "should fire when new duration reached")
		require.Equal(t, clock.Now(), <-ticker.Ch(), "should post current time")
	})

	t.Run("RegisterAsPending", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ticker := clock.NewTicker(5 * time.Second)
		defer ticker.Stop()

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		require.True(t, clock.WaitForNewPendingTask(ctx), "should have added a new pending task")
	})
}

func TestNewTimer(t *testing.T) {
	t.Run("FireOnceAfterDuration", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		timer := clock.NewTimer(5 * time.Second)

		require.Len(t, timer.Ch(), 0, "should not fire immediately")

		clock.AdvanceTime(4 * time.Second)
		require.Len(t, timer.Ch(), 0, "should not fire before due")

		clock.AdvanceTime(1 * time.Second)
		require.Len(t, timer.Ch(), 1, "should fire when due")
		require.Equal(t, clock.Now(), <-timer.Ch(), "should post current time")

		clock.AdvanceTime(6 * time.Second)
		require.Len(t, timer.Ch(), 0, "should not fire when due again")
	})

	t.Run("StopBeforeExecuted", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		timer := clock.NewTimer(5 * time.Second)

		require.True(t, timer.Stop())

		clock.AdvanceTime(10 * time.Second)
		require.Len(t, timer.Ch(), 0, "should not fire after stop")
	})

	t.Run("StopAfterExecuted", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		timer := clock.NewTimer(5 * time.Second)

		clock.AdvanceTime(10 * time.Second)
		require.Len(t, timer.Ch(), 1, "should fire when due")
		require.Equal(t, clock.Now(), <-timer.Ch(), "should post current time")

		require.False(t, timer.Stop())
	})
}

func TestWaitForPending(t *testing.T) {
	t.Run("DoNotBlockWhenAlreadyPending", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		_ = clock.After(5 * time.Minute)
		_ = clock.After(5 * time.Minute)

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		require.True(t, clock.WaitForNewPendingTask(ctx), "should have added a new pending task")
	})

	t.Run("ResetNewPendingFlagAfterWaiting", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		_ = clock.After(5 * time.Minute)
		_ = clock.After(5 * time.Minute)

		ctx, cancelFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFunc()
		require.True(t, clock.WaitForNewPendingTask(ctx), "should have added a new pending task")

		ctx, cancelFunc = context.WithTimeout(context.Background(), 250*time.Millisecond)
		defer cancelFunc()
		require.False(t, clock.WaitForNewPendingTask(ctx), "should have reset new pending task flag")
	})
}

func TestSleepCtx(t *testing.T) {
	t.Run("ReturnWhenContextComplete", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := clock.SleepCtx(ctx, 5*time.Minute)
		require.ErrorIs(t, err, context.Canceled)
	})

	t.Run("ReturnWhenDurationComplete", func(t *testing.T) {
		clock := NewDeterministicClock(time.UnixMilli(1000))
		var wg sync.WaitGroup
		var result atomic.Value
		wg.Add(1)
		go func() {
			err := clock.SleepCtx(context.Background(), 5*time.Minute)
			if err != nil {
				result.Store(err)
			}
			wg.Done()
		}()

		ctx, cancelFunc := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelFunc()
		// Wait until the SleepCtx is called and schedules a pending task
		clock.WaitForNewPendingTask(ctx)

		clock.AdvanceTime(5 * time.Minute)

		// Wait for the call to return
		wg.Wait()
		require.Nil(t, result.Load())
	})
}
