package event

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestSysTracing(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	ex := NewGlobalSynchronous(context.Background())
	sys := NewSystem(logger, ex)
	count := 0
	foo := DeriverFunc(func(ev Event) bool {
		switch ev.(type) {
		case TestEvent:
			count += 1
			return true
		}
		return false
	})
	lgr, logs := testlog.CaptureLogger(t, log.LevelDebug)
	logTracer := NewLogTracer(lgr, log.LevelDebug)
	sys.AddTracer(logTracer)

	em := sys.Register("foo", foo, DefaultRegisterOpts())
	em.Emit(TestEvent{})
	require.Equal(t, 0, count, "no event processing before synchronous executor explicitly drains")
	require.NoError(t, ex.Drain())
	require.Equal(t, 1, count)

	hasDebugLevel := testlog.NewLevelFilter(log.LevelDebug)
	require.NotNil(t, logs.FindLog(hasDebugLevel,
		testlog.NewMessageContainsFilter("Emitting event")))
	require.NotNil(t, logs.FindLog(hasDebugLevel,
		testlog.NewMessageContainsFilter("Processing event")))
	require.NotNil(t, logs.FindLog(hasDebugLevel,
		testlog.NewMessageContainsFilter("Processed event")))
	em.Emit(FooEvent{})
	require.NoError(t, ex.Drain())
	require.Equal(t, 1, count, "foo does not count")

	em.Emit(TestEvent{})
	require.NoError(t, ex.Drain())
	require.Equal(t, 2, count)

	logs.Clear()
	sys.RemoveTracer(logTracer)
	em.Emit(TestEvent{})
	require.NoError(t, ex.Drain())
	require.Equal(t, 3, count)
	require.Equal(t, 0, len(*logs.Logs), "no logs when tracer is not active anymore")
}

func TestSystemBroadcast(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	ex := NewGlobalSynchronous(context.Background())
	sys := NewSystem(logger, ex)
	fooCount := 0
	foo := DeriverFunc(func(ev Event) bool {
		switch ev.(type) {
		case TestEvent:
			fooCount += 1
		case FooEvent:
			fooCount += 1
		default:
			return false
		}
		return true
	})
	barCount := 0
	bar := DeriverFunc(func(ev Event) bool {
		switch ev.(type) {
		case TestEvent:
			barCount += 1
		case BarEvent:
			barCount += 1
		default:
			return false
		}
		return true
	})
	fooEm := sys.Register("foo", foo, DefaultRegisterOpts())
	fooEm.Emit(TestEvent{})
	barEm := sys.Register("bar", bar, DefaultRegisterOpts())
	barEm.Emit(TestEvent{})
	// events are broadcast to every deriver, regardless who sends them
	require.NoError(t, ex.Drain())
	require.Equal(t, 2, fooCount)
	require.Equal(t, 2, barCount)
	// emit from bar, process in foo
	barEm.Emit(FooEvent{})
	require.NoError(t, ex.Drain())
	require.Equal(t, 3, fooCount)
	require.Equal(t, 2, barCount)
	// emit from foo, process in bar
	fooEm.Emit(BarEvent{})
	require.NoError(t, ex.Drain())
	require.Equal(t, 3, fooCount)
	require.Equal(t, 3, barCount)
}

func TestCriticalError(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	count := 0
	seenCrit := 0
	deriverFn := DeriverFunc(func(ev Event) bool {
		switch ev.(type) {
		case CriticalErrorEvent:
			seenCrit += 1
		default:
			count += 1
		}
		return true
	})
	exec := NewGlobalSynchronous(context.Background())
	sys := NewSystem(logger, exec)
	emitterA := sys.Register("a", deriverFn, DefaultRegisterOpts())
	emitterB := sys.Register("b", deriverFn, DefaultRegisterOpts())

	require.NoError(t, exec.Drain(), "can drain, even if empty")
	emitterA.Emit(TestEvent{})
	require.Equal(t, 0, count, "no processing yet, queued event")
	require.NoError(t, exec.Drain())
	require.Equal(t, 2, count, "both A and B processed the event")

	emitterA.Emit(TestEvent{})
	emitterB.Emit(TestEvent{})
	testErr := errors.New("test crit error")
	emitterB.Emit(CriticalErrorEvent{Err: testErr})
	require.Equal(t, 2, count, "no processing yet, queued events")
	require.Equal(t, 0, seenCrit, "critical error events are still scheduled like normal")
	require.True(t, sys.abort.Load(), "we are aware of the crit")
	require.NoError(t, exec.Drain())
	require.Equal(t, 2, count, "still no processing, since we hit a crit error, the events are ignored")
	require.Equal(t, 2, seenCrit, "but everyone has seen the crit now")

	// We are able to stop the processing now
	sys.Stop()

	emitterA.Emit(TestEvent{})
	require.NoError(t, exec.Drain(), "system is closed, no further event processing")
	require.Equal(t, 2, count)
}
