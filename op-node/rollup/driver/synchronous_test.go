package driver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

type TestEvent struct{}

func (ev TestEvent) String() string {
	return "X"
}

func TestSynchronousEvents(t *testing.T) {
	logger := testlog.Logger(t, log.LevelError)
	ctx, cancel := context.WithCancel(context.Background())
	count := 0
	deriver := rollup.DeriverFunc(func(ev rollup.Event) {
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
	deriver := rollup.DeriverFunc(func(ev rollup.Event) {
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
