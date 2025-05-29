package loadtest

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// AIMD scheduler (additive-increase, multiplicative-decrease).
type AIMD struct {
	// rps can be thought of to mean "requests per slot", although the unit and quantity are flexible.
	rps    atomic.Uint64
	maxRPS uint64

	increaseDelta  uint64  // additive delta
	decreaseFactor float64 // multiplicative factor

	failRateThreshold float64 // when to start decreasing (e.g., 0.05 of all requests are failures)

	adjustWindow uint64 // how many operations to perform before adjusting rps

	metricsMu sync.Mutex
	metrics   aimdMetrics

	slotTime time.Duration
	ready    chan struct{}
}

type aimdMetrics struct {
	Completed uint64
	Failed    uint64
}

func NewAIMD(baseRPS uint64, slotTime time.Duration, opts ...AIMDOption) *AIMD {
	aimd := &AIMD{
		maxRPS:            10 * baseRPS,
		increaseDelta:     max(baseRPS/10, 1),
		decreaseFactor:    0.5,
		failRateThreshold: 0.05,
		ready:             make(chan struct{}),
		slotTime:          slotTime,
		adjustWindow:      50,
		metrics:           aimdMetrics{},
	}
	aimd.rps.Store(baseRPS)
	for _, opt := range opts {
		opt(aimd)
	}
	return aimd
}

type AIMDOption func(*AIMD)

func WithMaxRPS(maxRPS uint64) AIMDOption {
	return func(aimd *AIMD) {
		aimd.maxRPS = maxRPS
	}
}

func WithIncreaseDelta(delta uint64) AIMDOption {
	return func(aimd *AIMD) {
		aimd.increaseDelta = delta
	}
}

func WithDecreaseFactor(factor float64) AIMDOption {
	return func(aimd *AIMD) {
		aimd.decreaseFactor = factor
	}
}

func WithFailRateThreshold(threshold float64) AIMDOption {
	return func(aimd *AIMD) {
		aimd.failRateThreshold = threshold
	}
}

func WithAdjustWindow(window uint64) AIMDOption {
	return func(aimd *AIMD) {
		aimd.adjustWindow = window
	}
}

func (c *AIMD) Start(ctx context.Context) {
	defer close(c.ready)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(c.slotTime / time.Duration(c.rps.Load())):
			select {
			case c.ready <- struct{}{}:
			default: // Skip if readers are not ready.
			}
		}
	}
}

func (c *AIMD) Adjust(success bool) {
	c.metricsMu.Lock()
	defer c.metricsMu.Unlock()
	c.metrics.Completed++
	if !success {
		c.metrics.Failed++
	}
	if c.metrics.Completed == c.adjustWindow {
		failRate := float64(c.metrics.Failed) / float64(c.metrics.Completed+1)
		if failRate > c.failRateThreshold {
			newRPS := max(uint64(float64(c.rps.Load())*c.decreaseFactor), 1)
			c.rps.Store(newRPS)
		} else {
			newRPS := min(c.rps.Load()+c.increaseDelta, c.maxRPS)
			c.rps.Store(newRPS)
		}
		c.metrics = aimdMetrics{}
	}
}

func (c *AIMD) Ready() <-chan struct{} {
	return c.ready
}
