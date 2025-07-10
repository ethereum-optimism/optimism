package event

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
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
	handles     []*gsBus
	handlesLock sync.RWMutex

	ctx context.Context

	// if true, no events may be processed, except CriticalError itself
	abort atomic.Bool

	// used to generate a unique id for each event handler processing call.
	derivID atomic.Uint64
	// used to generate a unique id for each event-emission.
	emitID atomic.Uint64

	metrics Metrics

	tracers *Tracers
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

func (gs *GlobalSyncExec) NewBus(cfg *RegisterConfig) Bus {
	gs.handlesLock.Lock()
	defer gs.handlesLock.Unlock()
	h := newGsBus(gs, cfg)
	gs.handles = append(gs.handles, h)
	// sort by descending executor priority
	sort.Slice(gs.handles, func(i, j int) bool {
		return gs.handles[i].cfg.Executor.Priority > gs.handles[j].cfg.Executor.Priority
	})
	return h.bus
}

func (gs *GlobalSyncExec) remove(h *gsBus) {
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

	if gs.abort.Load() && !Is[CriticalErrorEvent](ev.Event) {
		// if aborting, and not the CriticalErrorEvent itself, then do not process the event
		return
	}

	for _, h := range gs.handles {
		h.onEvent(ev)
	}
	if taskID := TaskIDFromCtx(ev.Ctx); taskID != UndefinedTask {
		// TODO: unregister task handler
	}
	gs.tracers.OnAfterProcessed(ev.Event.String())
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

type gsBus struct {
	name    string
	g       atomic.Pointer[GlobalSyncExec]
	bus     *basicBus
	cfg     *RegisterConfig
	limiter *rate.Limiter
}

func newGsBus(gs *GlobalSyncExec, cfg *RegisterConfig) *gsBus {
	g := &gsBus{name: cfg.Name}
	if cfg.Emitter.Limiting {
		g.limiter = rate.NewLimiter(cfg.Emitter.Rate, cfg.Emitter.Burst)
	}
	b := &basicBus{
		tasks:   make(map[HandlerKey]*Handler),
		watches: make(map[HandlerKey][]*Handler),
		closer:  nil,
	}
	h := &gsBus{bus: b, cfg: cfg}
	h.g.Store(gs)
	b.closer = h.leave
	b.emitter = EmitterFunc(h.emit)
	return g
}

func (gh *gsBus) emit(ctx context.Context, ev Event) error {
	g := gh.g.Load()
	if g == nil {
		return errors.New("bus was closed")
	}

	emitID := g.emitID.Add(1)

	// Make the main system aware of a critical error
	// as soon as anything emits a critical event.
	if Is[CriticalErrorEvent](ev) {
		g.abort.Store(true)
	}

	if ctx == nil {
		return fmt.Errorf("emitter %s must provide a context with the emitted event %s", gh.name, ev.String())
	}
	if e := ctx.Err(); e != nil {
		return e
	}
	if gh.limiter != nil {
		if err := gh.limiter.Wait(ctx); err != nil {
			return err
		}
	}
	return g.Enqueue(AnnotatedEvent{
		Ctx:          ctx,
		Event:        ev,
		EmitContext:  emitID,
		EmitPriority: gh.cfg.Emitter.Priority,
	})
}

func (gh *gsBus) onEvent(ev AnnotatedEvent) {
	g := gh.g.Load()
	if g == nil { // don't process more events while we are being removed
		return
	}
	derivCtx := g.derivID.Add(1)
	startTime := time.Now()
	g.tracers.OnDeriveStart(gh.name, ev, derivCtx, startTime)
	gh.bus.processEvent(ev.Ctx, ev.Event)
	duration := time.Since(startTime)
	g.tracers.OnDeriveEnd(gh.name, ev, derivCtx, startTime, duration, true)
}

func (gh *gsBus) leave() {
	if old := gh.g.Swap(nil); old != nil {
		old.remove(gh)
	}
	return
}
