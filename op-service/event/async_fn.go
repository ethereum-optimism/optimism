package event

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
)

// Async arities with explicit error type parameter E.
type Async0[E interface{ error }] func(ctx context.Context) E
type Async1[T any, E interface{ error }] func(ctx context.Context) (T, E)
type Async2[T1 any, T2 any, E interface{ error }] func(ctx context.Context) (T1, T2, E)

// asyncInvokeEvent schedules a function by ID via the cooperative loop.
type asyncInvokeEvent struct {
	id          string
	legacyEvent Event
}

func (asyncInvokeEvent) String() string { return "async-invoke" }

// taskRunner receives asyncInvokeEvent and runs the associated function.
type taskRunner struct {
	em    Emitter
	sys   System
	mu    sync.Mutex
	tasks map[string]func(ctx context.Context)
	ctr   atomic.Uint64
}

func (tr *taskRunner) nextID() string { return strconv.FormatUint(tr.ctr.Add(1), 10) }

func (tr *taskRunner) OnEvent(ctx context.Context, ev Event) bool {
	a, ok := ev.(asyncInvokeEvent)
	if !ok {
		return false
	}
	tr.mu.Lock()
	fn := tr.tasks[a.id]
	if fn == nil {
		tr.mu.Unlock()
		return false
	}
	delete(tr.tasks, a.id)
	tr.mu.Unlock()
	// ensure the context is system-aware for ctx-only APIs
	ctx = WithSystem(ctx, tr.sys)
	fn(ctx)
	return true
}

var (
	runnersMu sync.Mutex
	// runners is keyed by System, then by Priority. This allows us to respect spawn priority
	// while keeping the ability to collapse to a single Normal-priority runner.
	runners = make(map[*Sys]map[Priority]*taskRunner)
	// Toggle to enable/disable respecting SpawnN priority. If false, all spawns use Normal.
	respectSpawnPriority = true
)

// SetSpawnPriorityEnabled toggles whether SpawnN respects the provided priority (default: true).
func SetSpawnPriorityEnabled(enabled bool) { respectSpawnPriority = enabled }

func ensureTaskRunner(s System, pr Priority) (*taskRunner, error) {
	sys, ok := s.(*Sys)
	if !ok {
		return nil, fmt.Errorf("unexpected system type")
	}
	// Optionally collapse to Normal priority runner.
	effectivePr := pr
	if !respectSpawnPriority {
		effectivePr = Normal
	}

	runnersMu.Lock()
	defer runnersMu.Unlock()
	byPr, ok := runners[sys]
	if !ok {
		byPr = make(map[Priority]*taskRunner)
		runners[sys] = byPr
	}
	if tr := byPr[effectivePr]; tr != nil {
		return tr, nil
	}
	// Register a new runner with both executor and emitter priorities set.
	tr := &taskRunner{sys: sys, tasks: make(map[string]func(context.Context))}
	name := "async-runner-" + strconv.Itoa(int(effectivePr))
	em := sys.Register(name, tr, WithExecPriority(effectivePr), WithEmitPriority(effectivePr))
	tr.em = em
	byPr[effectivePr] = tr
	return tr, nil
}

// runNow executes the task with given id immediately if still pending.
// It is safe to call multiple times; only the first call that finds the task runs it.
func (tr *taskRunner) runNow(id string) {
	tr.mu.Lock()
	fn := tr.tasks[id]
	if fn != nil {
		delete(tr.tasks, id)
	}
	tr.mu.Unlock()
	if fn == nil {
		return
	}
	// Use a background context; the function should derive the System from context APIs.
	ctx := WithSystem(context.Background(), tr.sys)
	fn(ctx)
}

// Spawn options to configure priority and name (optional).
type SpawnOption func(*SpawnConfig)

type SpawnConfig struct {
	Priority    Priority
	Name        string
	LegacyEvent Event
}

func defaultSpawnConfig() *SpawnConfig {
	return &SpawnConfig{Priority: Normal, Name: ""}
}

func WithSpawnPriority(p Priority) SpawnOption { return func(c *SpawnConfig) { c.Priority = p } }
func WithSpawnName(name string) SpawnOption    { return func(c *SpawnConfig) { c.Name = name } }

// SpawnLegacyEvent is used to pass a legacy event to the framework for the
// benefits of tests that will drain the event queue until the event appears.
func WithSpawnLegacyEvent(ev Event) SpawnOption { return func(c *SpawnConfig) { c.LegacyEvent = ev } }

// Context-based spawn helpers that derive the System from the context.
// Spawn0 schedules f and returns a Promise0. Setup errors are pre-rejected.
func Spawn0[E interface{ error }](ctx context.Context, f Async0[E], opts ...SpawnOption) Promise0[E] {
	p, r := NewPromise0[E]()
	sys, ok := SystemFromContext(ctx)
	if !ok {
		r.Reject(any(ErrNoSystemInContext).(E))
		return p
	}
	// implement using taskRunner, resolving the promise on completion
	cfg := defaultSpawnConfig()
	for _, o := range opts {
		o(cfg)
	}
	tr, err := ensureTaskRunner(sys, cfg.Priority)
	if err != nil {
		r.Reject(any(fmt.Errorf("%w: %w", ErrTaskRunnerInit, err)).(E))
		return p
	}
	id := tr.nextID()
	tr.mu.Lock()
	tr.tasks[id] = func(ctx context.Context) {
		err := f(ctx)
		if any(err) != nil {
			r.Reject(err)
			return
		}
		r.Resolve()
	}
	tr.mu.Unlock()
	tr.em.Emit(ctx, asyncInvokeEvent{id: id, legacyEvent: cfg.LegacyEvent})
	// Always return a spawn-bound promise with optional start-now behavior
	sp := &spawnP0[E]{Promise0: p, sys: sys}
	if sys2, ok := sys.(*Sys); ok {
		switch sys2.executor.(type) {
		case *SingleThreadCooperativeExec, *CooperativeExec:
			sp.start = func() { tr.runNow(id) }
		}
	}
	return sp
}

