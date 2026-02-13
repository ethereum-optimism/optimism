package logfilter

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/ethereum-optimism/optimism/op-service/logmods"
)

type contextKey struct {
	name string
}

type ctxKeyIndex struct{}

var contextLogAttrIndexCtxKey = ctxKeyIndex{}

// LogKeyIndexFromContext reads a list of (name, key) pairs from the context
// where the keys can be used to read other context values which must implement
// slog.LogValuer and are used by contextHandler to add attributes to log records.
func LogKeyIndexFromContext(ctx context.Context) []string {
	v := ctx.Value(contextLogAttrIndexCtxKey)
	if v == nil {
		return nil
	}
	return v.([]string)
}

// WithValue adds a key-value pair to the context while preventing key collision
// with any other package's string keys on the context.
func WithValue(ctx context.Context, key string, value any) context.Context {
	return context.WithValue(ctx, contextKey{key}, value)
}

// AddLogAttrToContext configures the context so that if a contextHandler
// is in the log handler chain it will add an attr with key `name` and
// value of `ctx.Value(key).(slog.LogValuer).LogValue()` to all log records.
func AddLogAttrToContext(ctx context.Context, name string, value any) context.Context {
	var logValuer slog.LogValuer
	if v, ok := value.(slog.LogValuer); ok {
		logValuer = v
	} else if v, ok := value.(slog.Value); ok {
		logValuer = ValueValuer(v)
	} else {
		logValuer = ValueValuer(slog.AnyValue(value))
	}
	prevIndex := LogKeyIndexFromContext(ctx)
	ctx = WithValue(ctx, name, logValuer)
	// prevIndex is possibly nil, but this should not break the append() call.
	// Independently, we need to force copy the prevIndex slice to avoid mutating the slice stored in the parent context.
	// Filter out any previous contextLogAttrs with the same name to prevent duplicates.
	var filteredIndex []string
	for _, prevKey := range prevIndex {
		if prevKey != name {
			filteredIndex = append(filteredIndex, prevKey)
		}
	}
	nextIndex := append(filteredIndex, name)
	return context.WithValue(ctx, contextLogAttrIndexCtxKey, nextIndex)
}

func ValueFromContext[T slog.LogValuer](ctx context.Context, name string) (T, bool) {
	v, ok := ctx.Value(contextKey{name}).(T)
	return v, ok
}

var _ logmods.Handler = (*contextHandler)(nil)

func WrapContextHandler(h slog.Handler) slog.Handler {
	return &contextHandler{
		handler: h,
	}
}

// Currently used by op-devstack/devtest to unify logging and contextual config.
type contextHandler struct {
	handler slog.Handler
}

func (c *contextHandler) Unwrap() slog.Handler {
	return c.handler
}

func (c *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{handler: c.handler.WithAttrs(attrs)}
}

func (c *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{handler: c.handler.WithGroup(name)}
}

func (c *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return c.handler.Enabled(ctx, level)
}

func (c *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	index := LogKeyIndexFromContext(ctx)
	for _, name := range index {
		if logValuer, ok := ValueFromContext[slog.LogValuer](ctx, name); ok {
			attr := slog.Attr{
				Key:   name,
				Value: logValuer.LogValue(),
			}
			record.Add(attr)
		} else {
			// Log the error to stderr to make the problem visible instead of silently failing
			fmt.Fprintf(os.Stderr, "ERROR: invalid value %v in context for key of type %T, expected value to implement slog.LogValuer\n", logValuer, name)
			// Still continue processing other attributes to avoid complete logging failure
			continue
		}
	}
	return c.handler.Handle(ctx, record)
}
