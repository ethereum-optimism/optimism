package loadtest

import (
	"context"
	"errors"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/accounting"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/core"
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

type AIMDObserver interface {
	UpdateRPS(uint64)
}

type NoOpAIMDObserver struct{}

var _ AIMDObserver = NoOpAIMDObserver{}

func (NoOpAIMDObserver) UpdateRPS(uint64) {}

type aimdConfig struct {
	increaseDelta     uint64       // additive delta
	decreaseFactor    float64      // multiplicative factor
	failRateThreshold float64      // when to start decreasing (e.g., 0.05 of all requests are failures)
	adjustWindow      uint64       // how many operations to perform before adjusting rps
	observer          AIMDObserver // callback interface for metrics and logging
}

func NewAIMD(baseRPS uint64, slotTime time.Duration, opts ...AIMDOption) *AIMD {
	cfg := &aimdConfig{
		increaseDelta:     max(baseRPS/10, 1),
		decreaseFactor:    0.5,
		failRateThreshold: 0.05,
		adjustWindow:      50,
		observer:          NoOpAIMDObserver{},
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
	aimd.cfg.observer.UpdateRPS(baseRPS)
	return aimd
}

type AIMDOption func(*aimdConfig)

func WithAIMDOptsCombined(opts ...AIMDOption) AIMDOption {
	return func(cfg *aimdConfig) {
		for _, opt := range opts {
			opt(cfg)
		}
	}
}

func WithAIMDObserver(observer AIMDObserver) AIMDOption {
	return func(cfg *aimdConfig) {
		cfg.observer = observer
	}
}

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
	c.cfg.observer.UpdateRPS(newRPS)
	c.metrics = aimdMetrics{}
}

func (c *AIMD) Ready() <-chan struct{} {
	return c.ready
}

// Spammer spams. Every invocation of Spam puts additional load on a system. Despite taking a
// devtest.T, implementations should return virtually all errors so the Controller can adjust
// spamming accordingly (very few errors are fatal in a load test).
type Spammer interface {
	Spam(devtest.T) error
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

// Run will spam until the budget is depleted before exiting successfully.
func (b *Burst) Run(t devtest.T, spammer Spammer) {
	ctx, cancel := context.WithCancel(t.Ctx())
	defer cancel()
	t = t.WithCtx(ctx)

	aimd := setupAIMD(t, b.blockTime, b.opts...)

	var wg sync.WaitGroup
	defer wg.Wait()
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
			t.Logger().Warn("Spammer error", "err", err)
			aimd.Adjust(false)
		}()
	}
}

type InfoByLabel interface {
	InfoByLabel(ctx context.Context, label eth.BlockLabel) (eth.BlockInfo, error)
}

type Steady struct {
	elasticityMultiplier uint64
	blockTime            time.Duration
	el                   InfoByLabel
	opts                 []AIMDOption
}

var _ Schedule = (*Steady)(nil)

func NewSteady(el InfoByLabel, elasticityMultiplier uint64, blockTime time.Duration, opts ...AIMDOption) *Steady {
	return &Steady{
		el:                   el,
		elasticityMultiplier: elasticityMultiplier,
		blockTime:            blockTime,
		opts:                 opts,
	}
}

// Run will spam just enough to keep the network within 95%-100% of the gas target. It exists
// successfully upon NAT_STEADY_TIMEOUT.
func (s *Steady) Run(t devtest.T, spammer Spammer) {
	// Configure a context that will allow us to exit the test on time. We set the following
	// deadlines/timeouts on the context and let the context package choose the minimum:
	//
	// 1. Test context deadline (minus 10s for cleanup), if it exists.
	// 2. NAT_STEADY_TIMEOUT or 3m if it doesn't exist.
	ctx, cancel := context.WithCancel(t.Ctx())
	t.Cleanup(cancel)
	if deadline, exists := ctx.Deadline(); exists {
		ctx, cancel = context.WithDeadline(ctx, deadline.Add(-10*time.Second))
		t.Cleanup(cancel)
	}
	if timeoutStr, exists := os.LookupEnv("NAT_STEADY_TIMEOUT"); exists {
		timeout, err := time.ParseDuration(timeoutStr)
		t.Require().NoError(err)
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithTimeout(ctx, 3*time.Minute)
	}
	t = t.WithCtx(ctx)
	t.Cleanup(cancel)

	// The backpressure algorithm will adjust every slot to stay within 95-100% of the gas target.
	aimd := setupAIMD(t, s.blockTime, WithAIMDOptsCombined(s.opts...), WithAdjustWindow(1), WithDecreaseFactor(0.95))
	var wg sync.WaitGroup
	t.Cleanup(wg.Wait)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.blockTime):
				unsafe, err := s.el.InfoByLabel(ctx, eth.Unsafe)
				if err != nil {
					if errors.Is(err, context.DeadlineExceeded) {
						return
					}
					t.Require().NoError(err)
				}
				gasTarget := unsafe.GasLimit() / s.elasticityMultiplier
				// Apply backpressure when we meet or exceed the gas target.
				aimd.Adjust(unsafe.GasUsed() < gasTarget)
			}
		}
	}()

	for range aimd.Ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := spammer.Spam(t)
			if err == nil {
				return
			}
			if isOverdraftErr(err) {
				cancel()
				t.Require().NoError(err)
			}
			t.Logger().Warn("Spammer error", "err", err)
		}()
	}
}

func setupAIMD(t devtest.T, blockTime time.Duration, aimdOpts ...AIMDOption) *AIMD {
	targetMessagePassesPerBlock := uint64(100)
	if targetMsgPassesStr, exists := os.LookupEnv("NAT_INTEROP_LOADTEST_TARGET"); exists {
		var err error
		targetMessagePassesPerBlock, err = strconv.ParseUint(targetMsgPassesStr, 10, 0)
		t.Require().NoError(err)
	}
	aimd := NewAIMD(targetMessagePassesPerBlock, blockTime, aimdOpts...)
	var wg sync.WaitGroup
	t.Cleanup(wg.Wait)
	wg.Add(1)
	go func() {
		defer wg.Done()
		aimd.Start(t.Ctx())
	}()
	return aimd
}

// TODO(16536): sometimes the onchain budget depletes before the offchain budget. It would be good to
// understand why that happens.
func isOverdraftErr(err error) bool {
	var overdraft *accounting.OverdraftError
	return errors.As(err, &overdraft) ||
		errors.Is(err, core.ErrInsufficientFunds) ||
		errors.Is(err, core.ErrInsufficientFundsForTransfer) ||
		errors.Is(err, core.ErrInsufficientBalanceWitness)
}