// Spawn1 schedules f and returns a Promise1. Setup errors are pre-rejected.
func Spawn1[T any, E interface{ error }](ctx context.Context, f Async1[T, E], opts ...SpawnOption) Promise1[T, E] {
	p, r := NewPromise1[T, E]()
	sys, ok := SystemFromContext(ctx)
	if !ok {
		r.Reject(any(ErrNoSystemInContext).(E))
		return p
	}
	cfg := defaultSpawnConfig()
	for _, o := range opts {
		o(cfg)
	}
	tr, err := ensureTaskRunner(sys, cfg.Priority)
	if err != nil {
		r.Reject(any(fmt.Errorf("%w: %w", ErrTaskRunnerInit, err)).(E))
		return p
	}
	id := tr.nextID()
	tr.mu.Lock()
	tr.tasks[id] = func(ctx context.Context) {
		t, e := f(ctx)
		if any(e) != nil {
			r.Reject(e)
			return
		}
		r.Resolve(t)
	}
	tr.mu.Unlock()
	tr.em.Emit(ctx, asyncInvokeEvent{id: id, legacyEvent: cfg.LegacyEvent})
	sp := &spawnP1[T, E]{Promise1: p, sys: sys}
	if sys2, ok := sys.(*Sys); ok {
		switch sys2.executor.(type) {
		case *SingleThreadCooperativeExec, *CooperativeExec:
			sp.start = func() { tr.runNow(id) }
		}
	}
	return sp
}

// Spawn2 schedules f and returns a Promise2. Setup errors are pre-rejected.
func Spawn2[T1 any, T2 any, E interface{ error }](ctx context.Context, f Async2[T1, T2, E], opts ...SpawnOption) Promise2[T1, T2, E] {
	p, r := NewPromise2[T1, T2, E]()
	sys, ok := SystemFromContext(ctx)
	if !ok {
		r.Reject(any(ErrNoSystemInContext).(E))
		return p
	}
	cfg := defaultSpawnConfig()
	for _, o := range opts {
		o(cfg)
	}
	tr, err := ensureTaskRunner(sys, cfg.Priority)
	if err != nil {
		r.Reject(any(fmt.Errorf("%w: %w", ErrTaskRunnerInit, err)).(E))
		return p
	}
	id := tr.nextID()
	tr.mu.Lock()
	tr.tasks[id] = func(ctx context.Context) {
		a, b, e := f(ctx)
		if any(e) != nil {
			r.Reject(e)
			return
		}
		r.Resolve(a, b)
	}
	tr.mu.Unlock()
	tr.em.Emit(ctx, asyncInvokeEvent{id: id, legacyEvent: cfg.LegacyEvent})
	sp := &spawnP2[T1, T2, E]{Promise2: p, sys: sys}
	if sys2, ok := sys.(*Sys); ok {
		switch sys2.executor.(type) {
		case *SingleThreadCooperativeExec, *CooperativeExec:
			sp.start = func() { tr.runNow(id) }
		}
	}
	return sp
}

// spawn-bound wrappers provide start-on-first-await (for single-thread exec)
// and expose Await() that yields cooperatively using the bound System and then waits for readiness.
type spawnP0[E interface{ error }] struct {
	Promise0[E]
	sys   System
	once  sync.Once
	start func()
}

func (s *spawnP0[E]) startNow() {
	s.once.Do(func() {
		if s.start != nil {
			s.start()
		}
	})
}

func (s *spawnP0[E]) Await(ctx context.Context) error {
	// Yield only when called within executor-managed handler goroutine
	if isExecutorManaged(ctx) {
		if sys, ok := s.sys.(*Sys); ok {
			if ay, ok2 := sys.executor.(AwaitYield); ok2 {
				end := ay.OnAwaitStart()
				defer end()
			}
		}
	}
	s.startNow()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.Ready():
		return nil
	}
}

type spawnP1[T any, E interface{ error }] struct {
	Promise1[T, E]
	sys   System
	once  sync.Once
	start func()
}

func (s *spawnP1[T, E]) startNow() {
	s.once.Do(func() {
		if s.start != nil {
			s.start()
		}
	})
}

func (s *spawnP1[T, E]) Await(ctx context.Context) error {
	// Yield only when called within executor-managed handler goroutine
	if isExecutorManaged(ctx) {
		if sys, ok := s.sys.(*Sys); ok {
			if ay, ok2 := sys.executor.(AwaitYield); ok2 {
				end := ay.OnAwaitStart()
				defer end()
			}
		}
	}
	s.startNow()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.Ready():
		return nil
	}
}

type spawnP2[A any, B any, E interface{ error }] struct {
	Promise2[A, B, E]
	sys   System
	once  sync.Once
	start func()
}

func (s *spawnP2[A, B, E]) startNow() {
	s.once.Do(func() {
		if s.start != nil {
			s.start()
		}
	})
}

func (s *spawnP2[A, B, E]) Await(ctx context.Context) error {
	// Yield only when called within executor-managed handler goroutine
	if isExecutorManaged(ctx) {
		if sys, ok := s.sys.(*Sys); ok {
			if ay, ok2 := sys.executor.(AwaitYield); ok2 {
				end := ay.OnAwaitStart()
				defer end()
			}
		}
	}
	s.startNow()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.Ready():
		return nil
	}
}
