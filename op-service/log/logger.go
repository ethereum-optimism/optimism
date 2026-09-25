// Package log is the monorepo's logging API. Code logs through these names
// rather than importing github.com/ethereum/go-ethereum/log directly, so that
// the logger it depends on is owned here.
//
// Every go-ethereum name re-exported here is a type alias, a constant, or a
// variable bound directly to the go-ethereum implementation, never a wrapper
// function: the geth logger derives a record's source location with a fixed
// runtime.Callers skip depth, so a wrapper around log.Info would attribute
// every message to this file instead of to its call site.
//
// The package imports no other package of this module, so any package in the
// monorepo can import it.
package log

import (
	"github.com/ethereum/go-ethereum/log"
)

// Logger writes key/value pairs to a handler. Each key/value pair can be two
// consecutive arguments or a single slog.Attr, and both forms may occur in the
// same call.
type Logger = log.Logger

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

// Package-level logging through the global default logger. Prefer passing a
// [Logger] explicitly; these exist because geth and other libraries log
// globally and some tooling follows suit.
//
// Crit logs and then calls os.Exit(1).
var (
	Trace = log.Trace
	Debug = log.Debug
	Info  = log.Info
	Warn  = log.Warn
	Error = log.Error
	Crit  = log.Crit

	// New returns a logger derived from the global default logger with the
	// given attributes.
	New = log.New

	// Root returns the global default logger.
	Root = log.Root

	// SetDefault installs l as the global default logger. Prefer
	// logcli.SetGlobalLogHandler, which also tags the logger as global.
	SetDefault = log.SetDefault
)

// NewLogger returns a Logger that writes to h.
//
// Note that logcli.NewLogger is a different constructor: it builds a logger
// from an io.Writer and a parsed CLIConfig.
var NewLogger = log.NewLogger

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
