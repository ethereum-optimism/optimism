package karsttest

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"golang.org/x/sync/errgroup"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/txinclude"
	"github.com/ethereum-optimism/optimism/op-service/txplan"
)

// Spam parameters shared with the EIP-7934 block-size acceptance test (runSpam
// in op-acceptance-tests/tests/osaka_on_l2_test.go). The CLI spam command and
// that test flood L2 with the same transaction shape; keeping the values here
// lets both sides reference one source of truth so they can never drift.
const (
	// SpamCalldataSize is the per-transaction calldata size. op-geth and op-reth
	// cap mempool txs at 128 kB; we leave an 8 kB buffer for the tx fields
	// outside the calldata.
	SpamCalldataSize = 120 * 1024

	// SpamGasLimit is the per-transaction gas limit. EIP-7623 charges 10 gas per
	// calldata byte, so 120 kB needs ~1.2M gas; 1.25M leaves headroom.
	SpamGasLimit = 1_250_000

	// spamMaxConcurrentTxs mirrors reth's per-account mempool cap of 16: more than
	// this many in-flight txs from one account are rejected, so each spammer EOA
	// caps its own concurrency here. This is why the spammer fans out across many
	// accounts rather than blasting from one.
	spamMaxConcurrentTxs = 16

	// AIMD tuning, matching loadtest.NewAIMD's defaults (the scheduler the
	// acceptance test drives via loadtest.NewBurst).
	spamDecreaseFactor    = 0.5
	spamFailRateThreshold = 0.05
	spamAdjustWindow      = 50
)

// SpamConfig configures SpamCalldata.
type SpamConfig struct {
	// NumAccounts is how many ephemeral EOAs to fund and round-robin across.
	NumAccounts uint64
	// FundPerAccount is the value transferred to each ephemeral EOA up front.
	FundPerAccount eth.ETH
	// BaseRPS is the burst scheduler's starting send rate per block slot.
	BaseRPS uint64
	// To is the destination of every calldata-heavy tx. Any address works; the
	// target only needs to absorb the calldata.
	To common.Address
	// BlockTime paces both the burst scheduler and the reliable-inclusion retry
	// loop. Set it to the L2 block time.
	BlockTime time.Duration
}

// SpamCalldata floods the L2 with calldata-heavy transactions until ctx is
// cancelled (e.g. via Ctrl+C). It funds cfg.NumAccounts ephemeral EOAs from
// funderKey, then round-robins SpamCalldataSize-byte transactions across them on
// an additive-increase/multiplicative-decrease burst schedule. This mirrors the
// EIP-7934 block-size acceptance test, which spams L2 the same way to push block
// data past the (disabled) max-block-size limit and drive the base fee up.
//
// It uses the same txplan + txinclude primitives as the acceptance test's
// loadtest layer, but is self-contained so the CLI need not depend on the
// devstack test harness.
func SpamCalldata(ctx context.Context, logger log.Logger, cl *ethclient.Client, funderKey *ecdsa.PrivateKey, cfg SpamConfig) error {
	if cfg.NumAccounts == 0 {
		return errors.New("spam: need at least one account")
	}
	if cfg.BaseRPS == 0 {
		return errors.New("spam: rps must be positive")
	}
	if cfg.BlockTime <= 0 {
		return errors.New("spam: block time must be positive")
	}

	chainID, err := cl.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("fetch chain id: %w", err)
	}
	// txinclude assumes a reliable EL: SendTransaction retries transient submit
	// failures and TransactionReceipt keeps polling until a receipt exists.
	el := txinclude.NewReliableEL(cl, cfg.BlockTime)

	eoas, err := fundSpamEOAs(ctx, logger, cl, funderKey, chainID, el, cfg)
	if err != nil {
		return err
	}

	logger.Info("starting calldata spam (until interrupted)",
		"accounts", len(eoas), "baseRPS", cfg.BaseRPS, "to", cfg.To,
		"calldataBytes", SpamCalldataSize, "gasLimit", SpamGasLimit)
	return runBurst(ctx, logger, eoas, cfg)
}

