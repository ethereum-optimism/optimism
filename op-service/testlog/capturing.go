package testlog

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
)

// CapturedAttributes forms a chain of inherited attributes, to traverse on captured log records.
type CapturedAttributes struct {
	Parent     *CapturedAttributes
	Attributes []slog.Attr
}

// Attrs calls f on each Attr in the [CapturedAttributes].
// Iteration stops if f returns false.
func (r *CapturedAttributes) Attrs(f func(slog.Attr) bool) {
	for _, a := range r.Attributes {
		if !f(a) {
			return
		}
	}
	if r.Parent != nil {
		r.Parent.Attrs(f)
	}
}

// CapturedRecord is a wrapped around a regular log-record,
// to preserve the inherited attributes context, without mutating the record or reordering attributes.
type CapturedRecord struct {
	Parent *CapturedAttributes
	*slog.Record
}

// Attrs calls f on each Attr in the [CapturedRecord].
// Iteration stops if f returns false.
func (r *CapturedRecord) Attrs(f func(slog.Attr) bool) {
	searching := true
	r.Record.Attrs(func(a slog.Attr) bool {
		searching = f(a)
		return searching
	})
	if !searching { // if we found it already, then don't traverse the remainder
		return
	}
	if r.Parent != nil {
		r.Parent.Attrs(f)
	}
}

// CapturingHandler provides a log handler that captures all log records and optionally forwards them to a delegate.
// It is safe for concurrent use, including through handlers derived from it with WithAttrs or WithGroup.
type CapturingHandler struct {
	handler slog.Handler
	// records is shared among derived CapturingHandlers.
	records *recordStore
	// attrs are inherited log record attributes, from a logger that this CapturingHandler may be derived from
	attrs *CapturedAttributes
}

var _ logmods.Handler = (*CapturingHandler)(nil)

// recordStore holds the captured records together with the lock that guards them.
type recordStore struct {
	mu   sync.Mutex
	logs []*CapturedRecord
}

func (s *recordStore) add(r *CapturedRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, r)
}

func (s *recordStore) clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = s.logs[:0] // reuse slice
}

// snapshot returns a copy of the captured records, so that callers can filter them without holding the lock.
func (s *recordStore) snapshot() []*CapturedRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.logs)
}

func WrapCaptureLogger(h slog.Handler) slog.Handler {
	return &CapturingHandler{handler: h, records: new(recordStore)}
}

func CaptureLogger(t Testing, level slog.Level) (_ log.Logger, ch *CapturingHandler) {
	logger := LoggerWithHandlerMod(t, level, WrapCaptureLogger)
	out, ok := logmods.FindHandler[*CapturingHandler](logger.Handler())
	if !ok {
		panic("failed to get attached log-capturing handler")
	}
	return logger, out
}

func (c *CapturingHandler) Unwrap() slog.Handler {
	return c.handler
}

func (c *CapturingHandler) Handle(ctx context.Context, r slog.Record) error {
	c.records.add(&CapturedRecord{
		Parent: c.attrs,
		Record: &r,
	})
	return c.handler.Handle(ctx, r)
}

func (c *CapturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CapturingHandler{
		handler: c.handler.WithAttrs(attrs),
		records: c.records,
		attrs: &CapturedAttributes{
			Parent:     c.attrs,
			Attributes: attrs,
		},
	}
}

func (c *CapturingHandler) WithGroup(name string) slog.Handler {
	return &CapturingHandler{
		handler: c.handler.WithGroup(name),
		records: c.records,
	}
}

func (c *CapturingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return c.handler.Enabled(ctx, level)
}

func (c *CapturingHandler) Clear() {
	c.records.clear()
}

func NewLevelFilter(level slog.Level) LogFilter {
	return func(r *CapturedRecord) bool {
		return r.Record.Level == level
	}
}

func NewAttributesFilter(key, value string) LogFilter {
	return func(r *CapturedRecord) bool {
		found := false
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == key && a.Value.String() == value {
				found = true
				return false
			}
			return true // try next
		})
		return found
	}
}

func NewAttributesContainsFilter(key, value string) LogFilter {
	return func(r *CapturedRecord) bool {
		found := false
		r.Attrs(func(a slog.Attr) bool {
			if a.Key == key && strings.Contains(a.Value.String(), value) {
				found = true
				return false
			}
			return true // try next
		})
		return found
	}
}

func NewMessageFilter(message string) LogFilter {
	return func(r *CapturedRecord) bool {
		return r.Record.Message == message
	}
}

func NewMessageContainsFilter(message string) LogFilter {
	return func(r *CapturedRecord) bool {
		return strings.Contains(r.Record.Message, message)
	}
}

func NewErrContainsFilter(errMessage string) LogFilter {
	return func(r *CapturedRecord) bool {
		found := false
		r.Attrs(func(a slog.Attr) bool {
			if a.Key != "err" {
				return true
			}
			if err, ok := a.Value.Any().(error); ok && strings.Contains(err.Error(), errMessage) {
				found = true
				return false
			}
			return true
		})
		return found
	}
}

type LogFilter func(record *CapturedRecord) bool

func (c *CapturingHandler) FindLog(filters ...LogFilter) *CapturedRecord {
	for _, record := range c.records.snapshot() {
		match := true
		for _, filter := range filters {
			if !filter(record) {
				match = false
				break
			}
		}
		if match {
			return record
		}
	}
	return nil
}

func (c *CapturingHandler) FindLogs(filters ...LogFilter) []*CapturedRecord {
	var logs []*CapturedRecord
	for _, record := range c.records.snapshot() {
		match := true
		for _, filter := range filters {
			if !filter(record) {
				match = false
				break
			}
		}
		if match {
			logs = append(logs, record)
		}
	}
	return logs
}

func (c *CapturingHandler) RequireMessageContained(t require.TestingT, message string, filters ...LogFilter) {
	filters = append(filters, NewMessageContainsFilter(message))
	require.NotNil(t, c.FindLog(filters...), "expected message %s in logs", message)
}

func (c *CapturingHandler) RequireMessageContainedOnce(t require.TestingT, message string, filters ...LogFilter) {
	filters = append(filters, NewMessageContainsFilter(message))
	require.Len(t, c.FindLogs(filters...), 1, "expected message %s exactly once in logs", message)
}

func (c *CapturingHandler) RequireMessageContainedNTimes(t require.TestingT, message string, n int, filters ...LogFilter) {
	filters = append(filters, NewMessageContainsFilter(message))
	require.Len(t, c.FindLogs(filters...), n, "expected message %s %d times in logs", message, n)
}

func (r *CapturedRecord) AttrValue(name string) (v any) {
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == name {
			v = a.Value.Any()
			return false
		}
		return true // try next
	})
	return
}

var _ slog.Handler = (*CapturingHandler)(nil)

type Capturer interface {
	slog.Handler
	Clear()
	FindLog(filters ...LogFilter) *CapturedRecord
	FindLogs(filters ...LogFilter) []*CapturedRecord
}

var _ Capturer = (*CapturingHandler)(nil)
