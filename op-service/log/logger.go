// Package log is the monorepo's logging API: a [Logger] interface, its
// slog-backed implementation, and the global default logger. Code logs through
// this package rather than importing github.com/ethereum/go-ethereum/log.
//
// The handlers and level helpers are go-ethereum's, re-exported under the same
// names. [ToGeth] converts a Logger for go-ethereum APIs that take a
// go-ethereum logger.
//
// The package imports no other package of this module, so any package in the
// monorepo can import it.
package log

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"
)

// Logger emits structured log records to an slog.Handler. Record attributes are
// given as alternating keys and values, as slog.Attr values, or as any mix of
// the two. Every record is attributed to the code that called the logging
// method.
//
// A Logger is immutable: With, New and WithContext return a derived Logger and
// leave the receiver unchanged.
//
// Its method set is upstream go-ethereum's log.Logger without Write, with With
// and New returning a Logger, plus WithContext, LogAttrs, TraceContext,
// DebugContext, InfoContext, WarnContext and ErrorContext. Use [ToGeth] to pass
// a Logger to a go-ethereum API.
type Logger interface {
	// With returns a Logger that adds the given attributes to every record.
	With(args ...any) Logger

	// New is identical to With.
	New(args ...any) Logger

	// WithContext returns a Logger that logs every record without an explicit
	// context with ctx. Handlers can use the context to filter records; see
	// slog.Handler.Enabled.
	WithContext(ctx context.Context) Logger

	// Log logs a message at the given level. It does not exit, even at
	// LevelCrit.
	Log(level slog.Level, msg string, args ...any)

	Trace(msg string, args ...any)
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)

	// Crit logs a message at LevelCrit and then calls os.Exit(1).
	Crit(msg string, args ...any)

	// Enabled reports whether the logger emits records at the given context
	// and level.
	Enabled(ctx context.Context, level slog.Level) bool

	// Handler returns the handler that records are written to.
	Handler() slog.Handler

	// LogAttrs logs a message at the given level with attributes only, which
	// avoids parsing key/value pairs.
	LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr)

	TraceContext(ctx context.Context, msg string, args ...any)
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

// logger is the slog-backed Logger returned by NewLogger.
type logger struct {
	inner *slog.Logger
	ctx   context.Context // used for records logged without an explicit context
}

var _ Logger = (*logger)(nil)

// NewLogger returns a Logger that writes to h.
//
// Note that logcli.NewLogger is a different constructor: it builds a logger
// from an io.Writer and a parsed CLIConfig.
func NewLogger(h slog.Handler) Logger {
	return &logger{inner: slog.New(h), ctx: context.Background()}
}

// emit writes a record attributed to the frame skip levels above the caller of
// the function that calls emit: skip 0 is that function's caller.
//
// Every logging method calls emit (or emitAttrs) directly with skip 0, so that
// the stack depth, and hence the attribution, does not depend on the call path.
func (l *logger) emit(ctx context.Context, skip int, level slog.Level, msg string, args []any) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.inner.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(time.Now(), level, msg, callerPC(skip))
	r.Add(args...)
	_ = l.inner.Handler().Handle(ctx, r)
}

func (l *logger) emitAttrs(ctx context.Context, skip int, level slog.Level, msg string, attrs []slog.Attr) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.inner.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(time.Now(), level, msg, callerPC(skip))
	r.AddAttrs(attrs...)
	_ = l.inner.Handler().Handle(ctx, r)
}

// callerPC returns the program counter of the frame skip levels above the
// caller of the function that calls emit. The fixed offset skips
// runtime.Callers, callerPC, emit and that function.
func callerPC(skip int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(4+skip, pcs[:])
	return pcs[0]
}

func (l *logger) With(args ...any) Logger {
	return &logger{inner: l.inner.With(args...), ctx: l.ctx}
}

func (l *logger) New(args ...any) Logger {
	return &logger{inner: l.inner.With(args...), ctx: l.ctx}
}

func (l *logger) WithContext(ctx context.Context) Logger {
	if ctx == nil {
		ctx = context.Background()
	}
	return &logger{inner: l.inner, ctx: ctx}
}

func (l *logger) Log(level slog.Level, msg string, args ...any) { l.emit(l.ctx, 0, level, msg, args) }

func (l *logger) Trace(msg string, args ...any) { l.emit(l.ctx, 0, LevelTrace, msg, args) }
func (l *logger) Debug(msg string, args ...any) { l.emit(l.ctx, 0, LevelDebug, msg, args) }
func (l *logger) Info(msg string, args ...any)  { l.emit(l.ctx, 0, LevelInfo, msg, args) }
func (l *logger) Warn(msg string, args ...any)  { l.emit(l.ctx, 0, LevelWarn, msg, args) }
func (l *logger) Error(msg string, args ...any) { l.emit(l.ctx, 0, LevelError, msg, args) }

func (l *logger) Crit(msg string, args ...any) {
	l.emit(l.ctx, 0, LevelCrit, msg, args)
	os.Exit(1)
}

func (l *logger) Enabled(ctx context.Context, level slog.Level) bool {
	return l.inner.Enabled(ctx, level)
}

func (l *logger) Handler() slog.Handler { return l.inner.Handler() }

func (l *logger) LogAttrs(ctx context.Context, level slog.Level, msg string, attrs ...slog.Attr) {
	l.emitAttrs(ctx, 0, level, msg, attrs)
}

func (l *logger) TraceContext(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, 0, LevelTrace, msg, args)
}

func (l *logger) DebugContext(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, 0, LevelDebug, msg, args)
}

func (l *logger) InfoContext(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, 0, LevelInfo, msg, args)
}

func (l *logger) WarnContext(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, 0, LevelWarn, msg, args)
}

func (l *logger) ErrorContext(ctx context.Context, msg string, args ...any) {
	l.emit(ctx, 0, LevelError, msg, args)
}
