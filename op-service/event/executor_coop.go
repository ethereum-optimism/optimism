package event

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
)

// CooperativeExec mirrors GlobalSyncExec behavior for enqueue/drain semantics,
// while paving the way for cooperative async/await scheduling.
type CooperativeExec struct {
	// event queue and notifier
	eventsLock sync.Mutex
	events     prioritizedEvents // protected by eventsLock

	// queued is closed and replaced whenever a new item is enqueued.
	// It is nil when no reader is awaiting.
	// Protected by eventsLock.
	queued chan struct{}

	// registered executables, sorted by descending priority
	handles     []*coopHandle
	handlesLock sync.RWMutex

	// cooperative run token: at most one task holds it when running
	runToken chan struct{}

	// event waiters (awaiters), one-shot per waiter
	waitersLock  sync.Mutex
	eventWaiters []eventWaiter

	// lightweight worker pool for handler execution
	jobs           chan func()
	workers        int
	maxWorkers     int
	growLock       sync.Mutex
	currentWorkers int

	ctx context.Context

	metrics Metrics
}

var _ Executor = (*CooperativeExec)(nil)
var _ AwaitYield = (*CooperativeExec)(nil)

const defaultWorkerCount = 4
const defaultJobQueueSize = 1024
const defaultMaxWorkers = 64

func NewCooperative(ctx context.Context) *CooperativeExec {
	var byPriority [priorityCount]*eventsList
	for i := range byPriority {
		byPriority[i] = &eventsList{make([]AnnotatedEvent, 0, 100)}
	}
	c := &CooperativeExec{
		ctx: ctx,
		events: prioritizedEvents{
			byPriority: byPriority,
			count:      0,
		},
		queued:     nil,
		runToken:   make(chan struct{}, 1),
		metrics:    &NoopMetrics{},
		jobs:       make(chan func(), defaultJobQueueSize),
		workers:    defaultWorkerCount,
		maxWorkers: defaultMaxWorkers,
	}
	// initialize run token available
	c.runToken <- struct{}{}
	c.ensureWorkers(c.workers)
	return c
}

// WithWorkers sets the base worker count (must be >0). Returns self for chaining.
func (c *CooperativeExec) WithWorkers(n int) *CooperativeExec {
	if n <= 0 {
		n = 1
	}
	c.workers = n
	c.ensureWorkers(n)
	return c
}

// WithMaxWorkers sets the maximum number of workers the pool may grow to.
func (c *CooperativeExec) WithMaxWorkers(n int) *CooperativeExec {
	if n <= 0 {
		n = 1
	}
	c.maxWorkers = n
	return c
}

func (c *CooperativeExec) ensureWorkers(target int) {
	c.growLock.Lock()
	defer c.growLock.Unlock()
	for c.currentWorkers < target {
		c.currentWorkers++
		go func() {
			for {
				select {
				case <-c.ctx.Done():
					return
				case job := <-c.jobs:
					if job != nil {
						job()
					}
				}
			}
		}()
	}
}

func (c *CooperativeExec) maybeGrowPool() {
	// Grow if backlog exceeds available workers, up to maxWorkers.
	pending := len(c.jobs)
	c.growLock.Lock()
	cur := c.currentWorkers
	max := c.maxWorkers
	c.growLock.Unlock()
	if pending >= cur && cur < max {
		c.ensureWorkers(cur + 1)
	}
}

func (c *CooperativeExec) submit(job func()) {
	// Try to enqueue; if queue is full, grow and block until enqueued.
	select {
	case c.jobs <- job:
		c.maybeGrowPool()
		return
	default:
		c.maybeGrowPool()
		c.jobs <- job
	}
}

func (c *CooperativeExec) WithMetrics(m Metrics) *CooperativeExec {
	c.metrics = m
	return c
}

func (c *CooperativeExec) Add(d Executable, cfg *ExecutorConfig) (leaveExecutor func()) {
	c.handlesLock.Lock()
	defer c.handlesLock.Unlock()
	h := &coopHandle{d: d, priority: cfg.Priority}
	h.g.Store(c)
	c.handles = append(c.handles, h)
	// sort by descending priority
	sort.Slice(c.handles, func(i, j int) bool {
		return c.handles[i].priority > c.handles[j].priority
	})
	return h.leave
}

func (c *CooperativeExec) remove(h *coopHandle) {
	c.handlesLock.Lock()
	defer c.handlesLock.Unlock()
	for i, v := range c.handles {
		if v == h {
			c.handles = slices.Delete(c.handles, i, i+1)
			return
		}
	}
}

func (c *CooperativeExec) Enqueue(ev AnnotatedEvent) error {
	c.eventsLock.Lock()
	defer c.eventsLock.Unlock()
	// sanity limit, never queue too many events
	count := c.events.Count()
	c.metrics.SetTotalEnqueuedEvents(count)
	if count >= sanityEventLimit {
		return fmt.Errorf("something is very wrong, queued up too many events! Dropping event %q", ev.Event)
	}
	// Wake all matching event waiters before queuing so tasks can become pending
	c.notifyEventWaiters(ev)
	c.events.Add(ev)
	if c.queued != nil {
		close(c.queued)
		c.queued = nil
	}
	return nil
}

