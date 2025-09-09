package event

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
)

type Ready struct{ N int }

func (Ready) String() string { return "ready" }

//

func TestCooperativeAwaitEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewCooperative(ctx)
	sys := NewSystem(logger, exec)

	// Background drain loop
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			_ = exec.Drain()
			select {
			case <-ctx.Done():
				return
			case <-exec.Await():
			}
		}
	}()

	// Async function that waits for a Ready event and returns its N.
	f := func(ctx context.Context) (int, error) {
		// Register a one-shot waiter for Ready events directly on the cooperative executor
		sys2, _ := SystemFromContext(ctx)
		s := sys2.(*Sys)
		ex := s.executor.(*CooperativeExec)
		ch := make(chan AnnotatedEvent, 1)
		ex.registerEventWaiter(ByType[Ready](), ch)
		defer ex.unregisterEventWaiter(ch)
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case ev := <-ch:
			rdy, _ := ev.Event.(Ready)
			return rdy.N, nil
		}
	}

	ctx2 := WithSystem(ctx, sys)
	p := Spawn1[int, error](ctx2, f, WithSpawnPriority(Normal), WithSpawnName("await-ready"))

	em := sys.Register("app", nil)
	// Emit repeatedly until the handler completes to avoid race with waiter registration.
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				emitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				em.Emit(emitCtx, Ready{N: 7})
				cancel()
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	_ = p.Await(ctx2)
	got, _, _ := p.Result()
	if got != 7 {
		t.Fatalf("unexpected result: got %d, want %d", got, 7)
	}
	// stop emitter loop
	cancel()
	<-done
}

func TestCooperativeAwaitPromise(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewCooperative(ctx)
	sys := NewSystem(logger, exec)

	// Background drain loop
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			_ = exec.Drain()
			select {
			case <-ctx.Done():
				return
			case <-exec.Await():
			}
		}
	}()

	p0, r := NewPromise1[int, error]()

	f := func(ctx context.Context) (int, error) {
		_ = p0.Await(ctx)
		v, err, _ := p0.Result()
		if err != nil {
			return 0, err
		}
		return v, nil
	}

	ctx2 := WithSystem(ctx, sys)
	p1 := Spawn1[int, error](ctx2, f, WithSpawnPriority(Normal), WithSpawnName("await-promise"))

	// Resolve asynchronously
	go func() {
		time.Sleep(10 * time.Millisecond)
		r.Resolve(42)
	}()

	_ = p1.Await(ctx2)
	got, _, _ := p1.Result()
	if got != 42 {
		t.Fatalf("unexpected result: got %d, want %d", got, 42)
	}
}

func TestSpawnPriorityOrdering(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewSingleThreadCooperative(ctx)
	sys := NewSystem(logger, exec)

	// Record completion in a shared slice to assert order

	// Shared slice to record completion order
	var mu sync.Mutex
	var order []string

	// two async functions that append their name on completion
	f := func(name string, delay time.Duration) Async0[error] {
		return func(ctx context.Context) error {
			ctx = WithSystem(ctx, sys)
			_ = AwaitSleep(ctx, delay)
			mu.Lock()
			order = append(order, name)
			mu.Unlock()
			return nil
		}
	}

	// Spawn low-priority and high-priority tasks (same delay; priority must decide order)
	ctx2 := WithSystem(ctx, sys)
	_ = Spawn0[error](ctx2, f("low", 5*time.Millisecond), WithSpawnPriority(Low), WithSpawnName("low"))
	_ = Spawn0[error](ctx2, f("high", 5*time.Millisecond), WithSpawnPriority(High), WithSpawnName("high"))

	// Now start drain loop so both events are present and priority ordering applies
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}
			_ = exec.Drain()
			select {
			case <-ctx.Done():
				return
			case <-exec.Await():
			}
		}
	}()

	// Wait until both complete
	for {
		mu.Lock()
		n := len(order)
		mu.Unlock()
		if n == 2 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for tasks to complete, order=%v", order)
		default:
			time.Sleep(1 * time.Millisecond)
		}
	}

	// Assert that high priority completed before low priority
	mu.Lock()
	defer mu.Unlock()
	if len(order) != 2 {
		t.Fatalf("unexpected completion count: %d", len(order))
	}
	if order[0] != "high" {
		t.Fatalf("expected high to complete first, got %+v", order)
	}

}

func TestAwaitAny(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewCooperative(ctx)
	sys := NewSystem(logger, exec)

	// Bind system to context for ctx-only helpers
	ctx = WithSystem(ctx, sys)

	p1, r1 := NewPromise1[int, error]()
	p2, r2 := NewPromise1[int, error]()

	// Resolve p2 first
	go func() {
		time.Sleep(10 * time.Millisecond)
		r2.Resolve(2)
	}()
	go func() {
		time.Sleep(30 * time.Millisecond)
		r1.Resolve(1)
	}()

	idx, v, perr, err := AwaitAny[int, error](ctx, p1, p2)
	if err != nil {
		t.Fatalf("await any error: %v", err)
	}
	if perr != nil {
		t.Fatalf("unexpected per-promise error: %v", perr)
	}
	if idx != 1 || v != 2 {
		t.Fatalf("unexpected (idx,value): got (%d,%d), want (1,2)", idx, v)
	}
}

func TestAwaitAll(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewCooperative(ctx)
	sys := NewSystem(logger, exec)

	ctx = WithSystem(ctx, sys)

	p1, r1 := NewPromise1[int, error]()
	p2, r2 := NewPromise1[int, error]()
	p3, r3 := NewPromise1[int, error]()

	go func() { time.Sleep(30 * time.Millisecond); r1.Resolve(5) }()
	go func() { time.Sleep(10 * time.Millisecond); r2.Resolve(6) }()
	go func() { time.Sleep(20 * time.Millisecond); r3.Resolve(7) }()

	vals, errs, err := AwaitAll[int, error](ctx, p1, p2, p3)
	if err != nil {
		t.Fatalf("await all error: %v", err)
	}
	if len(vals) != 3 || len(errs) != 3 {
		t.Fatalf("unexpected lengths: vals=%d errs=%d", len(vals), len(errs))
	}
	if vals[0] != 5 || vals[1] != 6 || vals[2] != 7 {
		t.Fatalf("unexpected vals: %v", vals)
	}
	if errs[0] != nil || errs[1] != nil || errs[2] != nil {
		t.Fatalf("unexpected per-promise errors: %v", errs)
	}
}

func TestAsyncWrappedPromise(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	logger := log.New()
	exec := NewCooperative(ctx)
	sys := NewSystem(logger, exec)
	ctx = WithSystem(ctx, sys)

	go func() { _ = sys.Drive(ctx) }()

	doBlockingTask := func(ctx context.Context) Promise1[int, error] {
		return Spawn1(ctx, func(ctx context.Context) (int, error) {
			pp, r := NewPromise1[int, error]()
			go func() { time.Sleep(10 * time.Millisecond); r.Resolve(42) }()

			_ = pp.Await(ctx)
			v, err, _ := pp.Result()
			return v, err
		})
	}

	p := doBlockingTask(ctx)
	err := p.Await(ctx) // blocks
	if err != nil {
		t.Fatalf("context timeout")
	}
	result, perr, _ := p.Result()
	if perr != nil {
		t.Fatalf("await error: %v", perr)
	}
	if result != 42 {
		t.Fatalf("unexpected result: got %d, want 42", result)
	}
}
