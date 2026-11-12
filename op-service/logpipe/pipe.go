package logpipe

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"

	"github.com/ethereum-optimism/optimism/op-service/log"
)

type rawRustJSONLog struct {
	//"timestamp" ignored
	Level  string         `json:"level"`
	Fields map[string]any `json:"fields"`
	Target string         `json:"target"`
}

type StructuredRustLogEntry struct {
	Message string
	Level   slog.Level
	Fields  map[string]any
	Target  string

	// Unstructured is set when the line was not a structured JSON log entry. The entry then holds
	// the raw line under the "line" field.
	Unstructured bool
}

func ParseRustStructuredLogs(line []byte) LogEntry {
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.UseNumber() // to preserve number formatting
	var e rawRustJSONLog
	if err := dec.Decode(&e); err != nil {
		return StructuredRustLogEntry{
			Message:      "Invalid JSON",
			Level:        slog.LevelWarn,
			Fields:       map[string]any{"line": string(line)},
			Unstructured: true,
		}
	}
	lvl, err := log.LevelFromString(e.Level)
	if err != nil {
		lvl = log.LevelInfo
	}
	msg, _ := e.Fields["message"].(string)
	delete(e.Fields, "message")

	return StructuredRustLogEntry{
		Message: msg,
		Level:   lvl,
		Fields:  e.Fields,
		Target:  e.Target,
	}
}

// ParseRustStderr parses a line a Rust process wrote to stderr. An unstructured line that reports
// a panic, an abort or an error returned from main is raised to ERROR; every other entry keeps
// its level.
func ParseRustStderr(line []byte) LogEntry {
	e := ParseRustStructuredLogs(line)
	r, ok := e.(StructuredRustLogEntry)
	if !ok || !r.Unstructured || !isRustFailureLine(string(line)) {
		return e
	}
	r.Level = log.LevelError
	return r
}

// isRustFailureLine reports whether line is what the Rust runtime writes to stderr when a thread
// panics or the process aborts, or what main prints when it returns an error.
func isRustFailureLine(line string) bool {
	if strings.Contains(line, "panicked at") {
		return true
	}
	return slices.ContainsFunc([]string{"Error:", "fatal runtime error", "thread '"}, func(prefix string) bool {
		return strings.HasPrefix(line, prefix)
	})
}

func (e StructuredRustLogEntry) LogLevel() slog.Level {
	return e.Level
}

func (e StructuredRustLogEntry) LogMessage() string {
	return e.Message
}

func (e StructuredRustLogEntry) LogFields() []any {
	attrs := make([]any, 0, len(e.Fields)+1)
	if e.Target != "" {
		attrs = append(attrs, slog.String("target", e.Target))
	}
	for k, v := range e.Fields {
		if x, ok := v.(json.Number); ok {
			v = x.String()
		}
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

func (e StructuredRustLogEntry) FieldValue(key string) any {
	return e.Fields[key]
}

type LogEntry interface {
	LogLevel() slog.Level
	LogMessage() string
	LogFields() []any
	FieldValue(key string) any
}

type LogParser func(line []byte) LogEntry

func ToLogger(logger log.Logger) func(e LogEntry) {
	return ToLoggerWithMinLevel(logger, log.LevelTrace)
}

func ToLoggerWithMinLevel(logger log.Logger, minLevel slog.Level) func(e LogEntry) {
	return func(e LogEntry) {
		msg := e.LogMessage()
		attrs := e.LogFields()
		lvl := e.LogLevel()
		if lvl < minLevel {
			return
		}

		if lvl >= log.LevelCrit {
			// If a sub-process has a critical error, this process can handle it
			// Don't force an os.Exit, downgrade to error instead
			lvl = log.LevelError
			attrs = append(attrs, slog.String("innerLevel", "CRIT"))
		}
		logger.Log(lvl, msg, attrs...)
	}
}

// ToLoggerRaisedToLevel emits every entry at minLevel or above: entries below
// minLevel are raised to minLevel, carrying the original level in an
// "innerLevel" attribute, so they survive downstream min-level log filters.
func ToLoggerRaisedToLevel(logger log.Logger, minLevel slog.Level) func(e LogEntry) {
	emit := ToLoggerWithMinLevel(logger, minLevel)
	return func(e LogEntry) {
		lvl := e.LogLevel()
		if lvl >= minLevel {
			emit(e)
			return
		}
		attrs := append(e.LogFields(), slog.String("innerLevel", strings.ToUpper(log.LevelString(lvl))))
		logger.Log(minLevel, e.LogMessage(), attrs...)
	}
}
