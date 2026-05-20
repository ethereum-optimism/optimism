package clock

import (
	"context"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// AdvancingClock tracks wall time and can be jumped forward via AdvanceTime.
// Now() is lazy (wall + offset) so it always reads the current time.
// A single scheduler goroutine fires pending timers when due; AdvanceTime
// signals it so jumped-past timers fire immediately.
type AdvancingClock struct {
	wallClock Clock

	mu sync.Mutex
	// offset is the manual time-travel delta added on top of wallClock.Now().
	// Guarded by mu.
	offset       time.Duration
	pending      []*pendingAction // sorted ascending by due
	wakeup       chan struct{}    // buffer 1; signals the scheduler to re-evaluate
	newPendingCh chan struct{}    // buffer 1; signals WaitForNewPendingAction
	quit         chan struct{}    // closed by Stop
	running      atomic.Bool
	wg           sync.WaitGroup
}

// pendingAction is one scheduled event. advTimer / advTicker wrap it to
// adapt the differing Timer / Ticker Stop signatures.
type pendingAction struct {
	mu       sync.Mutex
	clock    *AdvancingClock
	due      time.Time
	period   time.Duration // zero for one-shot, non-zero for ticker
	ch       chan time.Time
	callback func()
	stopped  bool
	run      bool
}

// NewAdvancingClock creates a clock that advances continuously with the
// system clock and can also be jumped forward by arbitrary amounts via
// AdvanceTime. Now() always reflects the latest wall time plus any
// accumulated manual jumps. Scheduled actions (Timer, Ticker, After,
// AfterFunc) fire when due in logical time; a manual AdvanceTime that
// jumps past a due time fires the action immediately.
func NewAdvancingClock() *AdvancingClock {
	return newAdvancingClock(SystemClock)
}

func newAdvancingClock(wall Clock) *AdvancingClock {
	return &AdvancingClock{
		wallClock:    wall,
		wakeup:       make(chan struct{}, 1),
		newPendingCh: make(chan struct{}, 1),
		quit:         make(chan struct{}),
	}
}

func (c *AdvancingClock) Start() {
	if !c.running.CompareAndSwap(false, true) {
		return
	}
	c.mu.Lock()
	c.quit = make(chan struct{})
	c.mu.Unlock()
	c.wg.Add(1)
	go c.run()
	c.signal()
}

func (c *AdvancingClock) Stop() {
	if !c.running.CompareAndSwap(true, false) {
		return
	}
	c.mu.Lock()
	close(c.quit)
	c.mu.Unlock()
	c.wg.Wait()
}

func (c *AdvancingClock) Now() time.Time {
	c.mu.Lock()
	offset := c.offset
	c.mu.Unlock()
	return c.wallClock.Now().Add(offset)
}

func (c *AdvancingClock) Since(t time.Time) time.Duration {
	return c.Now().Sub(t)
}

func (c *AdvancingClock) Until(t time.Time) time.Duration {
	return t.Sub(c.Now())
}

// AdvanceTime jumps time forward by d. Overdue actions fire when the
// scheduler next iterates (signalled below).
func (c *AdvancingClock) AdvanceTime(d time.Duration) {
	if d == 0 {
		return
	}
	c.mu.Lock()
	c.offset += d
	c.mu.Unlock()
	c.signal()
}

func (c *AdvancingClock) signal() {
	select {
	case c.wakeup <- struct{}{}:
	default:
	}
}

func (c *AdvancingClock) After(d time.Duration) <-chan time.Time {
	ch := make(chan time.Time, 1)
	if d <= 0 {
		ch <- c.Now()
		return ch
	}
	c.addAction(&pendingAction{
		clock: c,
		due:   c.Now().Add(d),
		ch:    ch,
	})
	return ch
}

func (c *AdvancingClock) AfterFunc(d time.Duration, f func()) Timer {
	a := &pendingAction{
		clock:    c,
		due:      c.Now().Add(d),
		callback: f,
	}
	if d <= 0 {
		a.run = true
		go f()
		return &advTimer{a: a}
	}
	c.addAction(a)
	return &advTimer{a: a}
}

func (c *AdvancingClock) NewTicker(d time.Duration) Ticker {
	if d <= 0 {
		panic("Continuously firing tickers are a really bad idea")
	}
	ch := make(chan time.Time, 1)
	a := &pendingAction{
		clock:  c,
		due:    c.Now().Add(d),
		period: d,
		ch:     ch,
	}
	c.addAction(a)
	return &advTicker{a: a}
}

func (c *AdvancingClock) NewTimer(d time.Duration) Timer {
	ch := make(chan time.Time, 1)
	a := &pendingAction{
		clock: c,
		due:   c.Now().Add(d),
		ch:    ch,
	}
	if d <= 0 {
		a.run = true
		select {
		case ch <- c.Now():
		default:
		}
		return &advTimer{a: a}
	}
	c.addAction(a)
	return &advTimer{a: a}
}

func (c *AdvancingClock) SleepCtx(ctx context.Context, d time.Duration) error {
	return sleepCtx(ctx, d, c)
}

func (c *AdvancingClock) addAction(a *pendingAction) {
	c.mu.Lock()
	c.insertLocked(a)
	c.mu.Unlock()
	select {
	case c.newPendingCh <- struct{}{}:
	default:
	}
	c.signal()
}

// WaitForNewPendingAction blocks until a new action is scheduled since the
// last call to this method, or ctx is done. Intended for tests that need to
// synchronise with goroutines that schedule timers asynchronously.
func (c *AdvancingClock) WaitForNewPendingAction(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-c.newPendingCh:
		return true
	}
}

