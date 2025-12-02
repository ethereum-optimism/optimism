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

// Don't queue up an endless number of events.
// At some point it's better to drop events and warn something is exploding the number of events.
const sanityEventLimit = 10_000

type eventsList struct {
	Events []AnnotatedEvent
}

type prioritizedEvents struct {
	// keyed by priority. May contain empty lists.
	byPriority [priorityCount]*eventsList

	// number of events
	count uint64

	// Note: there is a very limited number of different priorities, that continue to show up over time.
	// And events with equal priority should stay FIFO.
	// So we don't use a priority-queue, but just statically have a few sub-lists, and never remove keys.
}

// Add enqueues the given event.
func (a *prioritizedEvents) Add(event AnnotatedEvent) {
	if !event.EmitPriority.Valid() {
		event.EmitPriority = Normal // if the priority is invalid, try to correct it
	}
	p := a.byPriority[event.EmitPriority-priorityMin]
	p.Events = append(p.Events, event)
	a.count += 1
}

// Pop returns the highest-priority event, and removes it at the same time.
// Returns a zeroed AnnotatedEvent if there is no event to pop.
func (a *prioritizedEvents) Pop() AnnotatedEvent {
	for i := range a.byPriority {
		pe := a.byPriority[priorityCount-1-i] // highest priority first
		if len(pe.Events) > 0 {
			out := pe.Events[0]
			pe.Events = pe.Events[1:]
			a.count -= 1
			return out
		}
	}
	return AnnotatedEvent{}
}

// Peek returns the highest-priority event, without removing it.
// Returns a zeroed AnnotatedEvent if there is no event to peek.
func (a *prioritizedEvents) Peek() AnnotatedEvent {
	for i := range a.byPriority {
		pe := a.byPriority[priorityCount-1-i] // highest priority first
		if len(pe.Events) > 0 {
			return pe.Events[0]
		}
	}
	return AnnotatedEvent{}
}

// Count returns the number of currently queued events
func (a *prioritizedEvents) Count() uint64 {
	return a.count
}

type GlobalSyncExec struct {
	eventsLock sync.Mutex
	events     prioritizedEvents // protected by eventsLock

	// queued is closed and replaced whenever a new item is enqueued.
	// This is used to signal to Await callers when there are events.
	// It is nil when no reader is awaiting.
	// This is protected by eventsLock.
	queued chan struct{}

	// sorted by descending priority
	handles     []*globalHandle
	handlesLock sync.RWMutex

	ctx context.Context

	metrics Metrics
}

var _ Executor = (*GlobalSyncExec)(nil)

func NewGlobalSynchronous(ctx context.Context) *GlobalSyncExec {
	var byPriority [priorityCount]*eventsList
	for i := range byPriority {
		// pre-allocate with some default capacity
		byPriority[i] = &eventsList{make([]AnnotatedEvent, 0, 100)}
	}
	return &GlobalSyncExec{
		ctx: ctx,
		events: prioritizedEvents{
			byPriority: byPriority,
			count:      0,
		},
		queued:  nil,
		metrics: &NoopMetrics{},
	}
}

func (gs *GlobalSyncExec) WithMetrics(m Metrics) *GlobalSyncExec {
	gs.metrics = m
	return gs
}

func (gs *GlobalSyncExec) Add(d Executable, cfg *ExecutorConfig) (leaveExecutor func()) {
	gs.handlesLock.Lock()
	defer gs.handlesLock.Unlock()
	h := &globalHandle{d: d, priority: cfg.Priority}
	h.g.Store(gs)
	gs.handles = append(gs.handles, h)
	// sort by descending priority
	sort.Slice(gs.handles, func(i, j int) bool {
		return gs.handles[i].priority > gs.handles[j].priority
	})
	return h.leave
}

func (gs *GlobalSyncExec) remove(h *globalHandle) {
	gs.handlesLock.Lock()
	defer gs.handlesLock.Unlock()
	// Linear search to delete is fine,
	// since we delete much less frequently than we process events with these.
	for i, v := range gs.handles {
		if v == h {
			gs.handles = slices.Delete(gs.handles, i, i+1)
			return
		}
	}
}

