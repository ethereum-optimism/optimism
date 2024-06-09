package driver

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
)

// Don't queue up an endless number of events.
// At some point it's better to drop events and warn something is exploding the number of events.
const sanityEventLimit = 1000

// SynchronousEvents is a rollup.EventEmitter that a rollup.Deriver can emit events to.
// The events will be queued up, and can then be executed synchronously by calling the Drain function,
// which will apply all events to the root Deriver that
// New events may be queued up while events are being processed by the root rollup.Deriver.
type SynchronousEvents struct {
	// The lock is no-op in FP execution, if running in synchronous FP-VM.
	// This lock ensures that all emitted events are merged together correctly,
	// if this util is used in a concurrent context.
	evLock sync.Mutex

	events []rollup.Event

	log log.Logger

	ctx context.Context

	root rollup.Deriver
}

func NewSynchronousEvents(log log.Logger, ctx context.Context, root rollup.Deriver) *SynchronousEvents {
	return &SynchronousEvents{
		log:  log,
		ctx:  ctx,
		root: root,
	}
}

func (s *SynchronousEvents) Emit(event rollup.Event) {
	s.evLock.Lock()
	defer s.evLock.Unlock()

	if s.ctx.Err() != nil {
		s.log.Warn("Ignoring emitted event during shutdown", "event", event)
		return
	}

	// sanity limit, never queue too many events
	if len(s.events) >= sanityEventLimit {
		s.log.Error("something is very wrong, queued up too many events! Dropping event", "ev", event)
		return
	}
	s.events = append(s.events, event)
}

func (s *SynchronousEvents) Drain() error {
	for {
		if s.ctx.Err() != nil {
			return s.ctx.Err()
		}
		if len(s.events) == 0 {
			return nil
		}

		s.evLock.Lock()
		first := s.events[0]
		s.events = s.events[1:]
		s.evLock.Unlock()

		s.root.OnEvent(first)
	}
}

var _ rollup.EventEmitter = (*SynchronousEvents)(nil)