func (c *AdvancingClock) insertLocked(a *pendingAction) {
	idx := sort.Search(len(c.pending), func(i int) bool {
		return c.pending[i].due.After(a.due)
	})
	c.pending = append(c.pending, nil)
	copy(c.pending[idx+1:], c.pending[idx:])
	c.pending[idx] = a
}

func (c *AdvancingClock) removeLocked(a *pendingAction) {
	for i, item := range c.pending {
		if item == a {
			c.pending = append(c.pending[:i], c.pending[i+1:]...)
			return
		}
	}
}

// run sleeps until the next due item (or a wakeup/quit signal), then fires
// anything overdue.
func (c *AdvancingClock) run() {
	defer c.wg.Done()
	for {
		c.mu.Lock()
		quit := c.quit
		var nextDue time.Time
		hasPending := false
		for _, a := range c.pending {
			a.mu.Lock()
			stopped := a.stopped
			due := a.due
			a.mu.Unlock()
			if stopped {
				continue
			}
			nextDue = due
			hasPending = true
			break
		}
		c.mu.Unlock()

		if !hasPending {
			select {
			case <-c.wakeup:
				continue
			case <-quit:
				return
			}
		}

		now := c.Now()
		if !nextDue.After(now) {
			c.fireDue()
			continue
		}
		wallSleep := nextDue.Sub(now)
		timer := c.wallClock.NewTimer(wallSleep)
		select {
		case <-timer.Ch():
		case <-c.wakeup:
			timer.Stop()
		case <-quit:
			timer.Stop()
			return
		}
	}
}

// fireDue fires every action due at or before Now(); tickers re-arm.
func (c *AdvancingClock) fireDue() {
	for {
		now := c.Now()
		c.mu.Lock()
		var due *pendingAction
		for _, a := range c.pending {
			a.mu.Lock()
			stopped := a.stopped
			aDue := a.due
			a.mu.Unlock()
			if stopped {
				continue
			}
			if aDue.After(now) {
				break
			}
			due = a
			break
		}
		if due == nil {
			c.purgeStoppedLocked()
			c.mu.Unlock()
			return
		}
		c.removeLocked(due)
		c.mu.Unlock()

		rearm := due.fire(now)
		if rearm {
			c.mu.Lock()
			c.insertLocked(due)
			c.mu.Unlock()
		}
	}
}

func (c *AdvancingClock) purgeStoppedLocked() {
	kept := c.pending[:0]
	for _, a := range c.pending {
		a.mu.Lock()
		stopped := a.stopped
		a.mu.Unlock()
		if !stopped {
			kept = append(kept, a)
		}
	}
	c.pending = kept
}

// fire delivers one event. Returns true if the action should be re-armed (ticker).
func (a *pendingAction) fire(now time.Time) bool {
	a.mu.Lock()
	if a.stopped {
		a.mu.Unlock()
		return false
	}
	period := a.period
	ch := a.ch
	cb := a.callback
	a.run = true
	if period > 0 {
		a.due = now.Add(period)
	}
	a.mu.Unlock()

	if ch != nil {
		// Non-blocking send: matches stdlib time.Ticker semantics.
		select {
		case ch <- now:
		default:
		}
	}
	if cb != nil {
		// Run on its own goroutine so a slow callback cannot stall the
		// scheduler. Matches stdlib time.AfterFunc semantics.
		go cb()
	}
	return period > 0
}

type advTimer struct {
	a *pendingAction
}

func (t *advTimer) Ch() <-chan time.Time { return t.a.ch }

func (t *advTimer) Stop() bool {
	return t.a.stopExternal()
}

type advTicker struct {
	a *pendingAction
}

func (t *advTicker) Ch() <-chan time.Time { return t.a.ch }

func (t *advTicker) Stop() {
	t.a.stopExternal()
}

func (t *advTicker) Reset(d time.Duration) {
	if d <= 0 {
		panic("Continuously firing tickers are a really bad idea")
	}
	t.a.reset(d)
}

func (a *pendingAction) stopExternal() bool {
	a.mu.Lock()
	wasActive := !a.stopped && !a.run
	a.stopped = true
	a.mu.Unlock()
	a.clock.signal()
	return wasActive
}

func (a *pendingAction) reset(d time.Duration) bool {
	due := a.clock.Now().Add(d)
	a.mu.Lock()
	wasActive := !a.stopped && !a.run
	a.stopped = false
	a.run = false
	a.due = due
	if a.period > 0 {
		a.period = d
	}
	a.mu.Unlock()

	a.clock.mu.Lock()
	a.clock.removeLocked(a)
	a.clock.insertLocked(a)
	a.clock.mu.Unlock()
	a.clock.signal()
	return wasActive
}

var _ Clock = (*AdvancingClock)(nil)
var _ Timer = (*advTimer)(nil)
var _ Ticker = (*advTicker)(nil)
