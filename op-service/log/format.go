package log

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// FormatType defines a type of log format.
// Supported formats: 'text', 'terminal', 'logfmt', 'json'
type FormatType string

const (
	FormatText     FormatType = "text"
	FormatTerminal FormatType = "terminal"
	FormatLogFmt   FormatType = "logfmt"
	FormatLogFmtMs FormatType = "logfmtms"
	FormatJSON     FormatType = "json"
	FormatJSONMs   FormatType = "jsonms"
)

// All supported format types in a slice for iteration
var formatTypes = []FormatType{
	FormatText,
	FormatTerminal,
	FormatLogFmt,
	FormatLogFmtMs,
	FormatJSON,
	FormatJSONMs,
}

// SupportedFormatsString returns a comma-delimited string of supported formats,
func SupportedFormatsString() string {
	names := make([]string, 0, len(formatTypes))
	for _, f := range formatTypes {
		names = append(names, f.String())
	}
	return strings.Join(names, ", ")
}

// FormatHandler returns the correct slog handler factory for the provided format.
func FormatHandler(ft FormatType, color bool) func(io.Writer) slog.Handler {
	termColorHandler := func(w io.Writer) slog.Handler {
		return NewTerminalHandler(w, color)
	}
	logfmtHandler := func(w io.Writer) slog.Handler { return LogfmtHandlerWithLevel(w, LevelTrace) }
	logfmtMsHandler := func(w io.Writer) slog.Handler { return LogfmtMsHandlerWithLevel(w, LevelTrace) }
	switch ft {
	case FormatJSON:
		return JSONHandler
	case FormatJSONMs:
		return JSONMsHandler
	case FormatText:
		if color {
			return termColorHandler
		} else {
			return logfmtHandler
		}
	case FormatTerminal:
		return termColorHandler
	case FormatLogFmt:
		return logfmtHandler
	case FormatLogFmtMs:
		return logfmtMsHandler
	default:
		panic(fmt.Errorf("failed to create slog.Handler factory for format-type=%q and color=%v", ft, color))
	}
}

func (ft FormatType) String() string {
	return string(ft)
}
