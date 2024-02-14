package testlog_test

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestCaptureLogger(t *testing.T) {
	lgr, logs := testlog.CaptureLogger(t, log.LevelInfo)
	msg := "foo bar"
	lgr.Info(msg, "a", 1)
	rec, ok := logs.FindLogContaining("foo")
	require.True(t, ok)
	require.Equal(t, msg, rec.Message)
	require.EqualValues(t, 1, rec.AttrValue("a"))

	lgr.Debug("bug")
	_, ok = logs.FindLogContaining("bug")
	require.True(t, ok, "should capture all logs, not only above level")

	msgClear := "clear"
	lgr.Error(msgClear)
	require.NotNil(t, logs.FindLog(log.LevelError, msgClear))
	logs.Clear()
	_, ok = logs.FindLogContaining(msgClear)
	require.False(t, ok)

	lgrb := lgr.New("b", 2)
	msgOp := "optimistic"
	lgrb.Info(msgOp, "c", 3)
	recOp, ok := logs.FindLogContaining(msgOp)
	require.True(t, ok, "should still capture logs from derived logger")
	require.EqualValues(t, 3, recOp.AttrValue("c"))
	// Note: "b" attributes won't be visible on captured record
}
