package event

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

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
	leave := exec.Add(ex, &ExecutorConfig{Priority: Normal})
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
	leave := exec.Add(ex, &ExecutorConfig{Priority: Normal})
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
	Ctx
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
	leave := exec.Add(ex, &ExecutorConfig{Priority: Normal})
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
	leave := exec.Add(ex, &ExecutorConfig{Priority: Normal})
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
	leave := exec.Add(ex, &ExecutorConfig{Priority: Normal})
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
	leave := exec.Add(ex, &ExecutorConfig{Priority: Normal})
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

func TestPrioritized(t *testing.T) {
	items := []string{}
	ex1 := ExecutableFunc(func(ev AnnotatedEvent) {
		items = append(items, fmt.Sprintf("ex1: %d", ev.EmitContext))
	})
	ex2 := ExecutableFunc(func(ev AnnotatedEvent) {
		items = append(items, fmt.Sprintf("ex2: %d", ev.EmitContext))
	})
	ex3 := ExecutableFunc(func(ev AnnotatedEvent) {
		items = append(items, fmt.Sprintf("ex3: %d", ev.EmitContext))
	})
	ex4 := ExecutableFunc(func(ev AnnotatedEvent) {
		items = append(items, fmt.Sprintf("ex4: %d", ev.EmitContext))
	})
	exec := NewGlobalSynchronous(context.Background())
	leave1 := exec.Add(ex1, &ExecutorConfig{Priority: Low})
	leave2 := exec.Add(ex2, &ExecutorConfig{Priority: High})
	leave3 := exec.Add(ex3, &ExecutorConfig{Priority: Normal})
	leave4 := exec.Add(ex4, &ExecutorConfig{Priority: Normal})
	defer leave1()
	defer leave2()
	defer leave3()
	defer leave4()

	require.NoError(t, exec.Enqueue(AnnotatedEvent{
		Event:        FooEvent{},
		EmitContext:  0,
		EmitPriority: Low,
	}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{
		Event:        TestEvent{},
		EmitContext:  1,
		EmitPriority: High,
	}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{
		Event:        TestEvent{},
		EmitContext:  2,
		EmitPriority: Normal,
	}))
	require.NoError(t, exec.Drain())

	out := "\n" + strings.Join(items, "\n") + "\n"
	// Enqueued events are executed based on their emit priority.
	// Once an event is selected, the event is executed by executors in order of executor priority.
	expected := `
ex2: 1
ex3: 1
ex4: 1
ex1: 1
ex2: 2
ex3: 2
ex4: 2
ex1: 2
ex2: 0
ex3: 0
ex4: 0
ex1: 0
`
	require.Equal(t, expected, out)
	items = []string{}
	// Try emit another event, with a previously seen priority.
	// Other priorities may not have any events.
	require.NoError(t, exec.Enqueue(AnnotatedEvent{
		Event:        TestEvent{},
		EmitContext:  3,
		EmitPriority: High,
	}))
	// And another. FIFO.
	require.NoError(t, exec.Enqueue(AnnotatedEvent{
		Event:        TestEvent{},
		EmitContext:  4,
		EmitPriority: High,
	}))
	require.NoError(t, exec.Drain())
	out = "\n" + strings.Join(items, "\n") + "\n"
	expected = `
ex2: 3
ex3: 3
ex4: 3
ex1: 3
ex2: 4
ex3: 4
ex4: 4
ex1: 4
`
	require.Equal(t, expected, out)
}

func TestAwait(t *testing.T) {
	count := 0
	ex := ExecutableFunc(func(ev AnnotatedEvent) {
		count += 1
	})
	exec := NewGlobalSynchronous(context.Background())
	leave := exec.Add(ex, &ExecutorConfig{Priority: Normal})
	defer leave()

	// this event should be picked up as pre-existing queued event
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}, EmitContext: 1}))
	require.NotZero(t, exec.events.Count())
	ch := exec.Await()
	select {
	case <-time.After(time.Second * 10):
		t.Fatal("timeout")
	case _, ok := <-ch:
		require.False(t, ok, "should be unblocked now")
	}
	require.NoError(t, exec.Drain())
	require.Zero(t, exec.events.Count())

	// Here we expect to not have any events, yet.
	ch = exec.Await()
	select {
	case <-ch:
		t.Fatal("should not be unlocked yet")
	default:
	}
	// Enqueue an event, see if it unblocks the await as expected
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}, EmitContext: 2}))
	require.NotZero(t, exec.events.Count())
	select {
	case _, ok := <-ch:
		require.False(t, ok, "should be unblocked now")
	default:
		t.Fatal("should not be blocked")
	}
	require.NoError(t, exec.Drain())
	require.Zero(t, exec.events.Count())

	// Now try a pre-existing await, followed by multiple queued events, to drain all at once.
	ch = exec.Await()
	select {
	case <-ch:
		t.Fatal("should not be unlocked yet")
	default:
	}
	ch2 := exec.Await()
	require.Equal(t, ch, ch2, "should reuse the same channel")

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}, EmitContext: 3}))

	ch3 := exec.Await()
	require.NotEqual(t, ch, ch3, "enqueue should replace the await channel")

	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}, EmitContext: 4}))
	require.NoError(t, exec.Enqueue(AnnotatedEvent{Event: TestEvent{}, EmitContext: 5}))
	require.Equal(t, uint64(3), exec.events.Count())

	select {
	case _, ok := <-ch:
		require.False(t, ok, "original should be unblocked since a while")
	default:
		t.Fatal("should not be blocked")
	}
	select {
	case _, ok := <-ch3:
		require.False(t, ok, "later channel should be unblocked also")
	default:
		t.Fatal("should not be blocked")
	}
	require.NoError(t, exec.Drain())
	require.Zero(t, exec.events.Count())
}
