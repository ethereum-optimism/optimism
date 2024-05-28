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
	msgFilter := testlog.NewMessageFilter(msg)
	rec := logs.FindLog(msgFilter)
	require.Equal(t, msg, rec.Message)
	require.EqualValues(t, 1, rec.AttrValue("a"))

	lgr.Debug("bug")
	containsFilter := testlog.NewMessageContainsFilter("bug")
	l := logs.FindLog(containsFilter)
	require.NotNil(t, l, "should capture all logs, not only above level")

	msgClear := "clear"
	lgr.Error(msgClear)
	levelFilter := testlog.NewLevelFilter(log.LevelError)
	msgFilter = testlog.NewMessageFilter(msgClear)
	require.NotNil(t, logs.FindLog(levelFilter, msgFilter))
	logs.Clear()
	containsFilter = testlog.NewMessageContainsFilter(msgClear)
	l = logs.FindLog(containsFilter)
	require.Nil(t, l)

	lgrb := lgr.New("b", 2)
	msgOp := "optimistic"
	lgrb.Info(msgOp, "c", 3)
	containsFilter = testlog.NewMessageContainsFilter(msgOp)
	recOp := logs.FindLog(containsFilter)
	require.NotNil(t, recOp, "should still capture logs from derived logger")
	require.EqualValues(t, 3, recOp.AttrValue("c"))
	// Note: "b" attributes won't be visible on captured record
}

func TestCaptureLoggerAttributesFilter(t *testing.T) {
	lgr, logs := testlog.CaptureLogger(t, log.LevelInfo)
	msg := "foo bar"
	lgr.Info(msg, "a", "test")
	lgr.Info(msg, "a", "test 2")
	lgr.Info(msg, "a", "random")
	msgFilter := testlog.NewMessageFilter(msg)
	attrFilter := testlog.NewAttributesFilter("a", "random")

	rec := logs.FindLog(msgFilter, attrFilter)
	require.Equal(t, msg, rec.Message)
	require.EqualValues(t, "random", rec.AttrValue("a"))

	recs := logs.FindLogs(msgFilter, attrFilter)
	require.Len(t, recs, 1)
}
