package log_test

import (
	"context"
	"log/slog"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

type recordingHandler struct {
	records []slog.Record
}

func (h *recordingHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordingHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *recordingHandler) WithAttrs([]slog.Attr) slog.Handler { return h }

func (h *recordingHandler) WithGroup(string) slog.Handler { return h }

// TestPackageLevelSourceAttribution guards the rule that the package-level log
// functions are bound function values rather than wrappers. The underlying
// logger derives a record's source location from a fixed stack depth, so an
// intermediate wrapper would attribute every message to the logging package
// instead of to the caller.
func TestPackageLevelSourceAttribution(t *testing.T) {
	h := new(recordingHandler)
	prev := oplog.Root()
	t.Cleanup(func() { oplog.SetDefault(prev) })
	oplog.SetDefault(oplog.NewLogger(h))

	oplog.Info("info")
	oplog.Warn("warn")
	oplog.Error("error")
	oplog.Debug("debug")
	oplog.Trace("trace")

	_, wantFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	require.Len(t, h.records, 5)
	for _, r := range h.records {
		frame, _ := runtime.CallersFrames([]uintptr{r.PC}).Next()
		require.Equal(t, filepath.Base(wantFile), filepath.Base(frame.File),
			"%q was attributed to %s:%d, not to its call site", r.Message, frame.File, frame.Line)
	}
}