// fundSpamEOAs creates cfg.NumAccounts fresh EOAs and funds each with
// cfg.FundPerAccount from funderKey. Funding runs concurrently; the funder's
// own txinclude.Limit caps in-flight funding txs at spamMaxConcurrentTxs and its
// nonce manager assigns sequential nonces.
func fundSpamEOAs(ctx context.Context, logger log.Logger, cl *ethclient.Client, funderKey *ecdsa.PrivateKey, chainID *big.Int, el txinclude.EL, cfg SpamConfig) ([]*spamEOA, error) {
	funderAddr := crypto.PubkeyToAddress(funderKey.PublicKey)
	funderNonce, err := cl.PendingNonceAt(ctx, funderAddr)
	if err != nil {
		return nil, fmt.Errorf("fetch funder nonce: %w", err)
	}
	funder := newSpamEOA(cl, funderKey, chainID, el, funderNonce)

	logger.Info("funding spam accounts", "count", cfg.NumAccounts, "weiEach", cfg.FundPerAccount, "funder", funderAddr)
	eoas := make([]*spamEOA, cfg.NumAccounts)
	g, gctx := errgroup.WithContext(ctx)
	for i := range eoas {
		g.Go(func() error {
			key, err := crypto.GenerateKey()
			if err != nil {
				return fmt.Errorf("generate spam key: %w", err)
			}
			addr := crypto.PubkeyToAddress(key.PublicKey)
			if _, err := funder.include(gctx,
				txplan.WithTo(&addr),
				txplan.WithValue(cfg.FundPerAccount),
				txplan.WithGasLimit(params.TxGas),
			); err != nil {
				return fmt.Errorf("fund spam account %s: %w", addr, err)
			}
			// Fresh EOAs start at nonce 0, the txinclude default.
			eoas[i] = newSpamEOA(cl, key, chainID, el, 0)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	logger.Info("funded spam accounts", "count", len(eoas))
	return eoas, nil
}

// runBurst round-robins calldata-heavy txs across eoas on an AIMD burst
// schedule until ctx is cancelled. It also stops early if an account runs out
// of funds, since further sends from it can only fail.
func runBurst(ctx context.Context, logger log.Logger, eoas []*spamEOA, cfg SpamConfig) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	rr := newRoundRobin(eoas)
	a := newAIMD(cfg.BaseRPS, cfg.BlockTime)
	// All-zero calldata, shared read-only across goroutines.
	calldata := make([]byte, SpamCalldataSize)

	var sent, failed atomic.Uint64
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		a.start(ctx)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				logger.Info("spamming L2 with calldata-heavy txs",
					"rps", a.rps.Load(), "sent", sent.Load(), "failed", failed.Load())
			}
		}
	}()

	// Drain in-flight work before returning. The ready channel closes when
	// a.start observes ctx cancellation, which ends the loop below; no new
	// goroutines are spawned after that, so wg.Add never races wg.Wait.
	defer func() {
		cancel()
		wg.Wait()
	}()

	for range a.ready() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := rr.get().include(ctx,
				txplan.WithTo(&cfg.To),
				txplan.WithData(calldata),
				txplan.WithGasLimit(SpamGasLimit),
			)
			if err == nil {
				sent.Add(1)
				a.adjust(true)
				return
			}
			failed.Add(1)
			a.adjust(false)
			switch {
			case errors.Is(err, context.Canceled):
				// Shutting down; not a real failure.
			case isInsufficientFundsErr(err):
				logger.Warn("spam account out of funds; stopping", "err", err)
				cancel()
			default:
				logger.Warn("spam tx failed", "err", err)
			}
		}()
	}

	if err := ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func isInsufficientFundsErr(err error) bool {
	return errors.Is(err, core.ErrInsufficientFunds) ||
		errors.Is(err, core.ErrInsufficientFundsForTransfer)
}

