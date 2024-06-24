package rollup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestSynchronousEvents(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	ctx, cancel := context.WithCancel(context.Background())
	count := 0
	deriver := DeriverFunc(func(ev Event) {
		count += 1
	})
	syncEv := NewSynchronousEvents(logger, ctx, deriver)
	require.NoError(t, syncEv.Drain(), "can drain, even if empty")

	syncEv.Emit(TestEvent{})
	require.Equal(t, 0, count, "no processing yet, queued event")
	require.NoError(t, syncEv.Drain())
	require.Equal(t, 1, count, "processed event")

	syncEv.Emit(TestEvent{})
	syncEv.Emit(TestEvent{})
	require.Equal(t, 1, count, "no processing yet, queued events")
	require.NoError(t, syncEv.Drain())
	require.Equal(t, 3, count, "processed events")

	cancel()
	syncEv.Emit(TestEvent{})
	require.Equal(t, ctx.Err(), syncEv.Drain(), "no draining after close")
	require.Equal(t, 3, count, "didn't process event after trigger close")
}

func TestSynchronousEventsSanityLimit(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	count := 0
	deriver := DeriverFunc(func(ev Event) {
		count += 1
	})
	syncEv := NewSynchronousEvents(logger, context.Background(), deriver)
	// emit 1 too many events
	for i := 0; i < sanityEventLimit+1; i++ {
		syncEv.Emit(TestEvent{})
	}
	require.NoError(t, syncEv.Drain())
	require.Equal(t, sanityEventLimit, count, "processed all non-dropped events")

	syncEv.Emit(TestEvent{})
	require.NoError(t, syncEv.Drain())
	require.Equal(t, sanityEventLimit+1, count, "back to normal after drain")
}

type CyclicEvent struct {
	Count int
}

func (ev CyclicEvent) String() string {
	return "cyclic-event"
}

func TestSynchronousCyclic(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	var emitter EventEmitter
	result := false
	deriver := DeriverFunc(func(ev Event) {
		logger.Info("received event", "event", ev)
		switch x := ev.(type) {
		case CyclicEvent:
			if x.Count < 10 {
				emitter.Emit(CyclicEvent{Count: x.Count + 1})
			} else {
				result = true
			}
		}
	})
	syncEv := NewSynchronousEvents(logger, context.Background(), deriver)
	emitter = syncEv
	syncEv.Emit(CyclicEvent{Count: 0})
	require.NoError(t, syncEv.Drain())
	require.True(t, result, "expecting event processing to fully recurse")
}
