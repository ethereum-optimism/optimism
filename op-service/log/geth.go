package log

import (
	"context"
	"log/slog"

	"github.com/ethereum/go-ethereum/log"
)

// This file is the package's only binding to github.com/ethereum/go-ethereum/log.
// It re-exports the go-ethereum handlers and level helpers, and converts a
// [Logger] into the go-ethereum logger type for go-ethereum APIs that take one.
//
// Every re-export is a type alias, a constant, or a variable bound directly to
// the go-ethereum value, never a wrapper.

// TerminalHandler formats log records for human consumption, optionally with
// ANSI colour. Construct one with [NewTerminalHandler] or
// [NewTerminalHandlerWithLevel].
type TerminalHandler = log.TerminalHandler

// GlogHandler wraps another handler with glog-style, per-file verbosity rules.
// Construct one with [NewGlogHandler].
type GlogHandler = log.GlogHandler

// TerminalStringer is implemented by types that want a distinct rendering in
// terminal-formatted output.
type TerminalStringer = log.TerminalStringer

// Log levels. LevelTrace and LevelCrit extend the levels defined by log/slog;
// the rest are the slog levels under their conventional names.
const (
	LevelTrace = log.LevelTrace
	LevelDebug = log.LevelDebug
	LevelInfo  = log.LevelInfo
	LevelWarn  = log.LevelWarn
	LevelError = log.LevelError
	LevelCrit  = log.LevelCrit
)

// Legacy spellings of the log levels, identical in value to the Level*
// constants above. Prefer the Level* names in new code.
const (
	LvlTrace = log.LvlTrace
	LvlDebug = log.LvlDebug
	LvlInfo  = log.LvlInfo
)

// Handler constructors.
var (
	// DiscardHandler returns a handler that drops every record.
	DiscardHandler = log.DiscardHandler

	NewTerminalHandler          = log.NewTerminalHandler
	NewTerminalHandlerWithLevel = log.NewTerminalHandlerWithLevel

	JSONHandler            = log.JSONHandler
	JSONHandlerWithLevel   = log.JSONHandlerWithLevel
	LogfmtHandler          = log.LogfmtHandler
	LogfmtHandlerWithLevel = log.LogfmtHandlerWithLevel

	NewGlogHandler = log.NewGlogHandler
)

// Level formatting and conversion. See also [LevelFromString] for the inverse
// of LevelString.
var (
	// LevelString renders a level as a lowercase name, e.g. "info".
	LevelString = log.LevelString

	// LevelAlignedString renders a level as a 5-character padded name, e.g. "INFO ".
	LevelAlignedString = log.LevelAlignedString

	// FromLegacyLevel converts a pre-slog geth verbosity number to a level.
	FromLegacyLevel = log.FromLegacyLevel
)

// ToGeth returns l as a go-ethereum logger, for go-ethereum APIs that take one.
// The result forwards every call to l, so it shares l's handler, attributes and
// default context. Loggers derived from it with With or New stay go-ethereum
// loggers.
//
// Records are attributed as l attributes them: for a Logger returned by
// [NewLogger], to the go-ethereum code that logged them.
func ToGeth(l Logger) log.Logger {
	return gethForwarder{l}
}

// setGethDefault makes l the go-ethereum global logger, so that go-ethereum code
// logging through its package-level functions reaches the same handler.
func setGethDefault(l Logger) {
	log.SetDefault(ToGeth(l))
}

// gethForwarder presents a Logger as a go-ethereum logger. The embedded Logger
// provides the logging methods, which the compiler promotes through generated
// wrappers that runtime.Callers does not count, so the embedded Logger's source
// attribution is unaffected. gethForwarder adds the methods whose go-ethereum
// signature or semantics differ.
type gethForwarder struct {
	Logger
}

var _ log.Logger = gethForwarder{}

func (f gethForwarder) With(args ...any) log.Logger { return gethForwarder{f.Logger.With(args...)} }
func (f gethForwarder) New(args ...any) log.Logger  { return gethForwarder{f.Logger.New(args...)} }

// Write logs a message at the given level, attributing the record to the caller
// of the function that calls Write, as go-ethereum's package-level log
// functions expect: they call Write on go-ethereum's global logger.
func (f gethForwarder) Write(level slog.Level, msg string, args ...any) {
	if l, ok := f.Logger.(*logger); ok {
		l.emit(l.ctx, 1, level, msg, args)
	} else {
		f.Logger.Log(level, msg, args...)
	}
}

// WriteCtx is Write with an explicit context. It exists to satisfy op-geth's
// log.Logger.
func (f gethForwarder) WriteCtx(ctx context.Context, level slog.Level, msg string, args ...any) {
	if l, ok := f.Logger.(*logger); ok {
		l.emit(ctx, 1, level, msg, args)
	} else {
		f.Logger.WithContext(ctx).Log(level, msg, args...)
	}
}

// SetContext does nothing. It exists to satisfy op-geth's log.Logger; a
// Logger's context is fixed when it is derived, with WithContext.
func (f gethForwarder) SetContext(context.Context) {}
