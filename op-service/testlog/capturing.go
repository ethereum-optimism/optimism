package testlog

import (
	"context"
	"strings"

	"golang.org/x/exp/slog"

	"github.com/ethereum/go-ethereum/log"
)

// CapturingHandler provides a log handler that captures all log records and optionally forwards them to a delegate.
// Note that it is not thread safe.
type CapturingHandler struct {
	handler slog.Handler
	Logs    *[]*slog.Record // shared among derived CapturingHandlers
}

func CaptureLogger(t Testing, level slog.Level) (_ log.Logger, ch *CapturingHandler) {
	return LoggerWithHandlerMod(t, level, func(h slog.Handler) slog.Handler {
		ch = &CapturingHandler{handler: h, Logs: new([]*slog.Record)}
		return ch
	}), ch
}

func (c *CapturingHandler) Enabled(context.Context, slog.Level) bool {
	// We want to capture all logs, even if the underlying handler only logs
	// above a certain level.
	return true
}

func (c *CapturingHandler) Handle(ctx context.Context, r slog.Record) error {
	*c.Logs = append(*c.Logs, &r)
	if c.handler != nil && c.handler.Enabled(ctx, r.Level) {
		return c.handler.Handle(ctx, r)
	}
	return nil
}

func (c *CapturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Note: additional attributes won't be visible for captured logs
	return &CapturingHandler{
		handler: c.handler.WithAttrs(attrs),
		Logs:    c.Logs,
	}
}

func (c *CapturingHandler) WithGroup(name string) slog.Handler {
	return &CapturingHandler{
		handler: c.handler.WithGroup(name),
		Logs:    c.Logs,
	}
}

func (c *CapturingHandler) Clear() {
	*c.Logs = (*c.Logs)[:0] // reuse slice
}

func (c *CapturingHandler) FindLog(lvl slog.Level, msg string) *HelperRecord {
	for _, record := range *c.Logs {
		if record.Level == lvl && record.Message == msg {
			return &HelperRecord{record}
		}
	}
	return nil
}

func (c *CapturingHandler) FindLogContaining(msg string) (*HelperRecord, bool) {
	for _, record := range *c.Logs {
		if strings.Contains(record.Message, msg) {
			return &HelperRecord{record}, true
		}
	}
	return nil, false
}

type HelperRecord struct {
	*slog.Record
}

func (h HelperRecord) AttrValue(name string) (v any) {
	h.Attrs(func(a slog.Attr) bool {
		if a.Key == name {
			v = a.Value.Any()
			return false
		}
		return true // try next
	})
	return
}

var _ slog.Handler = (*CapturingHandler)(nil)
