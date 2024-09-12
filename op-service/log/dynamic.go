package log

import (
	"context"
	"log/slog"
)

type LvlSetter interface {
	SetLogLevel(lvl slog.Level)
}

// DynamicLogHandler allow runtime-configuration of the log handler.
type DynamicLogHandler struct {
	h      slog.Handler
	minLvl *slog.Level // shared with derived dynamic handlers
}

func NewDynamicLogHandler(lvl slog.Level, h slog.Handler) *DynamicLogHandler {
	return &DynamicLogHandler{
		h:      h,
		minLvl: &lvl,
	}
}

func (d *DynamicLogHandler) SetLogLevel(lvl slog.Level) {
	*d.minLvl = lvl
}

func (d *DynamicLogHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < *d.minLvl { // higher log level values are more critical
		return nil
	}
	return d.h.Handle(ctx, r) // process the log
}

func (d *DynamicLogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return (lvl >= *d.minLvl) && d.h.Enabled(ctx, lvl)
}

func (d *DynamicLogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DynamicLogHandler{
		h:      d.h.WithAttrs(attrs),
		minLvl: d.minLvl,
	}
}

func (d *DynamicLogHandler) WithGroup(name string) slog.Handler {
	return &DynamicLogHandler{
		h:      d.h.WithGroup(name),
		minLvl: d.minLvl,
	}
}
