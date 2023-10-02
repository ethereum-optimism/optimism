package log

import (
	"io"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

var _ io.Writer = (*Writer)(nil)

func TestLogWriter(t *testing.T) {
	setup := func(t *testing.T, lvl log.Lvl) (*Writer, *testlog.CapturingHandler) {
		logger := testlog.Logger(t, lvl)
		logs := testlog.Capture(logger)
		writer := NewWriter(logger, lvl)
		return writer, logs
	}

	t.Run("LogSingleLine", func(t *testing.T) {
		writer, logs := setup(t, log.LvlInfo)
		line := []byte("Test line\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LvlInfo, "Test line"))
	})

	t.Run("LogMultipleLines", func(t *testing.T) {
		writer, logs := setup(t, log.LvlInfo)
		line := []byte("Line 1\nLine 2\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LvlInfo, "Line 1"))
		require.NotNil(t, logs.FindLog(log.LvlInfo, "Line 2"))
	})

	t.Run("LogLineAcrossMultipleCalls", func(t *testing.T) {
		writer, logs := setup(t, log.LvlInfo)
		line := []byte("First line\nSplit ")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LvlInfo, "First line"))

		line = []byte("Line\nLast Line\n")
		count, err = writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LvlInfo, "Split Line"))
		require.NotNil(t, logs.FindLog(log.LvlInfo, "Last Line"))
	})

	// Can't test LvlCrit or it will call os.Exit
	for _, lvl := range []log.Lvl{log.LvlTrace, log.LvlDebug, log.LvlInfo, log.LvlWarn, log.LvlError} {
		lvl := lvl
		t.Run("LogLvl_"+lvl.String(), func(t *testing.T) {
			writer, logs := setup(t, lvl)
			line := []byte("Log line\n")
			count, err := writer.Write(line)
			require.NoError(t, err)
			require.Equal(t, len(line), count)
			require.NotNil(t, logs.FindLog(lvl, "Log line"))
		})
	}

	t.Run("UseErrorForUnknownLevels", func(t *testing.T) {
		logger := testlog.Logger(t, log.LvlInfo)
		logs := testlog.Capture(logger)
		writer := NewWriter(logger, 60027)
		line := []byte("Log line\n")
		count, err := writer.Write(line)
		require.NoError(t, err)
		require.Equal(t, len(line), count)
		require.NotNil(t, logs.FindLog(log.LvlError, "Unknown log level. Using Error"))
		require.NotNil(t, logs.FindLog(log.LvlError, "Log line"))
	})
}
