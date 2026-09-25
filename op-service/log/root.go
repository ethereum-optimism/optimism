package log

import (
	"log/slog"
	"os"
	"sync"
)

var (
	rootMu sync.RWMutex
	root   = NewLogger(slog.DiscardHandler)
)

// Root returns the global default logger. It discards every record until
// [SetDefault] installs another logger.
func Root() Logger {
	rootMu.RLock()
	defer rootMu.RUnlock()
	return root
}

// SetDefault installs l as the global default logger. Prefer
// logcli.SetGlobalLogHandler, which also tags the logger as global.
//
// l also becomes go-ethereum's global logger, and, if it was returned by
// [NewLogger], the log/slog default logger.
func SetDefault(l Logger) {
	rootMu.Lock()
	defer rootMu.Unlock()
	root = l
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
//
// Each function calls emit itself when the global logger was returned by
// [NewLogger], so that the record is attributed to the function's caller.
// Records of any other global Logger carry whatever source it assigns.

// Trace logs a message at LevelTrace through the global default logger.
func Trace(msg string, args ...any) {
	if l, ok := Root().(*logger); ok {
		l.emit(l.ctx, 0, LevelTrace, msg, args)
	} else {
		Root().Trace(msg, args...)
	}
}

// Debug logs a message at LevelDebug through the global default logger.
func Debug(msg string, args ...any) {
	if l, ok := Root().(*logger); ok {
		l.emit(l.ctx, 0, LevelDebug, msg, args)
	} else {
		Root().Debug(msg, args...)
	}
}

// Info logs a message at LevelInfo through the global default logger.
func Info(msg string, args ...any) {
	if l, ok := Root().(*logger); ok {
		l.emit(l.ctx, 0, LevelInfo, msg, args)
	} else {
		Root().Info(msg, args...)
	}
}

// Warn logs a message at LevelWarn through the global default logger.
func Warn(msg string, args ...any) {
	if l, ok := Root().(*logger); ok {
		l.emit(l.ctx, 0, LevelWarn, msg, args)
	} else {
		Root().Warn(msg, args...)
	}
}

// Error logs a message at LevelError through the global default logger.
func Error(msg string, args ...any) {
	if l, ok := Root().(*logger); ok {
		l.emit(l.ctx, 0, LevelError, msg, args)
	} else {
		Root().Error(msg, args...)
	}
}

// Crit logs a message at LevelCrit through the global default logger and then
// calls os.Exit(1).
func Crit(msg string, args ...any) {
	if l, ok := Root().(*logger); ok {
		l.emit(l.ctx, 0, LevelCrit, msg, args)
	} else {
		Root().Log(LevelCrit, msg, args...)
	}
	os.Exit(1)
}
