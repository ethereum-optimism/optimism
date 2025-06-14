package controller

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-service/retry"
)

var defaultBackoff = &retry.ExponentialStrategy{
	Min:       time.Second,
	Max:       time.Second * 10,
	MaxJitter: time.Second / 10,
}

type BackoffProvider interface {
	ResetBackoff()
	SetStrategy(strategy retry.Strategy)
	DoBackoff(err error, now time.Time)
	IsBackedOff(now time.Time) bool
}

type backoffState struct {
	strategy retry.Strategy
	attempt  int
	lastErr  error
	lastTime time.Time
	nextTime time.Time
}

var _ BackoffProvider = (*backoffState)(nil)

func (b *backoffState) ResetBackoff() {
	b.attempt = 0
	b.lastTime = time.Time{}
	b.nextTime = time.Time{}
	b.lastErr = nil
}

func (b *backoffState) SetStrategy(strategy retry.Strategy) {
	b.strategy = strategy
}

func (b *backoffState) DoBackoff(err error, now time.Time) {
	if b.strategy == nil {
		b.strategy = defaultBackoff
	}
	b.lastErr = err
	b.lastTime = now
	wait := b.strategy.Duration(b.attempt)
	b.nextTime = b.lastTime.Add(wait)
	b.attempt += 1
}

func (b *backoffState) IsBackedOff(now time.Time) bool {
	if b.attempt == 0 {
		return false
	}
	return now.Before(b.nextTime)
}

func NotBackedOff[V BackoffProvider](now time.Time) Predicate[V] {
	return func(v V) bool {
		return v.IsBackedOff(now)
	}
}
