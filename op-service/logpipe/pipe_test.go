package logpipe

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestWriteToLogProcessor(t *testing.T) {
	logger, capt := testlog.CaptureLogger(t, log.LevelTrace)

	logProc := LogProcessor(func(line []byte) {
		ToLogger(logger)(ParseRustStructuredLogs(line))
	})
	_, err := io.Copy(logProc, strings.NewReader(`{"level": "DEBUG", "fields": {"message": "hello", "foo": 1}}`+"\n"))
	require.NoError(t, err)
	_, err = io.Copy(logProc, strings.NewReader(`test invalid JSON`+"\n"))
	require.NoError(t, err)
	_, err = io.Copy(logProc, strings.NewReader(`{"fields": {"message": "world", "bar": "sunny"}, "level": "INFO"}`+"\n"))
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
