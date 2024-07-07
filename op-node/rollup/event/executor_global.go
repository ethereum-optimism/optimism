package event

import (
	"fmt"
	"io"
	"slices"
	"sync"
	"sync/atomic"
)

// Don't queue up an endless number of events.
// At some point it's better to drop events and warn something is exploding the number of events.
const sanityEventLimit = 1000

type GlobalSyncExec struct {
	eventsLock sync.Mutex
	events     []AnnotatedEvent

	handles     []*globalHandle
	handlesLock sync.RWMutex
}

var _ Executor = (*GlobalSyncExec)(nil)

func NewGlobalSynchronous() *GlobalSyncExec {
	return &GlobalSyncExec{}
}

func (gs *GlobalSyncExec) Add(d Executable, _ *ExecutorOpts) (leaveExecutor func()) {
	gs.handlesLock.Lock()
	defer gs.handlesLock.Unlock()
	h := &globalHandle{d: d}
	h.g.Store(gs)
	gs.handles = append(gs.handles, h)
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

func (s *GlobalSyncExec) Enqueue(ev AnnotatedEvent) error {
	// sanity limit, never queue too many events
	if len(s.events) >= sanityEventLimit {
		return fmt.Errorf("something is very wrong, queued up too many events! Dropping event %q", ev)
	}
	s.events = append(s.events, ev)
	return nil
}

func (s *GlobalSyncExec) pop() AnnotatedEvent {
	if len(s.events) == 0 {
		return AnnotatedEvent{}
	}

	first := s.events[0]
	s.events = s.events[1:]
	return first
}

func (gs *GlobalSyncExec) Drain() error {
	for {
		ev := gs.pop()
		if ev.Event == nil {
			return nil
		}
		for _, h := range gs.handles {
			h.onEvent(ev)
		}
	}
}

func (s *GlobalSyncExec) DrainUntil(fn func(ev Event) bool, excl bool) error {
	for {
		if len(s.events) == 0 {
			return io.EOF
		}

		s.eventsLock.Lock()
		ev := s.events[0]
		stop := fn(ev.Event)
		if excl && stop {
			s.eventsLock.Unlock()
			return nil
		}
		s.events = s.events[1:]
		s.eventsLock.Unlock()

		for _, h := range s.handles {
			h.onEvent(ev)
		}
		if stop {
			return nil
		}
	}
}

type globalHandle struct {
	g atomic.Pointer[GlobalSyncExec]
	d Executable
}

func (ga *globalHandle) onEvent(ev AnnotatedEvent) {
	if ga.g.Load() == nil { // don't process more events while we are being removed
		return
	}
	ga.d.RunEvent(ev)
}

func (ga *globalHandle) leave() {
	if old := ga.g.Swap(nil); old != nil {
		old.remove(ga)
	}
}
