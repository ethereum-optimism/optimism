package devtest

import (
	"context"
	"log/slog"

	"github.com/ethereum-optimism/optimism/devnet-sdk/devstack/telemetry"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
)

// This whole file is a hack around the fact that log.Logger doesn't support context passing.

type Logger interface {
	log.Logger
	WithContext(context.Context) Logger
}

func NewLogger(ctx context.Context, t testlog.Testing, level slog.Level) Logger {
	l := testlog.LoggerWithHandlerMod(t, level, func(h slog.Handler) slog.Handler {
		return &tracingHandler{ctx, h}
	})
	return &logger{
		Logger: l,
		level:  level,
		t:      t,
		ctx:    ctx,
	}
}

type logger struct {
	log.Logger

	level slog.Level
	t     testlog.Testing
	ctx   context.Context
}

func (l *logger) WithContext(ctx context.Context) Logger {
	// We need to reset the handler to take the new context into account
	return NewLogger(ctx, l.t, l.level)
}

var _ Logger = (*logger)(nil)

// a fake Logger, just so P can be a CommonT
type pkgLogger struct {
	log.Logger
}

func (l *pkgLogger) WithContext(ctx context.Context) Logger {
	return l
}

var _ Logger = (*pkgLogger)(nil)

type tracingHandler struct {
	ctx context.Context
	slog.Handler
}

func (h *tracingHandler) Handle(_ context.Context, record slog.Record) error {
	return telemetry.WrapHandler(h.Handler).Handle(h.ctx, record)
}
