package testlog_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

func TestCaptureLogger(t *testing.T) {
	lgr, logs := testlog.CaptureLogger(t, log.LevelTrace)
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

func TestCapturingAttrHandlerCapturesFirstMatchAndForwards(t *testing.T) {
	delegate, handled := newCountingHandler()
	handler := testlog.NewCapturingAttrHandler(delegate, "started metrics server", "addr")
	logger := slog.New(handler)

	logger.Info("unrelated", "addr", "ignored")
	logger.Info("started metrics server", "other", "ignored")
	logger.Info("started metrics server", "addr", "first")
	logger.Info("started metrics server", "addr", "second")

	value, ok := handler.Captured()
	require.True(t, ok)
	require.Equal(t, "first", value.String())
	require.Equal(t, int64(4), handled.Load())
}

func TestCapturingAttrHandlerSharesCaptureAcrossDerivedHandlers(t *testing.T) {
	tests := []struct {
		name string
		log  func(*slog.Logger)
		want string
	}{
		{
			name: "inherited attribute",
			log: func(logger *slog.Logger) {
				logger.With("addr", "inherited").Info("started metrics server")
			},
			want: "inherited",
		},
		{
			name: "grouped handler",
			log: func(logger *slog.Logger) {
				logger.WithGroup("network").Info("started metrics server", "addr", "grouped")
			},
			want: "grouped",
		},
		{
			name: "group attribute",
			log: func(logger *slog.Logger) {
				logger.Info(
					"started metrics server",
					slog.Group("network", slog.String("addr", "nested")),
				)
			},
			want: "nested",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			delegate, _ := newCountingHandler()
			handler := testlog.NewCapturingAttrHandler(delegate, "started metrics server", "addr")
			test.log(slog.New(handler))

			value, ok := handler.Captured()
			require.True(t, ok)
			require.Equal(t, test.want, value.String())
		})
	}
}

func TestCapturingAttrHandlerOwnsInheritedAttributes(t *testing.T) {
	delegate := &mutatingAttrsHandler{Handler: slog.NewTextHandler(io.Discard, nil)}
	handler := testlog.NewCapturingAttrHandler(delegate, "started metrics server", "addr")

	slog.New(handler).With("addr", "original").Info("started metrics server")

	value, ok := handler.Captured()
	require.True(t, ok)
	require.Equal(t, "original", value.String())
}

func TestCapturingAttrHandlerIsSafeForConcurrentUse(t *testing.T) {
	delegate, handled := newCountingHandler()
	handler := testlog.NewCapturingAttrHandler(delegate, "target", "id")
	logger := slog.New(handler)

	const logCount = 100
	var wg sync.WaitGroup
	wg.Add(logCount)
	for i := range logCount {
		go func() {
			defer wg.Done()
			logger.Info("target", "id", i)
		}()
	}
	wg.Wait()

	value, ok := handler.Captured()
	require.True(t, ok)
	require.GreaterOrEqual(t, value.Int64(), int64(0))
	require.Less(t, value.Int64(), int64(logCount))
	require.Equal(t, int64(logCount), handled.Load())
}

type countingHandler struct {
	slog.Handler
	handled *atomic.Int64
}

func newCountingHandler() (*countingHandler, *atomic.Int64) {
	handled := &atomic.Int64{}
	return &countingHandler{
		Handler: slog.NewTextHandler(io.Discard, nil),
		handled: handled,
	}, handled
}

func (h *countingHandler) Handle(ctx context.Context, record slog.Record) error {
	h.handled.Add(1)
	return h.Handler.Handle(ctx, record)
}

func (h *countingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &countingHandler{Handler: h.Handler.WithAttrs(attrs), handled: h.handled}
}

func (h *countingHandler) WithGroup(name string) slog.Handler {
	return &countingHandler{Handler: h.Handler.WithGroup(name), handled: h.handled}
}

type mutatingAttrsHandler struct {
	slog.Handler
}

func (h *mutatingAttrsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	attrs[0] = slog.String("delegate-owned", "mutated")
	return &mutatingAttrsHandler{Handler: h.Handler.WithAttrs(attrs)}
}
