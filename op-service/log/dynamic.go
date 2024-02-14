package log

import (
	"context"

	"golang.org/x/exp/slog"
)

type LvlSetter interface {
	SetLogLevel(lvl slog.Level)
}

// DynamicLogHandler allow runtime-configuration of the log handler.
type DynamicLogHandler struct {
	slog.Handler // embedded, to expose any extra methods the underlying handler might provide
	minLvl       slog.Level
}

func NewDynamicLogHandler(lvl slog.Level, h slog.Handler) *DynamicLogHandler {
	return &DynamicLogHandler{
		Handler: h,
		minLvl:  lvl,
	}
}

func (d *DynamicLogHandler) SetLogLevel(lvl slog.Level) {
	d.minLvl = lvl
}

func (d *DynamicLogHandler) Handle(ctx context.Context, r slog.Record) error {
	if r.Level < d.minLvl { // higher log level values are more critical
		return nil
	}
	return d.Handler.Handle(ctx, r) // process the log
}

func (d *DynamicLogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return (lvl >= d.minLvl) && d.Handler.Enabled(ctx, lvl)
}
