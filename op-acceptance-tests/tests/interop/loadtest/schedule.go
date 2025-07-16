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
	baseRPS           uint64
	increaseDelta     uint64       // additive delta
	decreaseFactor    float64      // multiplicative factor
	failRateThreshold float64      // when to start decreasing (e.g., 0.05 of all requests are failures)
	adjustWindow      uint64       // how many operations to perform before adjusting rps
	observer          AIMDObserver // callback interface for metrics and logging
}

func NewAIMD(baseRPS uint64, slotTime time.Duration, opts ...AIMDOption) *AIMD {
	cfg := &aimdConfig{
		baseRPS:           baseRPS,
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
	aimd.rps.Store(cfg.baseRPS)
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

func WithBaseRPS(rps uint64) AIMDOption {
	return func(cfg *aimdConfig) {
		cfg.baseRPS = rps
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

// Run executes spammer.Spam with increasing throughput. It decreases throughput when errors are
// encountered.
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

// Run will spam just enough to keep the network within 95%-100% of the gas target.
func (s *Steady) Run(t devtest.T, spammer Spammer) {
	ctx, cancel := context.WithCancel(t.Ctx())
	defer cancel()
	t = t.WithCtx(ctx)

	// The backpressure algorithm will adjust every slot to stay within 95-100% of the gas target.
	aimd := setupAIMD(t, s.blockTime, WithAIMDOptsCombined(s.opts...), WithAdjustWindow(1), WithDecreaseFactor(0.95))
	var wg sync.WaitGroup
	t.Cleanup(wg.Wait)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-t.Ctx().Done():
				return
			case <-time.After(s.blockTime):
				unsafe, err := s.el.InfoByLabel(t.Ctx(), eth.Unsafe)
				if err != nil {
					if errors.Is(err, t.Ctx().Err()) {
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
			}
			t.Logger().Warn("Spammer error", "err", err)
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
			if err := spammer.Spam(t); err != nil {
				t.Logger().Warn("Spammer error", "err", err)
			}
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
