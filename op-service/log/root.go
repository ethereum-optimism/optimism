package log

import (
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
)

var (
	// rootMu serializes SetDefault, so that the global loggers it sets always
	// agree. Root reads root without locking.
	rootMu      sync.Mutex
	root        atomic.Pointer[Logger]
	discardRoot = NewLogger(slog.DiscardHandler)
)

// Root returns the global default logger. It discards every record until
// [SetDefault] installs another logger.
func Root() Logger {
	if l := root.Load(); l != nil {
		return *l
	}
	return discardRoot
}

// SetDefault installs l as the global default logger. Prefer
// logcli.SetGlobalLogHandler, which also tags the logger as global.
//
// l also becomes go-ethereum's global logger, and, if it was returned by
// [NewLogger], the log/slog default logger.
func SetDefault(l Logger) {
	rootMu.Lock()
	defer rootMu.Unlock()
	root.Store(&l)
	if own, ok := l.(*logger); ok {
		slog.SetDefault(own.inner)
	}
	setGethDefault(l)
}

// New returns a logger derived from the global default logger with the given
// attributes.
func New(args ...any) Logger {
	return Root().With(args...)
}

// The package-level log functions log through the global default logger and
// attribute each record to their caller. Prefer passing a [Logger] explicitly;
// these exist because geth and other libraries log globally and some tooling
// follows suit.

// Trace logs a message at LevelTrace through the global default logger.
func Trace(msg string, args ...any) { logRoot(LevelTrace, msg, args) }

// Debug logs a message at LevelDebug through the global default logger.
func Debug(msg string, args ...any) { logRoot(LevelDebug, msg, args) }

// Info logs a message at LevelInfo through the global default logger.
func Info(msg string, args ...any) { logRoot(LevelInfo, msg, args) }

// Warn logs a message at LevelWarn through the global default logger.
func Warn(msg string, args ...any) { logRoot(LevelWarn, msg, args) }

// Error logs a message at LevelError through the global default logger.
func Error(msg string, args ...any) { logRoot(LevelError, msg, args) }

// Crit logs a message at LevelCrit through the global default logger and then
// calls os.Exit(1).
func Crit(msg string, args ...any) {
	logRoot(LevelCrit, msg, args)
	os.Exit(1)
}

// logRoot logs through the global default logger on behalf of a package-level
// log function. When that logger was returned by [NewLogger], it calls emit
// itself so the record is attributed to the log function's caller; any other
// Logger assigns its own source.
func logRoot(level slog.Level, msg string, args []any) {
	r := Root()
	if l, ok := r.(*logger); ok {
		l.emit(l.ctx, 1, level, msg, args)
		return
	}
	r.Log(level, msg, args...)
}
