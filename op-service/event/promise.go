package event

import (
	"context"
	"sync"
)

// Promise represents a future result. It can be resolved or rejected by a Resolver.
// The error type E is explicit to match the Async arity API design.
type Promise[T any, E interface{ error }] interface {
	isPromise()
	// Ready returns a channel that is closed when the promise settles.
	Ready() <-chan struct{}
	// Result returns the settled result. If not settled, settled is false.
	Result() (T, E, bool)
	// Await waits for readiness or context cancellation.
	// Implementations returned by Spawn* yield cooperatively using the bound System before waiting.
	Await(ctx context.Context) error
}

// Resolver completes a Promise.
type Resolver[T any, E interface{ error }] interface {
	Resolve(value T)
	Reject(err E)
}

type promise1Impl[T any, E interface{ error }] struct {
	mu      sync.Mutex
	ready   chan struct{}
	settled bool
	val     T
	err     E
}

func NewPromise1[T any, E interface{ error }]() (Promise[T, E], Resolver[T, E]) {
	p := &promise1Impl[T, E]{
		ready: make(chan struct{}),
	}
	return p, p
}

func (p *promise1Impl[T, E]) isPromise() {}

func (p *promise1Impl[T, E]) Ready() <-chan struct{} { return p.ready }

func (p *promise1Impl[T, E]) Result() (T, E, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.val, p.err, p.settled
}

func (p *promise1Impl[T, E]) Await(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.ready:
		return nil
	}
}

func (p *promise1Impl[T, E]) Resolve(value T) {
	p.mu.Lock()
	if p.settled {
		p.mu.Unlock()
		return
	}
	p.settled = true
	p.val = value
	close(p.ready)
	p.mu.Unlock()
}

func (p *promise1Impl[T, E]) Reject(err E) {
	p.mu.Lock()
	if p.settled {
		p.mu.Unlock()
		return
	}
	p.settled = true
	p.err = err
	close(p.ready)
	p.mu.Unlock()
}

// legacy AwaitPromise* helpers removed in favor of Promise.Await()

// Promise0 represents a future completion with only an error result.
type Promise0[E interface{ error }] interface {
	isPromise0()
	Ready() <-chan struct{}
	Result() (E, bool)
	Await(ctx context.Context) error
}

// Resolver0 completes a Promise0.
type Resolver0[E interface{ error }] interface {
	Resolve()
	Reject(err E)
}

type promise0Impl[E interface{ error }] struct {
	mu      sync.Mutex
	ready   chan struct{}
	settled bool
	err     E
}

func NewPromise0[E interface{ error }]() (Promise0[E], Resolver0[E]) {
	p := &promise0Impl[E]{
		ready: make(chan struct{}),
	}
	return p, p
}

func (p *promise0Impl[E]) isPromise0()            {}
func (p *promise0Impl[E]) Ready() <-chan struct{} { return p.ready }
func (p *promise0Impl[E]) Result() (E, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.err, p.settled
}
func (p *promise0Impl[E]) Await(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.ready:
		return nil
	}
}
func (p *promise0Impl[E]) Resolve() {
	p.mu.Lock()
	if p.settled {
		p.mu.Unlock()
		return
	}
	p.settled = true
	close(p.ready)
	p.mu.Unlock()
}
func (p *promise0Impl[E]) Reject(err E) {
	p.mu.Lock()
	if p.settled {
		p.mu.Unlock()
		return
	}
	p.settled = true
	p.err = err
	close(p.ready)
	p.mu.Unlock()
}

// Promise1 mirrors Promise with arity-1 without relying on type-alias type params.
type Promise1[T any, E interface{ error }] interface {
	isPromise()
	Ready() <-chan struct{}
	Result() (T, E, bool)
	Await(ctx context.Context) error
}

// Resolver1 completes a Promise1.
type Resolver1[T any, E interface{ error }] interface {
	Resolve(value T)
	Reject(err E)
}

// Promise2 represents a future result of two values and an error.
type Promise2[A any, B any, E interface{ error }] interface {
	isPromise2()
	Ready() <-chan struct{}
	Result() (A, B, E, bool)
	Await(ctx context.Context) error
}

// Resolver2 completes a Promise2.
type Resolver2[A any, B any, E interface{ error }] interface {
	Resolve(a A, b B)
	Reject(err E)
}

type promise2Impl[A any, B any, E interface{ error }] struct {
	mu      sync.Mutex
	ready   chan struct{}
	settled bool
	a       A
	b       B
	err     E
}

func NewPromise2[A any, B any, E interface{ error }]() (Promise2[A, B, E], Resolver2[A, B, E]) {
	p := &promise2Impl[A, B, E]{
		ready: make(chan struct{}),
	}
	return p, p
}

func (p *promise2Impl[A, B, E]) isPromise2()            {}
func (p *promise2Impl[A, B, E]) Ready() <-chan struct{} { return p.ready }
func (p *promise2Impl[A, B, E]) Result() (A, B, E, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.a, p.b, p.err, p.settled
}
func (p *promise2Impl[A, B, E]) Await(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-p.ready:
		return nil
	}
}
func (p *promise2Impl[A, B, E]) Resolve(a A, b B) {
	p.mu.Lock()
	if p.settled {
		p.mu.Unlock()
		return
	}
	p.settled = true
	p.a = a
	p.b = b
	close(p.ready)
	p.mu.Unlock()
}
func (p *promise2Impl[A, B, E]) Reject(err E) {
	p.mu.Lock()
	if p.settled {
		p.mu.Unlock()
		return
	}
	p.settled = true
	p.err = err
	close(p.ready)
	p.mu.Unlock()
}

// legacy helpers removed: AwaitPromise0, AwaitPromise1, AwaitPromise2
