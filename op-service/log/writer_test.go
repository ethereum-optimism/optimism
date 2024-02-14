package log_test

import (
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"

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
		require.NotNil(t, logs.FindLog(log.LevelInfo, "Test line"))
	})

	t.Run("LogMultipleLines", func(t *testing.T) {
		writer, logs := setup(t, log.LevelInfo)
		line := []byte("Line 1\nLine 2\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LevelInfo, "Line 1"))
		require.NotNil(t, logs.FindLog(log.LevelInfo, "Line 2"))
	})

	t.Run("LogLineAcrossMultipleCalls", func(t *testing.T) {
		writer, logs := setup(t, log.LevelInfo)
		line := []byte("First line\nSplit ")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LevelInfo, "First line"))

		line = []byte("Line\nLast Line\n")
		count, err = writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LevelInfo, "Split Line"))
		require.NotNil(t, logs.FindLog(log.LevelInfo, "Last Line"))
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
			require.NotNil(t, logs.FindLog(lvl, "Log line"))
		})
	}

	t.Run("UseErrorForUnknownLevels", func(t *testing.T) {
		logger, logs := testlog.CaptureLogger(t, log.LevelInfo)
		writer := NewWriter(logger, 60027)
		line := []byte("Log line\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LevelError, "Unknown log level. Using Error"))
		require.NotNil(t, logs.FindLog(log.LevelError, "Log line"))
	})
}
