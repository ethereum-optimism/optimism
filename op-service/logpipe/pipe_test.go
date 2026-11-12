package logpipe

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestWriteToLogProcessor(t *testing.T) {
	logger, capt := testlog.CaptureLogger(t, log.LevelTrace)

	proc := NewLineBuffer(func(line []byte) {
		ToLogger(logger)(ParseRustStructuredLogs(line))
	})
	_, err := io.Copy(proc, strings.NewReader(`{"level": "DEBUG", "fields": {"message": "hello", "foo": 1}}`+"\n"))
	require.NoError(t, err)
	_, err = io.Copy(proc, strings.NewReader(`test invalid JSON`+"\n"))
	require.NoError(t, err)
	_, err = io.Copy(proc, strings.NewReader(`{"fields": {"message": "world", "bar": "sunny"}, "level": "INFO"}`+"\n"))
	require.NoError(t, err)

	entry1 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelDebug),
		testlog.NewAttributesContainsFilter("foo", "1"))
	require.NotNil(t, entry1)
	require.Equal(t, "hello", entry1.Message)

	entry2 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewAttributesContainsFilter("line", "test invalid JSON"))
	require.NotNil(t, entry2)
	require.Equal(t, "Invalid JSON", entry2.Message)

	entry3 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelInfo),
		testlog.NewAttributesContainsFilter("bar", "sunny"))
	require.NotNil(t, entry3)
	require.Equal(t, "world", entry3.Message)
}

func TestWriteToLogProcessorWithMinLevel(t *testing.T) {
	logger, capt := testlog.CaptureLogger(t, log.LevelTrace)

	proc := NewLineBuffer(func(line []byte) {
		ToLoggerWithMinLevel(logger, log.LevelWarn)(ParseRustStructuredLogs(line))
	})
	_, err := io.Copy(proc, strings.NewReader(`{"level": "DEBUG", "fields": {"message": "hello", "foo": 1}}`+"\n"))
	require.NoError(t, err)
	_, err = io.Copy(proc, strings.NewReader(`{"fields": {"message": "world", "bar": "sunny"}, "level": "INFO"}`+"\n"))
	require.NoError(t, err)
	_, err = io.Copy(proc, strings.NewReader(`{"fields": {"message": "warn", "baz": "kept"}, "level": "WARN"}`+"\n"))
	require.NoError(t, err)

	require.Nil(t, capt.FindLog(
		testlog.NewLevelFilter(log.LevelDebug),
		testlog.NewAttributesContainsFilter("foo", "1")))
	require.Nil(t, capt.FindLog(
		testlog.NewLevelFilter(log.LevelInfo),
		testlog.NewAttributesContainsFilter("bar", "sunny")))

	entry := capt.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewAttributesContainsFilter("baz", "kept"))
	require.NotNil(t, entry)
	require.Equal(t, "warn", entry.Message)
}

func TestWriteToLogProcessorRaisedToLevel(t *testing.T) {
	logger, capt := testlog.CaptureLogger(t, log.LevelTrace)

	proc := NewLineBuffer(func(line []byte) {
		ToLoggerRaisedToLevel(logger, log.LevelInfo)(ParseRustStructuredLogs(line))
	})
	_, err := io.Copy(proc, strings.NewReader(`{"level": "DEBUG", "fields": {"message": "hello", "foo": 1}}`+"\n"))
	require.NoError(t, err)
	_, err = io.Copy(proc, strings.NewReader(`{"fields": {"message": "world", "bar": "sunny"}, "level": "WARN"}`+"\n"))
	require.NoError(t, err)

	// Below-min entries are raised to the min level, keeping the original level as attribute.
	entry1 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelInfo),
		testlog.NewAttributesContainsFilter("innerLevel", "DEBUG"))
	require.NotNil(t, entry1)
	require.Equal(t, "hello", entry1.Message)

	// Entries at or above the min level are passed through unchanged.
	entry2 := capt.FindLog(
		testlog.NewLevelFilter(log.LevelWarn),
		testlog.NewAttributesContainsFilter("bar", "sunny"))
	require.NotNil(t, entry2)
	require.Equal(t, "world", entry2.Message)
}

func TestParseRustStructuredLogsMarksUnstructuredLines(t *testing.T) {
	for _, tc := range []struct {
		name             string
		line             string
		wantUnstructured bool
	}{
		{name: "plain text", line: "thread 'main' panicked at src/main.rs:2:5:", wantUnstructured: true},
		{name: "structured entry", line: `{"level":"INFO","fields":{"message":"hello"}}`, wantUnstructured: false},
		{name: "structured entry with the unstructured message", line: `{"level":"INFO","fields":{"message":"Invalid JSON"}}`, wantUnstructured: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := ParseRustStructuredLogs([]byte(tc.line)).(StructuredRustLogEntry)
			require.Equal(t, tc.wantUnstructured, entry.Unstructured)
		})
	}
}

func TestParseRustStderr(t *testing.T) {
	for _, tc := range []struct {
		name      string
		line      string
		wantLevel slog.Level
	}{
		{name: "thread panic", line: "thread 'main' panicked at src/main.rs:2:5:", wantLevel: log.LevelError},
		{name: "panic without a thread prefix", line: "task panicked at crates/node/src/lib.rs:10:1", wantLevel: log.LevelError},
		{name: "thread stack overflow", line: "thread 'tokio-runtime-worker' has overflowed its stack", wantLevel: log.LevelError},
		{name: "runtime abort", line: "fatal runtime error: stack overflow", wantLevel: log.LevelError},
		{name: "error returned from main", line: "Error: failed to open the database", wantLevel: log.LevelError},
		{name: "panic note", line: "note: run with `RUST_BACKTRACE=1` environment variable to display a backtrace", wantLevel: log.LevelWarn},
		{name: "plain eprintln", line: "warning: --foo is deprecated", wantLevel: log.LevelWarn},
		{name: "error word mid-line", line: "retrying after Error: timeout", wantLevel: log.LevelWarn},
		{name: "structured warning", line: `{"level":"WARN","fields":{"message":"panicked at"}}`, wantLevel: log.LevelWarn},
		{name: "structured error", line: `{"level":"ERROR","fields":{"message":"x"}}`, wantLevel: log.LevelError},
		{name: "structured entry with the unstructured message", line: `{"level":"INFO","fields":{"message":"Invalid JSON","line":"Error: x"}}`, wantLevel: log.LevelInfo},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.wantLevel, ParseRustStderr([]byte(tc.line)).LogLevel())
		})
	}
}
