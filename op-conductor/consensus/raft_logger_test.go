package consensus

import (
	"log/slog"
	"testing"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestRaftLoggerUsesAppLogLevel(t *testing.T) {
	log, captured := testlog.CaptureLogger(t, slog.LevelInfo)
	raftLog := newRaftLogger(log)

	raftLog.Trace("trace message")
	raftLog.Debug("debug message")
	raftLog.Info("info message", "term", uint64(1))
	raftLog.Warn("warn message")
	raftLog.Error("error message")

	require.Nil(t, captured.FindLog(testlog.NewMessageFilter("trace message")))
	require.Nil(t, captured.FindLog(testlog.NewMessageFilter("debug message")))
	require.NotNil(t, captured.FindLog(
		testlog.NewMessageFilter("info message"),
		testlog.NewLevelFilter(slog.LevelInfo),
		testlog.NewAttributesFilter("component", "raft"),
		testlog.NewAttributesFilter("term", "1"),
	))
	require.NotNil(t, captured.FindLog(
		testlog.NewMessageFilter("warn message"),
		testlog.NewLevelFilter(slog.LevelWarn),
	))
	require.NotNil(t, captured.FindLog(
		testlog.NewMessageFilter("error message"),
		testlog.NewLevelFilter(slog.LevelError),
	))
}

func TestRaftLoggerMapsDebugAndTraceLevels(t *testing.T) {
	log, captured := testlog.CaptureLogger(t, oplog.LevelTrace)
	raftLog := newRaftLogger(log)

	raftLog.Log(hclog.Trace, "trace message")
	raftLog.Log(hclog.Debug, "debug message")

	require.NotNil(t, captured.FindLog(
		testlog.NewMessageFilter("trace message"),
		testlog.NewLevelFilter(oplog.LevelTrace),
	))
	require.NotNil(t, captured.FindLog(
		testlog.NewMessageFilter("debug message"),
		testlog.NewLevelFilter(slog.LevelDebug),
	))
}
