package logpipe

import (
	"strings"
	"sync"
)

// LogRecord keeps the entries of a log stream whose text contains a substring: the message of a
// structured entry, the raw line of an unstructured one. Observing and reading are safe from
// different goroutines, so a test can read a record while the stream is still being written.
type LogRecord struct {
	substring string

	mu      sync.Mutex
	entries []LogEntry
}

func newLogRecord(substring string) *LogRecord {
	return &LogRecord{substring: substring}
}

// Observe keeps e when its text contains the record's substring.
func (r *LogRecord) Observe(e LogEntry) {
	if !strings.Contains(entryText(e), r.substring) {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, e)
}

func entryText(e LogEntry) string {
	if structured, ok := e.(StructuredRustLogEntry); ok {
		if line, unparsed := structured.UnparsedLine(); unparsed {
			return line
		}
	}
	return e.LogMessage()
}

// Entries returns the kept entries in the order they were observed.
func (r *LogRecord) Entries() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]LogEntry(nil), r.entries...)
}

// Count is the number of kept entries.
func (r *LogRecord) Count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.entries)
}

// LogRecords fans one log stream out to every record registered on it. The zero value is ready
// to use.
type LogRecords struct {
	mu      sync.Mutex
	records []*LogRecord
}

// Record registers a new record for substring and returns it. Entries observed before the call
// are not part of it.
func (s *LogRecords) Record(substring string) *LogRecord {
	record := newLogRecord(substring)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
	return record
}

// Observe offers e to every registered record.
func (s *LogRecords) Observe(e LogEntry) {
	s.mu.Lock()
	records := append([]*LogRecord(nil), s.records...)
	s.mu.Unlock()
	for _, record := range records {
		record.Observe(e)
	}
}