// spamEOA is one funded sender: a txplan base plan paired with a reliable,
// nonce-managing, concurrency-limited includer. It mirrors loadtest.SyncEOA
// without the devstack test dependency.
type spamEOA struct {
	plan     txplan.Option
	includer txinclude.Includer
}

func (e *spamEOA) include(ctx context.Context, opts ...txplan.Option) (*txinclude.IncludedTx, error) {
	unsigned, err := txplan.NewPlannedTx(e.plan, txplan.Combine(opts...)).Unsigned.Eval(ctx)
	if err != nil {
		return nil, err
	}
	return e.includer.Include(ctx, unsigned)
}

func newSpamEOA(cl *ethclient.Client, key *ecdsa.PrivateKey, chainID *big.Int, el txinclude.EL, startNonce uint64) *spamEOA {
	signer := txinclude.NewPkSigner(key, chainID)
	return &spamEOA{
		plan: spamPlan(cl, key),
		includer: txinclude.NewLimit(
			txinclude.NewPersistent(signer, el, txinclude.WithStartNonce(startNonce)),
			spamMaxConcurrentTxs,
		),
	}
}

// spamPlan is a minimal txplan base for the spammer. It deliberately omits a
// nonce source — the txinclude.Persistent includer owns nonce assignment — and
// derives dynamic fees from the latest block (a 1 gwei tip over the current base
// fee), matching the acceptance test's EOA plan once its gas limit is overridden.
func spamPlan(cl *ethclient.Client, key *ecdsa.PrivateKey) txplan.Option {
	return txplan.Combine(
		txplan.WithChainID(cl),
		txplan.WithPrivateKey(key),
		txplan.WithAgainstLatestBlockEthClient(cl),
	)
}

type roundRobin struct {
	items []*spamEOA
	idx   atomic.Uint64
}

func newRoundRobin(items []*spamEOA) *roundRobin {
	return &roundRobin{items: items}
}

func (r *roundRobin) get() *spamEOA {
	next := (r.idx.Add(1) - 1) % uint64(len(r.items))
	return r.items[next]
}

// aimd is a self-contained additive-increase/multiplicative-decrease rate
// controller, a port of loadtest.AIMD (which is bound to the devstack test
// harness). rps starts at baseRPS, rises by baseRPS/10 every spamAdjustWindow
// completions while the failure rate stays below spamFailRateThreshold, and is
// halved otherwise.
type aimd struct {
	rps           atomic.Uint64
	increaseDelta uint64
	slotTime      time.Duration
	readyCh       chan struct{}

	mu        sync.Mutex
	completed uint64
	failed    uint64
}

func newAIMD(baseRPS uint64, slotTime time.Duration) *aimd {
	base := max(baseRPS, 1)
	a := &aimd{
		increaseDelta: max(base/10, 1),
		slotTime:      slotTime,
		readyCh:       make(chan struct{}),
	}
	a.rps.Store(base)
	return a
}

// start ticks the ready channel at the current rps until ctx is cancelled, then
// closes ready so consumers can range over it and stop.
func (a *aimd) start(ctx context.Context) {
	defer close(a.readyCh)
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(a.slotTime / time.Duration(a.rps.Load())):
			select {
			case a.readyCh <- struct{}{}:
			default: // consumers busy; skip this slot
			}
		}
	}
}

func (a *aimd) ready() <-chan struct{} {
	return a.readyCh
}

func (a *aimd) adjust(success bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.completed++
	if !success {
		a.failed++
	}
	if a.completed < spamAdjustWindow {
		return
	}
	failRate := float64(a.failed) / float64(a.completed)
	cur := a.rps.Load()
	if failRate > spamFailRateThreshold {
		a.rps.Store(max(uint64(float64(cur)*spamDecreaseFactor), 1))
	} else {
		a.rps.Store(cur + a.increaseDelta)
	}
	a.completed = 0
	a.failed = 0
}
