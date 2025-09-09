package event

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type testEvent struct{ n int }

func (testEvent) String() string { return "test-event" }

// A simple deriver that records when it runs
type recDeriver struct{ ran *atomic.Int32 }

func (r recDeriver) OnEvent(ctx context.Context, ev Event) bool {
	r.ran.Add(1)
	return true
}

func TestSingleThread_StartOnFirstAwait(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewSingleThreadCooperative(ctx)
	sys := NewSystem(logger, exec)
	ctx2 := WithSystem(ctx, sys)

	// Spawn a task that sleeps briefly and returns a value.
	started := atomic.Bool{}
	done := atomic.Bool{}
	f := func(ctx context.Context) (int, error) {
		started.Store(true)
		_ = AwaitSleep(ctx, 10*time.Millisecond)
		done.Store(true)
		return 123, nil
	}
	p := Spawn1[int, error](ctx2, f)

	// Not drained yet; since SingleThread mode, first await should trigger execution immediately.
	if started.Load() {
		t.Fatalf("task should not have started before await")
	}
	_ = p.Await(ctx2)
	if !done.Load() {
		t.Fatalf("task should be completed after await")
	}
	v, _, _ := p.Result()
	if v != 123 {
		t.Fatalf("unexpected result: %d", v)
	}
}

func TestSingleThread_DeferredIfNotAwaited(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewSingleThreadCooperative(ctx)
	sys := NewSystem(logger, exec)
	ctx2 := WithSystem(ctx, sys)

	ran := atomic.Bool{}

	f := func(ctx context.Context) error {
		ran.Store(true)
		return nil
	}
	p := Spawn0[error](ctx2, f)
	// Do not await p yet. Instead, enqueue another event and drain once.
	derivRan := atomic.Int32{}
	_ = sys.Register("rec", recDeriver{ran: &derivRan})
	em := sys.Register("em", nil)

	em.Emit(ctx2, testEvent{n: 1})
	// Drain should process async invoke (spawned) before or after based on priority; both are Normal.
	// In either case, the spawned task must have run after drain.
	_ = exec.Drain()

	if !ran.Load() {
		t.Fatalf("spawned task did not run after drain")
	}
	// Now Await to ensure promise is settled and no deadlocks
	_ = p.Await(ctx2)
}
