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

// SingleThreadCooperativeExec mirrors CooperativeExec enqueue/drain semantics
// but processes all handlers sequentially in a single goroutine context.
// It also supports event waiters for tests and utilities to wait on events.
type SingleThreadCooperativeExec struct {
	// event queue and notifier
	eventsLock sync.Mutex
	events     prioritizedEvents // protected by eventsLock

	// queued is closed and replaced whenever a new item is enqueued.
	// It is nil when no reader is awaiting.
	// Protected by eventsLock.
	queued chan struct{}

	// registered executables, sorted by descending priority
	handles     []*singleHandle
	handlesLock sync.RWMutex

	// event waiters (awaiters), one-shot per waiter
	waitersLock  sync.Mutex
	eventWaiters []eventWaiter

	ctx context.Context

	metrics Metrics
}

var _ Executor = (*SingleThreadCooperativeExec)(nil)
var _ AwaitYield = (*SingleThreadCooperativeExec)(nil)

func NewSingleThreadCooperative(ctx context.Context) *SingleThreadCooperativeExec {
	var byPriority [priorityCount]*eventsList
	for i := range byPriority {
		byPriority[i] = &eventsList{make([]AnnotatedEvent, 0, 100)}
	}
	return &SingleThreadCooperativeExec{
		ctx: ctx,
		events: prioritizedEvents{
			byPriority: byPriority,
			count:      0,
		},
		queued:  nil,
		metrics: &NoopMetrics{},
	}
}

func (s *SingleThreadCooperativeExec) WithMetrics(m Metrics) *SingleThreadCooperativeExec {
	s.metrics = m
	return s
}

func (s *SingleThreadCooperativeExec) Add(d Executable, cfg *ExecutorConfig) (leaveExecutor func()) {
	s.handlesLock.Lock()
	defer s.handlesLock.Unlock()
	h := &singleHandle{d: d, priority: cfg.Priority}
	h.g.Store(s)
	s.handles = append(s.handles, h)
	// sort by descending priority
	sort.Slice(s.handles, func(i, j int) bool {
		return s.handles[i].priority > s.handles[j].priority
	})
	return h.leave
}

func (s *SingleThreadCooperativeExec) remove(h *singleHandle) {
	s.handlesLock.Lock()
	defer s.handlesLock.Unlock()
	for i, v := range s.handles {
		if v == h {
			s.handles = slices.Delete(s.handles, i, i+1)
			return
		}
	}
}

func (s *SingleThreadCooperativeExec) Enqueue(ev AnnotatedEvent) error {
	s.eventsLock.Lock()
	defer s.eventsLock.Unlock()
	// sanity limit, never queue too many events
	count := s.events.Count()
	s.metrics.SetTotalEnqueuedEvents(count)
	if count >= sanityEventLimit {
		return fmt.Errorf("something is very wrong, queued up too many events! Dropping event %q", ev.Event)
	}
	// Wake all matching event waiters before queuing so tasks can become pending
	s.notifyEventWaiters(ev)
	s.events.Add(ev)
	if s.queued != nil {
		close(s.queued)
		s.queued = nil
	}
	return nil
}

func (s *SingleThreadCooperativeExec) processEvent(ev AnnotatedEvent) {
	// copy handles to avoid holding the lock while dispatching
	s.handlesLock.RLock()
	handles := make([]*singleHandle, len(s.handles))
	copy(handles, s.handles)
	s.handlesLock.RUnlock()

	for _, h := range handles {
		hh := h
		e := ev
		hh.onEvent(e)
	}
	if ev.PostProcessCallback != nil {
		ev.PostProcessCallback()
	}
}

// Await returns a channel that is closed if and when event(s) have been queued up.
// This may be used to await when Drain() can be called for event processing.
func (s *SingleThreadCooperativeExec) Await() <-chan struct{} {
	s.eventsLock.Lock()
	defer s.eventsLock.Unlock()
	if s.queued == nil {
		out := make(chan struct{})
		if s.events.Peek().Event != nil {
			close(out)
			return out
		}
		s.queued = out
	}
	return s.queued
}

func (s *SingleThreadCooperativeExec) Drain() error {
	for {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}
		s.eventsLock.Lock()
		ev := s.events.Pop()
		s.eventsLock.Unlock()
		if ev.Event == nil {
			return nil
		}
		s.processEvent(ev)
	}
}

func (s *SingleThreadCooperativeExec) DrainUntil(fn func(ev Event) bool, excl bool) error {
	iter := func() (ev AnnotatedEvent, stopIncl bool, stopExcl bool) {
		s.eventsLock.Lock()
		defer s.eventsLock.Unlock()

		ev = s.events.Peek()
		if ev.Event == nil {
			return AnnotatedEvent{}, false, false
		}
		stop := fn(ev.Event)
		if excl && stop {
			ev = AnnotatedEvent{}
			stopExcl = true
		} else {
			popped := s.events.Pop()
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
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}
		ev, stopIncl, stopExcl := iter()
		if stopExcl {
			return nil
		}
		if ev.Event == nil {
			return io.EOF
		}
		s.processEvent(ev)
		if stopIncl {
			return nil
		}
	}
}

type singleHandle struct {
	g        atomic.Pointer[SingleThreadCooperativeExec]
	d        Executable
	priority Priority
}

func (h *singleHandle) onEvent(ev AnnotatedEvent) {
	if h.g.Load() == nil {
		return
	}
	h.d.RunEvent(ev)
}

func (h *singleHandle) leave() {
	if old := h.g.Swap(nil); old != nil {
		old.remove(h)
	}
}

func (s *SingleThreadCooperativeExec) notifyEventWaiters(ev AnnotatedEvent) {
	s.waitersLock.Lock()
	defer s.waitersLock.Unlock()
	if len(s.eventWaiters) == 0 {
		return
	}
	remaining := s.eventWaiters[:0]
	for _, w := range s.eventWaiters {
		if w.sel != nil && w.sel.Matches(ev.Event) {
			select {
			case w.ch <- ev:
			default:
			}
			continue
		}
		remaining = append(remaining, w)
	}
	s.eventWaiters = remaining
}

// OnAwaitStart implements AwaitYield for the single-thread executor.
// There is no token to release; return a no-op to keep a uniform API.
func (s *SingleThreadCooperativeExec) OnAwaitStart() func() { return func() {} }
