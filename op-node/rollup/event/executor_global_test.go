package event

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestGlobalExecutor(t *testing.T) {
	count := 0
	ex := ExecutableFunc(func(ev AnnotatedEvent) {
		count += 1
	})
	exec := NewGlobalSynchronous(context.Background())
	leave := exec.Add(ex, nil)
	require.NoError(t, exec.Drain(), "can drain, even if empty")

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.Equal(t, 0, count, "no processing yet, queued event")
	require.NoError(t, exec.Drain())
	require.Equal(t, 1, count, "processed event")

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.Equal(t, 1, count, "no processing yet, queued events")
	require.NoError(t, exec.Drain())
	require.Equal(t, 3, count, "processed events")

	leave()
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NotEqual(t, exec.Drain(), "after deriver leaves the executor can still drain events")
	require.Equal(t, 3, count, "didn't process event after trigger close")
}

func TestQueueSanityLimit(t *testing.T) {
	count := 0
	ex := ExecutableFunc(func(ev AnnotatedEvent) {
		count += 1
	})
	exec := NewGlobalSynchronous(context.Background())
	leave := exec.Add(ex, nil)
	defer leave()
	// emit 1 too many events
	for i := 0; i < sanityEventLimit; i++ {
		require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	}
	require.ErrorContains(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}), "too many events")
	require.NoError(t, exec.Drain())
	require.Equal(t, sanityEventLimit, count, "processed all non-dropped events")

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NoError(t, exec.Drain())
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
	var exec *GlobalSyncExec
	result := false
	ex := ExecutableFunc(func(ev AnnotatedEvent) {
		logger.Info("received event", "event", ev)
		switch x := ev.Event.(type) {
		case CyclicEvent:
			if x.Count < 10 {
				require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: CyclicEvent{Count: x.Count + 1}}))
			} else {
				result = true
			}
		}
	})
	exec = NewGlobalSynchronous(context.Background())
	leave := exec.Add(ex, nil)
	defer leave()
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: CyclicEvent{Count: 0}}))
	require.NoError(t, exec.Drain())
	require.True(t, result, "expecting event processing to fully recurse")
}

func TestDrainCancel(t *testing.T) {
	count := 0
	ctx, cancel := context.WithCancel(context.Background())
	ex := ExecutableFunc(func(ev AnnotatedEvent) {
		count += 1
		cancel()
	})
	exec := NewGlobalSynchronous(ctx)
	leave := exec.Add(ex, nil)
	defer leave()

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	drainErr := exec.Drain()
	require.NotNil(t, ctx.Err())
	require.ErrorIs(t, ctx.Err(), drainErr)
	require.Equal(t, 1, count, "drain must be canceled before next event is processed")
}

func TestDrainUntilCancel(t *testing.T) {
	count := 0
	ctx, cancel := context.WithCancel(context.Background())
	ex := ExecutableFunc(func(ev AnnotatedEvent) {
		count += 1
		if _, ok := ev.Event.(FooEvent); ok {
			cancel()
		}
	})
	exec := NewGlobalSynchronous(ctx)
	leave := exec.Add(ex, nil)
	defer leave()

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: FooEvent{}}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	drainErr := exec.DrainUntil(Is[FooEvent], false)
	require.NoError(t, drainErr, "drained right until context started to matter")
	require.Equal(t, 2, count, "drain must be stopped at Foo (incl)")
	drainErr = exec.DrainUntil(Is[TestEvent], false)
	require.NotNil(t, ctx.Err())
	require.NotNil(t, drainErr)
	require.ErrorIs(t, ctx.Err(), drainErr)
	require.Equal(t, 2, count, "drain must be canceled, not processed next TestEvent")
}

func TestDrainUntilExcl(t *testing.T) {
	count := 0
	ex := ExecutableFunc(func(ev AnnotatedEvent) {
		count += 1
	})
	exec := NewGlobalSynchronous(context.Background())
	leave := exec.Add(ex, nil)
	defer leave()

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: FooEvent{}}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}}))
	require.NoError(t, exec.DrainUntil(Is[FooEvent], true))
	require.Equal(t, 1, count, "Foo must not be processed yet")
	require.NoError(t, exec.DrainUntil(Is[FooEvent], true))
	require.Equal(t, 1, count, "Foo still not processed, excl on first element")
	require.NoError(t, exec.DrainUntil(Is[FooEvent], false))
	require.Equal(t, 2, count, "Foo is processed, remainder is not, stop is inclusive now")
	require.NoError(t, exec.Drain())
	require.Equal(t, 4, count, "Done")
}
