package loadtest

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum/go-ethereum/core"
)

// aimd parameters. Only the base rate is configurable, through WithBaseRPS.
const (
	defaultBaseRPS    = 100                 // initial invocations per slot
	increaseDelta     = defaultBaseRPS / 10 // additive increase
	decreaseFactor    = 0.5                 // multiplicative decrease
	failRateThreshold = 0.05                // fail rate above which the rate decreases
	adjustWindow      = 50                  // invocations between rate adjustments
)

// aimd is an additive-increase, multiplicative-decrease scheduler.
type aimd struct {
	// rps can be thought of to mean "requests per slot", although the unit and quantity are
	// flexible.
	rps atomic.Uint64

	metricsMu sync.Mutex
	metrics   aimdMetrics

	slotTime time.Duration
	ready    chan struct{}
}

type aimdMetrics struct {
	Completed uint64
	Failed    uint64
}

type aimdConfig struct {
	baseRPS uint64
}

func newAIMD(slotTime time.Duration, opts ...AIMDOption) *aimd {
	cfg := &aimdConfig{baseRPS: defaultBaseRPS}
	for _, opt := range opts {
		opt(cfg)
	}
	a := &aimd{
		ready:    make(chan struct{}),
		slotTime: slotTime,
	}
	a.rps.Store(cfg.baseRPS)
	return a
}

type AIMDOption func(*aimdConfig)

func WithBaseRPS(rps uint64) AIMDOption {
	return func(cfg *aimdConfig) {
		cfg.baseRPS = rps
	}
}

func (c *aimd) Start(ctx context.Context) {
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

func (c *aimd) Adjust(success bool) {
	c.metricsMu.Lock()
	defer c.metricsMu.Unlock()
	c.metrics.Completed++
	if !success {
		c.metrics.Failed++
	}
	if c.metrics.Completed != adjustWindow {
		return
	}
	failRate := float64(c.metrics.Failed) / float64(c.metrics.Completed)
	var newRPS uint64
	if failRate > failRateThreshold {
		newRPS = max(uint64(float64(c.rps.Load())*decreaseFactor), 1)
	} else {
		newRPS = c.rps.Load() + increaseDelta
	}
	c.rps.Store(newRPS)
	c.metrics = aimdMetrics{}
}

func (c *aimd) Ready() <-chan struct{} {
	return c.ready
}

// Spammer spams. Every invocation of Spam puts additional load on a system. Despite taking a
// devtest.T, implementations should return virtually all errors so the schedule can adjust its
// rate (very few errors are fatal in a load test).
type Spammer interface {
	Spam(devtest.T) error
}

type SpammerFunc func(t devtest.T) error

func (s SpammerFunc) Spam(t devtest.T) error {
	return s(t)
}

// Schedule schedules a Spammer. It determines how often to spam and when to stop.
type Schedule interface {
	Run(devtest.T, Spammer)
}

type Burst struct {
	blockTime time.Duration
	opts      []AIMDOption
}

var _ Schedule = (*Burst)(nil)

func NewBurst(blockTime time.Duration, opts ...AIMDOption) *Burst {
	return &Burst{
		blockTime: blockTime,
		opts:      opts,
	}
}

// Run executes spammer.Spam with increasing throughput. It decreases throughput when errors are
// encountered.
func (b *Burst) Run(t devtest.T, spammer Spammer) {
	ctx, cancel := context.WithCancel(t.Ctx())
	t = t.WithCtx(ctx)

	aimd := setupAIMD(t, b.blockTime, b.opts...)

	var wg sync.WaitGroup
	defer func() {
		cancel()
		wg.Wait()
	}()
	for range aimd.Ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := spammer.Spam(t)
			if err == nil {
				aimd.Adjust(true)
				return
			}
			if isOverdraftErr(err) {
				cancel()
			}
			logOnError(t, err)
			aimd.Adjust(false)
		}()
	}
}

type Constant struct {
	blockTime time.Duration
	aimdOpts  []AIMDOption
}

var _ Schedule = (*Constant)(nil)

func NewConstant(blockTime time.Duration, aimdOpts ...AIMDOption) *Constant {
	return &Constant{
		blockTime: blockTime,
		aimdOpts:  aimdOpts,
	}
}

func (c *Constant) Run(t devtest.T, spammer Spammer) {
	aimd := setupAIMD(t, c.blockTime, c.aimdOpts...)
	var wg sync.WaitGroup
	defer wg.Wait()
	for range aimd.Ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			logOnError(t, spammer.Spam(t))
		}()
	}
}

func logOnError(t devtest.T, err error) {
	if errors.Is(err, context.Canceled) {
		// Context cancelation is typically caused by the test ending, which is not really a
		// spammer error. Don't spam warnings in that case.
		t.Logger().Debug("Spammer error", "err", err)
	} else if err != nil {
		t.Logger().Warn("Spammer error", "err", err)
	}
}

func setupAIMD(t devtest.T, blockTime time.Duration, aimdOpts ...AIMDOption) *aimd {
	a := newAIMD(blockTime, aimdOpts...)
	ctx, cancel := context.WithCancel(t.Ctx())
	var wg sync.WaitGroup
	t.Cleanup(func() {
		cancel()
		wg.Wait()
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		a.Start(ctx)
	}()
	return a
}

// isOverdraftErr reports whether the spammer ran out of funds, which ends a Burst.
func isOverdraftErr(err error) bool {
	return errors.Is(err, core.ErrInsufficientFunds) ||
		errors.Is(err, core.ErrInsufficientFundsForTransfer)
}
