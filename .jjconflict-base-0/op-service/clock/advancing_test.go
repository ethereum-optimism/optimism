package clock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAdvancingClock_AdvancesByTimeBetweenTicks(t *testing.T) {
	clock, realTime := newTestAdvancingClock(1 * time.Second)
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
	clock, realTime := newTestAdvancingClock(1 * time.Second)
	clock.Start()
	defer clock.Stop()
	eventTicker := clock.NewTicker(1 * time.Second)

	// Stop the clock again
	clock.Stop()

	start := clock.Now()
	realTime.AdvanceTime(15 * time.Second)

	clock.Start()
	// Trigger the next tick
	realTime.AdvanceTime(1 * time.Second)
	// Time advances by the whole time the clock was stopped
	// Note: if events were triggered while the clock was stopped, this event would be for the wrong time
	require.Equal(t, start.Add(16*time.Second), <-eventTicker.Ch(), "should trigger events again after restarting")
	require.Equal(t, start.Add(16*time.Second), clock.Now(), "Should advance by time between ticks after restarting")
}

func newTestAdvancingClock(advanceEvery time.Duration) (*AdvancingClock, *DeterministicClock) {
	systemTime := NewDeterministicClock(time.UnixMilli(1000))
	clock := &AdvancingClock{
		DeterministicClock: NewDeterministicClock(time.UnixMilli(5000)),
		systemTime:         systemTime,
		advanceEvery:       advanceEvery,
		quit:               make(chan interface{}),
		lastTick:           systemTime.Now(),
	}
	return clock, systemTime
}
