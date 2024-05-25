package clock

import (
	"context"
	"sync"
	"time"
)

type action interface {
	// Return true if the action is due to fire
	isDue(time.Time) bool

	// fire triggers the action. Returns true if the action needs to fire again in the future
	fire(time.Time) bool
}

type task struct {
	ch  chan time.Time
	due time.Time
}

func (t task) isDue(now time.Time) bool {
	return !t.due.After(now)
}

func (t task) fire(now time.Time) bool {
	t.ch <- now
	close(t.ch)
	return false
}

type timer struct {
	f       func()
	ch      chan time.Time
	due     time.Time
	stopped bool
	run     bool
	sync.Mutex
}

func (t *timer) isDue(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	return !t.due.After(now)
}

func (t *timer) fire(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	if !t.stopped {
		t.f()
		t.run = true
	}
	return false
}

func (t *timer) Ch() <-chan time.Time {
	return t.ch
}

func (t *timer) Stop() bool {
	t.Lock()
	defer t.Unlock()
	r := !t.stopped && !t.run
	t.stopped = true
	return r
}

type ticker struct {
	c       Clock
	ch      chan time.Time
	nextDue time.Time
	period  time.Duration
	stopped bool
	sync.Mutex
}

func (t *ticker) Ch() <-chan time.Time {
	return t.ch
}

func (t *ticker) Stop() {
	t.Lock()
	defer t.Unlock()
	t.stopped = true
}

func (t *ticker) Reset(d time.Duration) {
	if d <= 0 {
		panic("Continuously firing tickers are a really bad idea")
	}
	t.Lock()
	defer t.Unlock()
	t.period = d
	t.nextDue = t.c.Now().Add(d)
}

func (t *ticker) isDue(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	return !t.nextDue.After(now)
}

func (t *ticker) fire(now time.Time) bool {
	t.Lock()
	defer t.Unlock()
	if t.stopped {
		return false
	}
	// Publish without blocking and only update due time if we publish successfully
	select {
	case t.ch <- now:
		t.nextDue = now.Add(t.period)
	default:
	}
	return true
}

type DeterministicClock struct {
	now          time.Time
	pending      []action
	newPendingCh chan struct{}
	lock         sync.Mutex
}

// NewDeterministicClock creates a new clock where time only advances when the DeterministicClock.AdvanceTime method is called.
// This is intended for use in situations where a deterministic clock is required, such as testing or event driven systems.
func NewDeterministicClock(now time.Time) *DeterministicClock {
	return &DeterministicClock{
		now:          now,
		newPendingCh: make(chan struct{}, 1),
	}
}

func (s *DeterministicClock) Now() time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.now
}

func (s *DeterministicClock) After(d time.Duration) <-chan time.Time {
	s.lock.Lock()
	defer s.lock.Unlock()
	ch := make(chan time.Time, 1)
	if d.Nanoseconds() == 0 {
		ch <- s.now
		close(ch)
	} else {
		s.addPending(&task{ch: ch, due: s.now.Add(d)})
	}
	return ch
}

func (s *DeterministicClock) AfterFunc(d time.Duration, f func()) Timer {
	s.lock.Lock()
	defer s.lock.Unlock()
	timer := &timer{f: f, due: s.now.Add(d)}
	if d.Nanoseconds() == 0 {
		timer.fire(s.now)
	} else {
		s.addPending(timer)
	}
	return timer
}

func (s *DeterministicClock) NewTicker(d time.Duration) Ticker {
	if d <= 0 {
		panic("Continuously firing tickers are a really bad idea")
	}
	s.lock.Lock()
	defer s.lock.Unlock()
	ch := make(chan time.Time, 1)
	t := &ticker{
		c:       s,
		ch:      ch,
		nextDue: s.now.Add(d),
		period:  d,
	}
	s.addPending(t)
	return t
}

func (s *DeterministicClock) NewTimer(d time.Duration) Timer {
	s.lock.Lock()
	defer s.lock.Unlock()
	ch := make(chan time.Time, 1)
	t := &timer{
		f: func() {
			ch <- s.now
		},
		ch:  ch,
		due: s.now.Add(d),
	}
	s.addPending(t)
	return t
}

func (s *DeterministicClock) SleepCtx(ctx context.Context, d time.Duration) error {
	return sleepCtx(ctx, d, s)
}

func (s *DeterministicClock) addPending(t action) {
	s.pending = append(s.pending, t)
	select {
	case s.newPendingCh <- struct{}{}:
	default:
		// Must already have a new pending task flagged, do nothing
	}
}

func (s *DeterministicClock) WaitForNewPendingTaskWithTimeout(timeout time.Duration) bool {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.WaitForNewPendingTask(ctx)
}

// WaitForNewPendingTask blocks until a new task is scheduled since the last time this method was called.
// true is returned if a new task was scheduled, false if the context completed before a new task was added.
func (s *DeterministicClock) WaitForNewPendingTask(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-s.newPendingCh:
		return true
	}
}

// AdvanceTime moves the time forward by the specific duration
func (s *DeterministicClock) AdvanceTime(d time.Duration) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.now = s.now.Add(d)
	var remaining []action
	for _, a := range s.pending {
		if !a.isDue(s.now) || a.fire(s.now) {
			remaining = append(remaining, a)
		}
	}
	s.pending = remaining
}

var _ Clock = (*DeterministicClock)(nil)
