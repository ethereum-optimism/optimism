## Event package tutorial: cooperative executors and promises

This tutorial shows how to use the cooperative event system with promises to write simple, readable concurrent code without goroutine leaks or opaque channels.

### Core ideas

- **System + Executor**: a `System` routes events through an `Executor` that cooperatively schedules work.
- **Promises**: `Promise0/1/2` represent future results. They settle exactly once, with a value(s) or an error.
- **Spawn**: `Spawn0/1/2` submit asynchronous work to the event loop and return a promise immediately.
- **Await**: call `p.Await(ctx)` on a promise to wait cooperatively for it to settle, yielding to the executor instead of blocking threads.

### Quick start

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

logger := log.New()
exec := event.NewCooperative(ctx)
sys := event.NewSystem(logger, exec)
ctx = event.WithSystem(ctx, sys)

// Drive the executor via the System helper (must be running for work to execute)
go func() {
    if err := sys.Drive(ctx); err != nil && err != ctx.Err() {
        logger.Error("executor drive error", "err", err)
    }
}()

// Spawn async work that returns a value
p := event.Spawn1[int, error](ctx, func(ctx context.Context) (int, error) {
    // Cooperatively sleep; does not block other tasks from running
    _ = event.AwaitSleep(ctx, 50*time.Millisecond)
    return 123, nil
})

// Await the promise (inside handlers this yields cooperatively; here it blocks while sys.Drive runs)
if err := p.Await(ctx); err != nil { /* handle ctx error */ }
v, perr, _ := p.Result()
if perr != nil { /* handle task error */ }
_ = v // 123
```

### Promises at a glance

- **Arity**
  - `Promise0[E]` — completion with only error
  - `Promise1[T,E]` — one value + error
  - `Promise2[A,B,E]` — two values + error
- **API**
  - `Ready() <-chan struct{}` — closed when settled
  - `Result()` — returns values, error, and `settled` boolean (non-blocking; will always return immediately)
  - `Await(ctx context.Context) error` — Blocks until ready or ctx cancel; yields cooperatively if we're inside a task
- **Construction**: User code usually doesn't calls `NewPromise*` directly; promises are returned from calls to `Spawn*` functions.

### Spawning work

```go
// Spawn0: returns only an error
p0 := event.Spawn0[error](ctx, func(ctx context.Context) error {
    // ...
    return nil
})

// Spawn1: returns T and error
p1 := event.Spawn1[string, error](ctx, func(ctx context.Context) (string, error) {
    return "ok", nil
})

// Spawn2: returns two values and error
p2 := event.Spawn2[int, int, error](ctx, func(ctx context.Context) (int, int, error) {
    return 1, 2, nil
})
```

Spawn options:

```go
p := event.Spawn1[int, error](
    ctx,
    work,
    event.WithSpawnPriority(event.High),
    event.WithSpawnName("my-task"),

    // The following does not emit `WorkEvent`; it only keeps test helpers like
    // `drainer.DrainUntil(WorkEvent)` working.
    event.WithSpawnLegacyEvent(WorkEvent{Attributes: attrs})
)
```

#### Awaiting inside vs. outside event handlers

- Inside an event handler: the framework injects the `System` into the handler `ctx`, and the executor implements cooperative yield. `p.Await(ctx)` will temporarily yield so other work can progress, avoiding deadlocks.

```go
type worker struct{}

func (worker) OnEvent(ctx context.Context, ev event.Event) bool {
    // ctx already carries the System (injected by the framework)
    p := event.Spawn1[int, error](ctx, work)
    if err := p.Await(ctx); err != nil {
        return false // ctx canceled
    }
    v, perr, _ := p.Result()
    if perr != nil {
        return false
    }
    _ = v
    return true
}
```

- Outside an event handler: `p.Await(ctx)` blocks the calling goroutine until the promise settles or `ctx` is canceled. It does not yield. Only awaits executed inside executor-managed handler goroutines yield cooperatively. You may still bind the `System` to context for other utilities, but it does not change the blocking behavior of `Await` outside handlers.

```go
// Bind system to ctx for utilities; Await will still block outside handlers
ctx = event.WithSystem(ctx, sys)
go func() {
    if err := sys.Drive(ctx); err != nil && err != ctx.Err() {
        logger.Error("executor drive error", "err", err)
    }
}()

p := event.Spawn1[int, error](ctx, work)
_ = p.Await(ctx) // blocks; does not yield outside handlers
```

### Any/All combinators

```go
idx, v, perr, err := event.AwaitAny[int, error](ctx, p1, p2, p3)
// err is ctx error; perr is the winner's per-promise error

vals, errs, err := event.AwaitAll[int, error](ctx, p1, p2, p3)
// errs[i] is the per-promise error for vals[i]
```

### Error model and sentinel errors

- Application errors are produced by your async functions and settle into the promise (e.g., `return 0, errors.New("boom")`).
- System/setup errors are never returned immediately. They pre-reject the returned promise:
  - `event.ErrNoSystemInContext` — missing `System` in `ctx`
  - `event.ErrUnexpectedSystemType` — context contained a non-`*Sys` implementation
  - `event.ErrExecutorNotCooperative` — awaiting events requires a cooperative executor
  - `event.ErrTaskRunnerInit` — failed to prepare the internal task runner

Check them like any per-promise error:

```go
p := event.Spawn1[int, error](context.Background(), work) // no system bound
_ = event.AwaitPromise1(ctx, p) // returns ctx.Err() on cancel; promise remains pre-rejected
_, perr, _ := p.Result()
if errors.Is(perr, event.ErrNoSystemInContext) {
    // bind a System to ctx via event.WithSystem and try again
}
```