func (gs *GlobalSyncExec) Enqueue(ev AnnotatedEvent) error {
	gs.eventsLock.Lock()
	defer gs.eventsLock.Unlock()
	// sanity limit, never queue too many events
	count := gs.events.Count()
	gs.metrics.SetTotalEnqueuedEvents(count)
	if count >= sanityEventLimit {
		return fmt.Errorf("something is very wrong, queued up too many events! Dropping event %q", ev.Event)
	}
	gs.events.Add(ev)
	if gs.queued != nil {
		close(gs.queued) // To everyone waiting so far: let them know we have an event.
		gs.queued = nil  // To everyone in the future: they will need to Await for a new event again
	}
	return nil
}

func (gs *GlobalSyncExec) processEvent(ev AnnotatedEvent) {
	gs.handlesLock.RLock() // read lock, to allow Drain() to be called during event processing.
	defer gs.handlesLock.RUnlock()
	for _, h := range gs.handles {
		h.onEvent(ev)
	}
	if ev.PostProcessCallback != nil {
		ev.PostProcessCallback()
	}
}

// Await returns a channel that is closed if and when event(s) have been queued up.
// This may be used to await when Drain() can be called for event processing.
func (gs *GlobalSyncExec) Await() <-chan struct{} {
	gs.eventsLock.Lock()
	defer gs.eventsLock.Unlock()
	if gs.queued == nil { // If nobody was awaiting already, initialize.
		out := make(chan struct{})
		// If we already have events, close it immediately.
		if gs.events.Peek().Event != nil {
			close(out)
			// gs.queued is already nil: we want to keep the close signal coupled to the enqueuing of events.
			return out
		}
		gs.queued = out
	}
	return gs.queued
}

func (gs *GlobalSyncExec) Drain() error {
	for {
		if gs.ctx.Err() != nil {
			return gs.ctx.Err()
		}
		gs.eventsLock.Lock()
		ev := gs.events.Pop()
		gs.eventsLock.Unlock()
		if ev.Event == nil {
			return nil
		}
		// Note: event execution may call Drain(), that is allowed.
		gs.processEvent(ev)
	}
}

func (gs *GlobalSyncExec) DrainUntil(fn func(ev Event) bool, excl bool) error {
	// In order of operation:
	// stopExcl: stop draining, and leave the event.
	// no stopExcl, and no event: EOF, exhausted events before condition hit.
	// no stopExcl, and event: process event.
	// stopIncl: stop draining, after having processed the event first.
	iter := func() (ev AnnotatedEvent, stopIncl bool, stopExcl bool) {
		gs.eventsLock.Lock()
		defer gs.eventsLock.Unlock()

		ev = gs.events.Peek()
		if ev.Event == nil {
			return AnnotatedEvent{}, false, false
		}
		stop := fn(ev.Event)
		if excl && stop {
			ev = AnnotatedEvent{}
			stopExcl = true
		} else {
			popped := gs.events.Pop()
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
		if gs.ctx.Err() != nil {
			return gs.ctx.Err()
		}
		// includes popping of the event, so we can handle Drain() calls by onEvent() execution
		ev, stopIncl, stopExcl := iter()
		if stopExcl {
			return nil
		}
		if ev.Event == nil {
			return io.EOF
		}
		gs.processEvent(ev)
		if stopIncl {
			return nil
		}
	}
}

type globalHandle struct {
	g        atomic.Pointer[GlobalSyncExec]
	d        Executable
	priority Priority
}

func (gh *globalHandle) onEvent(ev AnnotatedEvent) {
	if gh.g.Load() == nil { // don't process more events while we are being removed
		return
	}
	gh.d.RunEvent(ev)
}

func (gh *globalHandle) leave() {
	if old := gh.g.Swap(nil); old != nil {
		old.remove(gh)
	}
}
