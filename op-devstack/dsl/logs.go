package dsl

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/logpipe"
	"github.com/ethereum-optimism/optimism/op-service/testreq"
)

// LogRecorder collects the log entries a node's process emits at or above a level, from the
// moment it was created until it is stopped.
type LogRecorder struct {
	entries  []logpipe.LogEntry
	minLevel slog.Level
	mu       sync.Mutex
	name     string
	require  *testreq.Assertions
	stop     func()
}

func newLogRecorder(name string, minLevel slog.Level, require *testreq.Assertions) *LogRecorder {
	return &LogRecorder{minLevel: minLevel, name: name, require: require}
}

func (r *LogRecorder) observe(e logpipe.LogEntry) {
	if e.LogLevel() < r.minLevel {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
}

// Entries returns the entries recorded so far.
func (r *LogRecorder) Entries() []logpipe.LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]logpipe.LogEntry(nil), r.entries...)
}

// Stop ends the recording. The entries recorded so far stay available.
func (r *LogRecorder) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.stop != nil {
		r.stop()
		r.stop = nil
	}
}

// VerifyEmpty fails the test, quoting every recorded entry, if the node logged anything at or
// above the recorder's level since the recording started. activity says what the node was doing.
func (r *LogRecorder) VerifyEmpty(activity string) {
	entries := r.Entries()
	if len(entries) == 0 {
		return
	}
	lines := make([]string, 0, len(entries))
	for _, e := range entries {
		lines = append(lines, fmt.Sprintf("%s %s %v", e.LogLevel(), e.LogMessage(), e.LogFields()))
	}
	r.require.FailNowf("unexpected logs", "%s logged %d entries at or above %s while %s:\n%s",
		r.name, len(entries), r.minLevel, activity, strings.Join(lines, "\n"))
}
