package event

import (
	"context"
	"fmt"
	"time"
)

// AwaitYield lets an executor expose cooperative yield hooks for Await* helpers.
// The function returned by OnAwaitStart must be deferred by the caller to restore state.
type AwaitYield interface {
	OnAwaitStart() func()
}

// Selector matches events.
type Selector interface {
	Matches(ev Event) bool
}

// ByType matches on Go type (using generics at call site); ByName matches by Event.String().
func ByType[T any]() Selector {
	return selectorFunc(func(ev Event) bool { _, ok := ev.(T); return ok })
}
func ByName(name string) Selector {
	return selectorFunc(func(ev Event) bool { return ev.String() == name })
}

type selectorFunc func(ev Event) bool

func (f selectorFunc) Matches(ev Event) bool { return f(ev) }

// Context key for binding a System
type sysCtxKey struct{}

// Context key indicating the call is running within an executor-managed handler goroutine
type execManagedKey struct{}

// WithSystem attaches a System to a context.
func WithSystem(ctx context.Context, sys System) context.Context {
	return context.WithValue(ctx, sysCtxKey{}, sys)
}

// SystemFromContext retrieves a System from a context.
func SystemFromContext(ctx context.Context) (System, bool) {
	v := ctx.Value(sysCtxKey{})
	if v == nil {
		return nil, false
	}
	s, ok := v.(System)
	return s, ok
}

// withExecutorManaged marks the context as being executed by an executor-managed handler goroutine.
// This is set only by the framework when invoking OnEvent handlers.
func withExecutorManaged(ctx context.Context) context.Context {
	return context.WithValue(ctx, execManagedKey{}, true)
}

// isExecutorManaged reports whether the Await caller is within an executor-managed handler path.
func isExecutorManaged(ctx context.Context) bool {
	v := ctx.Value(execManagedKey{})
	b, _ := v.(bool)
	return b
}

// NewAwaitEventPromise returns a promise that resolves with the next event of type T using the System in ctx.
// Any setup error is pre-rejected into the returned promise.
// Event awaiting helpers removed. Use explicit event waiter registration on the executor if needed.

// AwaitSleep yields cooperatively for d or until ctx is done using the System in ctx.
func AwaitSleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	if sys, ok := SystemFromContext(ctx); ok {
		if s, ok2 := sys.(*Sys); ok2 {
			if ay, ok3 := s.executor.(AwaitYield); ok3 {
				end := ay.OnAwaitStart()
				defer end()
			}
		}
	}
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// AwaitAny waits for the first of the provided promises to settle.
// Returns: index of the settled promise, its value and per-promise error, or ctx error.
func AwaitAny[T any, E interface{ error }](ctx context.Context, ps ...Promise[T, E]) (int, T, E, error) {
	var zero T
	var zeroE E
	if len(ps) == 0 {
		return -1, zero, zeroE, fmt.Errorf("no promises provided")
	}
	sys, ok := SystemFromContext(ctx)
	if !ok {
		return -1, zero, zeroE, fmt.Errorf("no system in context")
	}
	// Yield only when called from an executor-managed handler
	if isExecutorManaged(ctx) {
		if s, ok2 := sys.(*Sys); ok2 {
			if ay, ok3 := s.executor.(AwaitYield); ok3 {
				end := ay.OnAwaitStart()
				defer end()
			}
		}
	}
	type res struct {
		idx int
		v   T
		e   E
	}
	out := make(chan res, 1)
	for i, p := range ps {
		i := i
		p := p
		go func() {
			// Wait without taking the run token: use readiness channel
			select {
			case <-ctx.Done():
				return
			case <-p.Ready():
				v, e, _ := p.Result()
				select {
				case out <- res{idx: i, v: v, e: e}:
				default:
				}
			}
		}()
	}
	select {
	case <-ctx.Done():
		return -1, zero, zeroE, ctx.Err()
	case r := <-out:
		return r.idx, r.v, r.e, nil
	}
}

// AwaitAll waits for all provided promises to settle.
// Returns: slice of values and per-promise errors aligned with input order, or ctx error.
func AwaitAll[T any, E interface{ error }](ctx context.Context, ps ...Promise[T, E]) ([]T, []E, error) {
	n := len(ps)
	values := make([]T, n)
	perrs := make([]E, n)
	if n == 0 {
		return values, perrs, nil
	}
	sys, ok := SystemFromContext(ctx)
	if !ok {
		return nil, nil, fmt.Errorf("no system in context")
	}
	// Yield only when called from an executor-managed handler
	if isExecutorManaged(ctx) {
		if s, ok2 := sys.(*Sys); ok2 {
			if ay, ok3 := s.executor.(AwaitYield); ok3 {
				end := ay.OnAwaitStart()
				defer end()
			}
		}
	}
	done := make(chan struct{}, n)
	for i, p := range ps {
		i := i
		p := p
		go func() {
			// Wait on readiness only
			select {
			case <-ctx.Done():
				return
			case <-p.Ready():
				v, e, _ := p.Result()
				values[i] = v
				perrs[i] = e
				done <- struct{}{}
			}
		}()
	}
	completed := 0
	for completed < n {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-done:
			completed++
		}
	}
	return values, perrs, nil
}