func (c *CooperativeExec) processEvent(ev AnnotatedEvent) {
	// copy handles to avoid holding the lock while dispatching
	c.handlesLock.RLock()
	handles := make([]*coopHandle, len(c.handles))
	copy(handles, c.handles)
	c.handlesLock.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(handles))
	for _, h := range handles {
		hh := h
		e := ev
		c.submit(func() {
			c.acquireRun()
			hh.onEvent(e)
			c.releaseRun()
			wg.Done()
		})
	}
	// Wait for all handlers to complete to preserve Drain semantics
	wg.Wait()
	if ev.PostProcessCallback != nil {
		ev.PostProcessCallback()
	}
}

// Await returns a channel that is closed if and when event(s) have been queued up.
// This may be used to await when Drain() can be called for event processing.
func (c *CooperativeExec) Await() <-chan struct{} {
	c.eventsLock.Lock()
	defer c.eventsLock.Unlock()
	if c.queued == nil {
		out := make(chan struct{})
		if c.events.Peek().Event != nil {
			close(out)
			return out
		}
		c.queued = out
	}
	return c.queued
}

func (c *CooperativeExec) Drain() error {
	for {
		if c.ctx.Err() != nil {
			return c.ctx.Err()
		}
		c.eventsLock.Lock()
		ev := c.events.Pop()
		c.eventsLock.Unlock()
		if ev.Event == nil {
			return nil
		}
		c.processEvent(ev)
	}
}

func (c *CooperativeExec) DrainUntil(fn func(ev Event) bool, excl bool) error {
	iter := func() (ev AnnotatedEvent, stopIncl bool, stopExcl bool) {
		c.eventsLock.Lock()
		defer c.eventsLock.Unlock()

		ev = c.events.Peek()
		if ev.Event == nil {
			return AnnotatedEvent{}, false, false
		}
		stop := fn(ev.Event)
		if excl && stop {
			ev = AnnotatedEvent{}
			stopExcl = true
		} else {
			popped := c.events.Pop()
			if !ev.Equals(popped) {
				panic("expected popped event to match")
			}
		}
		if stop {
			stopIncl = true
		}
		return
	}

	for {
		if c.ctx.Err() != nil {
			return c.ctx.Err()
		}
		ev, stopIncl, stopExcl := iter()
		if stopExcl {
			return nil
		}
		if ev.Event == nil {
			return io.EOF
		}
		c.processEvent(ev)
		if stopIncl {
			return nil
		}
	}
}

type coopHandle struct {
	g        atomic.Pointer[CooperativeExec]
	d        Executable
	priority Priority
}

func (h *coopHandle) onEvent(ev AnnotatedEvent) {
	if h.g.Load() == nil {
		return
	}
	h.d.RunEvent(ev)
}

func (h *coopHandle) leave() {
	if old := h.g.Swap(nil); old != nil {
		old.remove(h)
	}
}

// cooperative control helpers
func (c *CooperativeExec) acquireRun() { <-c.runToken }
func (c *CooperativeExec) releaseRun() {
	select {
	case c.runToken <- struct{}{}:
	default:
	}
}

// OnAwaitStart implements AwaitYield by releasing the cooperative run token
// and returning a function to re-acquire it when awaiting completes.
func (c *CooperativeExec) OnAwaitStart() func() {
	c.releaseRun()
	return c.acquireRun
}

type eventWaiter struct {
	sel Selector
	ch  chan AnnotatedEvent
}

func (c *CooperativeExec) registerEventWaiter(sel Selector, ch chan AnnotatedEvent) {
	c.waitersLock.Lock()
	defer c.waitersLock.Unlock()
	c.eventWaiters = append(c.eventWaiters, eventWaiter{sel: sel, ch: ch})
}

func (c *CooperativeExec) unregisterEventWaiter(ch chan AnnotatedEvent) {
	c.waitersLock.Lock()
	defer c.waitersLock.Unlock()
	for i := 0; i < len(c.eventWaiters); i++ {
		if c.eventWaiters[i].ch == ch {
			c.eventWaiters = slices.Delete(c.eventWaiters, i, i+1)
			i--
		}
	}
}

func (c *CooperativeExec) notifyEventWaiters(ev AnnotatedEvent) {
	c.waitersLock.Lock()
	defer c.waitersLock.Unlock()
	if len(c.eventWaiters) == 0 {
		return
	}
	// collect remaining waiters after notifying matches
	remaining := c.eventWaiters[:0]
	for _, w := range c.eventWaiters {
		if w.sel != nil && w.sel.Matches(ev.Event) {
			select {
			case w.ch <- ev:
				// one-shot; do not keep
			default:
				// if receiver already ready but channel full, drop to avoid blocking
			}
			continue
		}
		remaining = append(remaining, w)
	}
	c.eventWaiters = remaining
}
