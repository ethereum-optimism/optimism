package sysgo

import (
	"maps"
	"slices"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/logpipe"
)

// logObservers fans a node's process log entries out to the observers registered through
// stack.LogObservable. It belongs to the node rather than to its SubProcess, so observers stay
// registered across restarts. The zero value is ready to use.
type logObservers struct {
	// mu is separate from the node's own lock because the process's log pipes deliver entries
	// while the node's Start holds that lock.
	mu        sync.Mutex
	nextID    int
	observers map[int]func(logpipe.LogEntry)
}

func (o *logObservers) observe(observer func(logpipe.LogEntry)) (stop func()) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.observers == nil {
		o.observers = make(map[int]func(logpipe.LogEntry))
	}
	id := o.nextID
	o.nextID++
	o.observers[id] = observer
	return func() {
		o.mu.Lock()
		defer o.mu.Unlock()
		delete(o.observers, id)
	}
}

func (o *logObservers) notify(e logpipe.LogEntry) {
	o.mu.Lock()
	observers := slices.Collect(maps.Values(o.observers))
	o.mu.Unlock()
	for _, observe := range observers {
		observe(e)
	}
}
