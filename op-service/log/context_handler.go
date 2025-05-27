package log

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/logmods"
)

type contextLogAttr struct {
	name string
	key  any
}

type logKeyIndexCtxType struct{}

var contextLogAttrIndexCtxKey = logKeyIndexCtxType{}

// LogKeyIndexFromContext reads a list of (name, key) pairs from the context
// where the keys can be used to read other context values which must implement
// slog.LogValuer and are used by contextHandler to add attributes to log records.
func LogKeyIndexFromContext(ctx context.Context) []contextLogAttr {
	v := ctx.Value(contextLogAttrIndexCtxKey)
	if v == nil {
		return nil
	}
	return v.([]contextLogAttr)
}

// RegisterLogAttrOnContext configures the context so that if a contextHandler
// is in the log handler chain it will add an attr with key `name` and
// value of `ctx.Value(key).(slog.LogValuer).LogValue()` to all log records.
func RegisterLogAttrOnContext(ctx context.Context, name string, key any) context.Context {
	prevIndex := LogKeyIndexFromContext(ctx)
	// prevIndex is possibly nil, but this should not break the append() call.
	// Independently, we need to force copy the prevIndex slice to avoid mutating the slice stored in the parent context.
	// Filter out any previous contextLogAttrs with the same name to prevent duplicates.
	var filteredIndex []contextLogAttr
	for _, attr := range prevIndex {
		if attr.name != name {
			filteredIndex = append(filteredIndex, attr)
		}
	}
	nextIndex := append(filteredIndex, contextLogAttr{name: name, key: key})
	return context.WithValue(ctx, contextLogAttrIndexCtxKey, nextIndex)
}

var _ logmods.Handler = (*contextHandler)(nil)

func WrapContextHandler(h slog.Handler) slog.Handler {
	return &contextHandler{
		inner: h,
	}
}

// Currently used by op-devstack/devtest to unify logging and contextual config.
type contextHandler struct {
	inner slog.Handler
}

func (c *contextHandler) Unwrap() slog.Handler {
	return c.inner
}

func (c *contextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return c.inner.Enabled(ctx, level)
}

func (c *contextHandler) Handle(ctx context.Context, record slog.Record) error {
	index := LogKeyIndexFromContext(ctx)
	for _, attr := range index {
		name, key := attr.name, attr.key
		value := ctx.Value(key)
		if value == nil {
			continue
		}
		if attr, ok := value.(slog.LogValuer); ok {
			record.Add(name, attr)
		} else {
			return fmt.Errorf("invalid value %v in context for key of type %T, expected value to implement slog.LogValuer", value, key)
		}
	}
	return c.inner.Handle(ctx, record)
}

func (c *contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextHandler{
		inner: c.inner.WithAttrs(attrs),
	}
}

func (c *contextHandler) WithGroup(name string) slog.Handler {
	return &contextHandler{
		inner: c.inner.WithGroup(name),
	}
}
