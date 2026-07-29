package log

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"
)

func TestDynamicLogHandler_SetLogLevel(t *testing.T) {
	h := new(testRecorder)
	d := NewDynamicLogHandler(log.LevelInfo, h)
	logger := log.NewLogger(d)
	logger.Info("hello world") // y
	logger.Error("error!")     // y
	logger.Debug("debugging")  // n

	// increase log level
	d.SetLogLevel(log.LevelDebug)

	logger.Info("hello again")        // y
	logger.Debug("can see debug now") // y
	logger.Trace("but no trace")      // n

	// and decrease log level
	d.SetLogLevel(log.LevelWarn)
	logger.Warn("visible warning")           // y
	logger.Info("info should be hidden now") // n
	logger.Error("another error")            // y

	require.Len(t, h.records, 2+2+2)
	require.Equal(t, h.records[0].Message, "hello world")
	require.Equal(t, h.records[1].Message, "error!")
	require.Equal(t, h.records[2].Message, "hello again")
	require.Equal(t, h.records[3].Message, "can see debug now")
	require.Equal(t, h.records[4].Message, "visible warning")
	require.Equal(t, h.records[5].Message, "another error")
}

func TestDynamicLogHandler_WithAttrs(t *testing.T) {
	h := new(testRecorder)
	d := NewDynamicLogHandler(log.LevelInfo, h)
	logger := log.NewLogger(d)
	logwith := logger.With("a", 1) // derived logger

	// increase log level
	d.SetLogLevel(log.LevelDebug)

	logwith.Info("info0")   // y
	logwith.Debug("debug0") // y
	logwith.Trace("trace0") // n

	// and decrease log level
	d.SetLogLevel(log.LevelWarn)

	logwith.Info("info1")   // n
	logwith.Warn("warn1")   // y
	logwith.Error("error1") // y

	require.Len(t, h.records, 2+2)
	require.Equal(t, h.records[0].Message, "info0")
	require.Equal(t, h.records[1].Message, "debug0")
	require.Equal(t, h.records[2].Message, "warn1")
	require.Equal(t, h.records[3].Message, "error1")
}

type testRecorder struct {
	records []slog.Record
}

func (r testRecorder) Enabled(context.Context, slog.Level) bool {
	return true
}

func (r *testRecorder) Handle(_ context.Context, rec slog.Record) error {
	r.records = append(r.records, rec)
	return nil
}

func (r *testRecorder) WithAttrs([]slog.Attr) slog.Handler { return r }
func (r *testRecorder) WithGroup(string) slog.Handler      { return r }
