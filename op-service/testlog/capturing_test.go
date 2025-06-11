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
	require.Equal(t, msg, rec.Record.Message)
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
	require.Equal(t, msg, rec.Record.Message)
	require.EqualValues(t, "random", rec.AttrValue("a"))

	recs := logs.FindLogs(msgFilter, attrFilter)
	require.Len(t, recs, 1)
}

func TestCaptureLoggerNested(t *testing.T) {
	lgrInner, logs := testlog.CaptureLogger(t, log.LevelInfo)

	lgrInner.Info("hi", "a", "test")

	lgrChildX := lgrInner.With("name", "childX")
	lgrChildX.Info("hello", "b", "42")

	lgrChildY := lgrInner.With("name", "childY")
	lgrChildY.Info("hola", "c", "7")

	lgrInner.Info("hello universe", "greeting", "from Inner")

	lgrChildX.Info("hello world", "greeting", "from X")

	require.Len(t, logs.FindLogs(testlog.NewAttributesFilter("name", "childX")), 2, "X logged twice")
	require.Len(t, logs.FindLogs(testlog.NewAttributesFilter("name", "childY")), 1, "Y logged once")

	require.Len(t, logs.FindLogs(
		testlog.NewAttributesContainsFilter("greeting", "from")), 2, "two greetings")
	require.Len(t, logs.FindLogs(
		testlog.NewAttributesContainsFilter("greeting", "from"),
		testlog.NewAttributesFilter("name", "childX")), 1, "only one greeting from X")

	require.Len(t, logs.FindLogs(
		testlog.NewAttributesFilter("a", "test")), 1, "root logger logged 'a' once")
}
