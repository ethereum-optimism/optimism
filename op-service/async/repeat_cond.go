package async

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type RepeatCond struct {
	lifeCtx    context.Context
	lifeCancel context.CancelFunc

	closeCtx    context.Context
	closeCancel context.CancelCauseFunc

	started atomic.Bool

	cond        sync.Cond
	conditional func() bool
	effect      func()
}

func NewRepeatCond(ctx context.Context, locker sync.Locker, conditional func() bool, effect func()) *RepeatCond {
	lifeCtx, lifeCancel := context.WithCancel(ctx)
	closeCtx, closeCancel := context.WithCancelCause(context.Background())

	return &RepeatCond{
		lifeCtx:     lifeCtx,
		lifeCancel:  lifeCancel,
		closeCtx:    closeCtx,
		closeCancel: closeCancel,
		cond:        sync.Cond{L: locker},
		conditional: conditional,
		effect:      effect,
	}
}

func (r *RepeatCond) Signal() {
	r.cond.Signal()
}

func (r *RepeatCond) Start() {
	// check if already started.
	if !r.started.CompareAndSwap(false, true) {
		return
	}

	// signal upon lifetime ctx completion, so we can awake to detect the ctx.Err() != nil
	go func() {
		<-r.lifeCtx.Done()
		r.cond.Signal()
	}()

	go func() {
		defer func() {
			if err := recover(); err != nil {
				r.closeCancel(fmt.Errorf("closed with panic: %v", err))
			}
		}()

		// By default, assume a locked position.
		// The Wait() will unlock it, allowing other resources to access the resource while it's waiting.
		r.cond.L.Lock()

		// repeat the condition over and over again
		for {
			for {
				if r.lifeCtx.Err() != nil { // when signaled, first check if it's time to exit
					r.closeCancel(r.lifeCtx.Err())
				}
				if r.conditional() { // stop waiting if we hit our condition
					break
				}
				// This unlocks the lock upon calling, so others can use it.
				// It re-locks just before the call completes.
				r.cond.Wait()
				// And when Signaled, it locks it again,
				// so we can proceed to do our processing without another routine using our resource.
			}
			r.effect()
		}
	}()
}

// Ctx returns a context that is canceled when the RepeatCond is fully stopped,
// either upon parent-lifetime context cancellation or with a cause on process panic.
func (r *RepeatCond) Ctx() context.Context {
	return r.closeCtx
}

// Note: we tend to use this with many other repeat-conditions,
// and closing it through a shared context reduces boilerplate,
// especially if we monitor the condition Ctx for errors anyway.
// If we need to, we can later add an optional Close() method
// that calls lifeCancel() and then awaits the closeCtx for convenience.
