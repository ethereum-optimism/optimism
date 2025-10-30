package logpipe

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

type rawRustJSONLog struct {
	//"timestamp" ignored
	Level  string         `json:"level"`
	Fields map[string]any `json:"fields"`
	//"target" ignored"
}

type StructuredRustLogEntry struct {
	Message string
	Level   slog.Level
	Fields  map[string]any
}

func ParseRustStructuredLogs(line []byte) LogEntry {
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.UseNumber() // to preserve number formatting
	var e rawRustJSONLog
	if err := dec.Decode(&e); err != nil {
		return StructuredRustLogEntry{
			Message: "Invalid JSON",
			Level:   slog.LevelWarn,
			Fields:  map[string]any{"line": string(line)},
		}
	}
	lvl, err := oplog.LevelFromString(e.Level)
	if err != nil {
		lvl = log.LevelInfo
	}
	msg, _ := e.Fields["message"].(string)
	delete(e.Fields, "message")

	return StructuredRustLogEntry{
		Message: msg,
		Level:   lvl,
		Fields:  e.Fields,
	}
}

func (e StructuredRustLogEntry) LogLevel() slog.Level {
	return e.Level
}

func (e StructuredRustLogEntry) LogMessage() string {
	return e.Message
}

func (e StructuredRustLogEntry) LogFields() []any {
	attrs := make([]any, 0, len(e.Fields))
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

type LogProcessor func(line []byte)

func (lo LogProcessor) Write(data []byte) (int, error) {
	startingLength := len(data)
	buf := bytes.NewBuffer(data)
	scanner := bufio.NewScanner(buf)
	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		if len(lineBytes) == 0 {
			continue // Skip empty lines
		}
		lo(lineBytes)
	}
	return startingLength - buf.Len(), scanner.Err()
}

type LogParser func(line []byte) LogEntry

func ToLogger(logger log.Logger) func(e LogEntry) {
	return func(e LogEntry) {
		msg := e.LogMessage()
		attrs := e.LogFields()
		lvl := e.LogLevel()

		if lvl >= log.LevelCrit {
			// If a sub-process has a critical error, this process can handle it
			// Don't force an os.Exit, downgrade to error instead
			lvl = log.LevelError
			attrs = append(attrs, slog.String("innerLevel", "CRIT"))
		}
		logger.Log(lvl, msg, attrs...)
	}
}
