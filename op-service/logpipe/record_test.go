package logpipe

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func rustLogLine(message string) []byte {
	return []byte(`{"timestamp":"2026-01-01T00:00:00Z","level":"DEBUG","fields":{"message":"` + message + `","block":7},"target":"engine::tree"}`)
}

func TestLogRecordKeepsMatchingEntriesInOrder(t *testing.T) {
	record := newLogRecord("reorg")
	record.Observe(ParseRustStructuredLogs(rustLogLine("on-disk reorg detected")))
	record.Observe(ParseRustStructuredLogs(rustLogLine("Block executed")))
	record.Observe(ParseRustStructuredLogs(rustLogLine("Handling reorg: unwinding and buffering new path")))

	require.Equal(t, 2, record.Count())
	entries := record.Entries()
	require.Len(t, entries, 2)
	require.Equal(t, "on-disk reorg detected", entries[0].LogMessage())
	require.Equal(t, "Handling reorg: unwinding and buffering new path", entries[1].LogMessage())
	require.Equal(t, json.Number("7"), entries[0].FieldValue("block"), "entries keep their fields")
}

func TestLogRecordMatchesTheRawTextOfUnstructuredLines(t *testing.T) {
	record := newLogRecord("panicked at")
	record.Observe(ParseRustStructuredLogs([]byte("thread 'main' panicked at src/lib.rs:7:9:")))
	record.Observe(ParseRustStructuredLogs([]byte("some other plain line")))
	record.Observe(ParseRustStructuredLogs(rustLogLine("Invalid JSON")))

	require.Equal(t, 1, record.Count(), "the raw line is matched, not the placeholder message of an unstructured entry")
	line, unparsed := record.Entries()[0].(StructuredRustLogEntry).UnparsedLine()
	require.True(t, unparsed)
	require.Equal(t, "thread 'main' panicked at src/lib.rs:7:9:", line)
}

func TestLogRecordEntriesIsACopy(t *testing.T) {
	record := newLogRecord("")
	record.Observe(ParseRustStructuredLogs(rustLogLine("first")))
	entries := record.Entries()
	record.Observe(ParseRustStructuredLogs(rustLogLine("second")))

	require.Len(t, entries, 1, "entries returned earlier must not grow with the record")
	require.Equal(t, 2, record.Count())
}

func TestLogRecordsFanOutFromRegistration(t *testing.T) {
	var records LogRecords
	records.Observe(ParseRustStructuredLogs(rustLogLine("before registration")))
	reorgs := records.Record("reorg")
	everything := records.Record("")
	records.Observe(ParseRustStructuredLogs(rustLogLine("on-disk reorg detected")))
	records.Observe(ParseRustStructuredLogs(rustLogLine("Block executed")))

	require.Equal(t, 1, reorgs.Count())
	require.Equal(t, 2, everything.Count(), "a record only sees entries observed after it was registered")
}
