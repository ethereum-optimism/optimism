package event

import (
	"context"

	"golang.org/x/time/rate"
)

type Limiter[E Emitter] struct {
	ctx       context.Context
	emitter   E
	rl        *rate.Limiter
	onLimited func()
}

// NewLimiter returns an event rate-limiter.
// This can be used to prevent event loops from (accidentally) running too hot.
// The eventRate is the number of events per second.
// The eventBurst is the margin of events to eat into until the rate-limit kicks in.
// The onLimited function is optional, and will be called if an emitted event is getting rate-limited
func NewLimiter[E Emitter](ctx context.Context, em E, eventRate rate.Limit, eventBurst int, onLimited func()) *Limiter[E] {
	return &Limiter[E]{
		ctx:       ctx,
		emitter:   em,
		rl:        rate.NewLimiter(eventRate, eventBurst),
		onLimited: onLimited,
	}
}

// Emit is thread-safe, multiple parallel derivers can safely emit events to it.
func (l *Limiter[E]) Emit(ev Event) {
	if l.onLimited != nil && l.rl.Tokens() < 1.0 {
		l.onLimited()
	}
	if err := l.rl.Wait(l.ctx); err != nil {
		return // ctx error, safe to ignore.
	}
	l.emitter.Emit(ev)
}

// LimiterDrainer is a variant of Limiter that supports event draining.
type LimiterDrainer Limiter[EmitterDrainer]

func NewLimiterDrainer(ctx context.Context, em EmitterDrainer, eventRate rate.Limit, eventBurst int, onLimited func()) *LimiterDrainer {
	return (*LimiterDrainer)(NewLimiter(ctx, em, eventRate, eventBurst, onLimited))
}

func (l *LimiterDrainer) Emit(ev Event) {
	l.emitter.Emit(ev)
}

func (l *LimiterDrainer) Drain() error {
	return l.emitter.Drain()
}

func (l *LimiterDrainer) DrainUntil(fn func(ev Event) bool, excl bool) error {
	return l.emitter.DrainUntil(fn, excl)
}
