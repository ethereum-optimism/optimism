package logfilter

import (
	"context"
	"log/slog"

	"github.com/ethereum-optimism/optimism/op-service/locks"
	"github.com/ethereum-optimism/optimism/op-service/logmods"
)

// Handler provides access to log filtering customization.
// In the handler stack, through embedding, this can be available in the logger.Handler().
type Handler interface {
	slog.Handler
	Set(fn LogFilter)
	Add(fn LogFilter)
}

type filterHandler struct {
	inner slog.Handler

	all *locks.RWValue[LogFilter]
}

var _ logmods.Handler = (*filterHandler)(nil)
var _ Handler = (*filterHandler)(nil)
var _ slog.Handler = (*filterHandler)(nil)

func WrapFilterHandler(h slog.Handler) slog.Handler {
	return &filterHandler{inner: h, all: &locks.RWValue[LogFilter]{Value: Noop()}}
}

func (f *filterHandler) Unwrap() slog.Handler {
	return f.inner
}

// Enabled runs before the slog logger call is turned into a record.
// This thus can filter logging without allocating.
func (f *filterHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// pre-process / adjust level
	level = f.all.Get()(ctx, level)
	return f.inner.Enabled(ctx, level)
}

// Handle receives the slog logger call
func (f *filterHandler) Handle(ctx context.Context, r slog.Record) error {
	// We can add additional more expensive filters here.
	// This runs once the
	return f.inner.Handle(ctx, r)
}

func (f *filterHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &filterHandler{
		inner: f.inner.WithAttrs(attrs),
		all:   f.all,
	}
}

func (f *filterHandler) WithGroup(name string) slog.Handler {
	return &filterHandler{
		inner: f.inner.WithGroup(name),
		all:   f.all,
	}
}

// Set sets the logging filter. No other filters will apply.
func (f *filterHandler) Set(fn LogFilter) {
	f.all.Set(fn)
}

// Add adds the logging filter. The original filter will apply first (if any).
func (f *filterHandler) Add(fn LogFilter) {
	f.all.Lock()
	defer f.all.Unlock()
	if f.all.Value == nil {
		f.all.Value = fn
	} else {
		f.all.Value = f.all.Value.Add(fn)
	}
}
