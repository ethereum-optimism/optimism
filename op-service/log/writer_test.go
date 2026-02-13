package log_test

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	. "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

var _ io.Writer = (*Writer)(nil)

func TestLogWriter(t *testing.T) {
	setup := func(t *testing.T, lvl slog.Level) (*Writer, *testlog.CapturingHandler) {
		logger, logs := testlog.CaptureLogger(t, lvl)
		writer := NewWriter(logger, lvl)
		return writer, logs
	}

	t.Run("LogSingleLine", func(t *testing.T) {
		writer, logs := setup(t, log.LevelInfo)
		line := []byte("Test line\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		levelFilter := testlog.NewLevelFilter(log.LevelInfo)
		msgFilter := testlog.NewMessageFilter("Test line")
		require.NotNil(t, logs.FindLog(levelFilter, msgFilter))
	})

	t.Run("LogMultipleLines", func(t *testing.T) {
		writer, logs := setup(t, log.LevelInfo)
		line := []byte("Line 1\nLine 2\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		levelFilter := testlog.NewLevelFilter(log.LevelInfo)
		lineOneFilter := testlog.NewMessageFilter("Line 1")
		lineTwoFilter := testlog.NewMessageFilter("Line 2")
		require.NotNil(t, logs.FindLog(levelFilter, lineOneFilter))
		require.NotNil(t, logs.FindLog(levelFilter, lineTwoFilter))
	})

	t.Run("LogLineAcrossMultipleCalls", func(t *testing.T) {
		writer, logs := setup(t, log.LevelInfo)
		line := []byte("First line\nSplit ")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		levelFilter := testlog.NewLevelFilter(log.LevelInfo)
		msgFilter := testlog.NewMessageFilter("First line")
		require.NotNil(t, logs.FindLog(levelFilter, msgFilter))

		line = []byte("Line\nLast Line\n")
		count, err = writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		levelFilter = testlog.NewLevelFilter(log.LevelInfo)
		splitLineFilter := testlog.NewMessageFilter("Split Line")
		lastLineFilter := testlog.NewMessageFilter("Last Line")
		require.NotNil(t, logs.FindLog(levelFilter, splitLineFilter))
		require.NotNil(t, logs.FindLog(levelFilter, lastLineFilter))
	})

	// Can't test LevelCrit or it will call os.Exit
	for _, lvl := range []slog.Level{log.LevelTrace, log.LevelDebug, log.LevelInfo, log.LevelWarn, log.LevelError} {
		lvl := lvl
		t.Run("LogLevel_"+lvl.String(), func(t *testing.T) {
			writer, logs := setup(t, lvl)
			line := []byte("Log line\n")
			count, err := writer.Write(line)
			require.NoError(t, err)
			require.Equal(t, len(line), count)
			levelFilter := testlog.NewLevelFilter(lvl)
			msgFilter := testlog.NewMessageFilter("Log line")
			require.NotNil(t, logs.FindLog(levelFilter, msgFilter))
		})
	}

	t.Run("UseErrorForUnknownLevels", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LevelInfo)
		writer := NewWriter(logger, 60027)
		line := []byte("Log line\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		levelFilter := testlog.NewLevelFilter(log.LevelError)
		unknownFilter := testlog.NewMessageFilter("Unknown log level. Using Error")
		logLineFilter := testlog.NewMessageFilter("Log line")
		require.NotNil(t, logs.FindLog(levelFilter, unknownFilter))
		require.NotNil(t, logs.FindLog(levelFilter, logLineFilter))
	})
}
