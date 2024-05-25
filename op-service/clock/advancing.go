package clock

import (
	"sync/atomic"
	"time"
)

type AdvancingClock struct {
	*DeterministicClock
	systemTime   Clock
	ticker       Ticker
	advanceEvery time.Duration
	quit         chan interface{}
	running      atomic.Bool

	lastTick time.Time
}

// NewAdvancingClock creates a clock that, when started, advances at the same rate as the system clock but
// can also be advanced arbitrary amounts using the AdvanceTime method.
// Unlike the system clock, time does not progress smoothly but only increments when AdvancedTime is called or
// approximately after advanceEvery duration has elapsed. When advancing based on the system clock, the total time
// the system clock has advanced is added to the current time, preventing time differences from building up over time.
func NewAdvancingClock(advanceEvery time.Duration) *AdvancingClock {
	now := SystemClock.Now()
	return &AdvancingClock{
		DeterministicClock: NewDeterministicClock(now),
		systemTime:         SystemClock,
		advanceEvery:       advanceEvery,
		quit:               make(chan interface{}),
		lastTick:           now,
	}
}

func (c *AdvancingClock) Start() {
	if !c.running.CompareAndSwap(false, true) {
		// Already running
		return
	}
	c.ticker = c.systemTime.NewTicker(c.advanceEvery)
	go func() {
		for {
			select {
			case now := <-c.ticker.Ch():
				c.onTick(now)
			case <-c.quit:
				return
			}
		}
	}()
}

func (c *AdvancingClock) Stop() {
	if !c.running.CompareAndSwap(true, false) {
		// Already stopped
		return
	}
	c.quit <- nil
}

func (c *AdvancingClock) onTick(now time.Time) {
	if !now.After(c.lastTick) {
		// Time hasn't progressed for some reason, so do nothing
		return
	}
	// Advance time by however long it has been since the last update.
	// Ensures we don't drift from system time by more and more over time
	advanceBy := now.Sub(c.lastTick)
	c.AdvanceTime(advanceBy)
	c.lastTick = now
}
