package loadtest

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// AIMD scheduler (additive-increase, multiplicative-decrease).
type AIMD struct {
	// rps can be thought of to mean "requests per slot", although the unit and quantity are
	// flexible.
	rps atomic.Uint64

	metricsMu sync.Mutex
	metrics   aimdMetrics

	cfg *aimdConfig

	slotTime time.Duration
	ready    chan struct{}
}

type aimdMetrics struct {
	Completed uint64
	Failed    uint64
}

type aimdConfig struct {
	increaseDelta     uint64  // additive delta
	decreaseFactor    float64 // multiplicative factor
	failRateThreshold float64 // when to start decreasing (e.g., 0.05 of all requests are failures)
	adjustWindow      uint64  // how many operations to perform before adjusting rps
}

func NewAIMD(baseRPS uint64, slotTime time.Duration, opts ...AIMDOption) *AIMD {
	cfg := &aimdConfig{
		increaseDelta:     max(baseRPS/10, 1),
		decreaseFactor:    0.5,
		failRateThreshold: 0.05,
		adjustWindow:      50,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	aimd := &AIMD{
		ready:    make(chan struct{}),
		slotTime: slotTime,
		metrics:  aimdMetrics{},
		cfg:      cfg,
	}
	aimd.rps.Store(baseRPS)
	targetMessagesPerBlock.Set(float64(baseRPS))
	return aimd
}

type AIMDOption func(*aimdConfig)

func WithIncreaseDelta(delta uint64) AIMDOption {
	return func(cfg *aimdConfig) {
		cfg.increaseDelta = delta
	}
}

func WithDecreaseFactor(factor float64) AIMDOption {
	return func(cfg *aimdConfig) {
		cfg.decreaseFactor = factor
	}
}

func WithFailRateThreshold(threshold float64) AIMDOption {
	return func(cfg *aimdConfig) {
		cfg.failRateThreshold = threshold
	}
}

func WithAdjustWindow(window uint64) AIMDOption {
	return func(cfg *aimdConfig) {
		cfg.adjustWindow = window
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
	if c.metrics.Completed != c.cfg.adjustWindow {
		return
	}
	failRate := float64(c.metrics.Failed) / float64(c.metrics.Completed)
	var newRPS uint64
	if failRate > c.cfg.failRateThreshold {
		newRPS = max(uint64(float64(c.rps.Load())*c.cfg.decreaseFactor), 1)
	} else {
		newRPS = c.rps.Load() + c.cfg.increaseDelta
	}
	c.rps.Store(newRPS)
	targetMessagesPerBlock.Set(float64(newRPS))
	c.metrics = aimdMetrics{}
}

func (c *AIMD) Ready() <-chan struct{} {
	return c.ready
}
