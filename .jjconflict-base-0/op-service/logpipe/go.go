package logpipe

import (
	"bytes"
	"encoding/json"
	"log/slog"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum/go-ethereum/log"
)

type rawGoJSONLog map[string]any

type StructuredGoLogEntry struct {
	Message string
	Level   slog.Level
	Fields  map[string]any
}

func ParseGoStructuredLogs(line []byte) LogEntry {
	dec := json.NewDecoder(bytes.NewReader(line))
	dec.UseNumber() // to preserve number formatting
	var e rawGoJSONLog
	if err := dec.Decode(&e); err != nil {
		return StructuredGoLogEntry{
			Message: "Invalid JSON",
			Level:   slog.LevelWarn,
			Fields:  map[string]any{"line": string(line)},
		}
	}
	lvl, err := oplog.LevelFromString(e["lvl"].(string))
	if err != nil {
		lvl = log.LevelInfo
	}
	msg, _ := e["msg"].(string)
	delete(e, "msg")

	return StructuredGoLogEntry{
		Message: msg,
		Level:   lvl,
		Fields:  e,
	}
}

func (e StructuredGoLogEntry) LogLevel() slog.Level {
	return e.Level
}

func (e StructuredGoLogEntry) LogMessage() string {
	return e.Message
}

func (e StructuredGoLogEntry) LogFields() []any {
	attrs := make([]any, 0, len(e.Fields))
	for k, v := range e.Fields {
		if x, ok := v.(json.Number); ok {
			v = x.String()
		}
		attrs = append(attrs, slog.Any(k, v))
	}
	return attrs
}

func (e StructuredGoLogEntry) FieldValue(key string) any {
	return e.Fields[key]
}
