// Package glamsterdamtest implements the post-Glamsterdam conformance checks
// shared between the op-acceptance-tests acceptance suite and the
// check-glamsterdam CLI. The checks mirror the original sysgo-bound
// TestSafeHeadAdvancesAfterGlamsterdam: confirm the L1 has activated
// Glamsterdam (upstream EL name: Amsterdam; EIP-7843 SlotNumber header
// field), confirm the L2 safe head reaches an L1 origin at or past the
// activation block, and optionally confirm the safe block carries enough
// traffic to demonstrate live block production.
package glamsterdamtest

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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-core/predeploys"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// LatestL1Header fetches the L1 latest unsafe header. The CLI passes a
// closure over ethclient.Client; the acceptance test passes a closure over
// the devstack L1EL DSL — neither side needs to share a fat interface.
type LatestL1Header func(ctx context.Context) (*types.Header, error)

// L2BlockGasUsed returns the gas used by the L2 block at hash. Closed over
// ethclient.Client in the CLI; closed over the devstack L2EL DSL in the test.
type L2BlockGasUsed func(ctx context.Context, hash common.Hash) (uint64, error)

// SyncStatusFetcher matches what op-node's rollup-rpc client exposes
// (`*sources.RollupClient`). The acceptance test passes the devstack's
// `sys.L2CL.Escape().RollupAPI()` (apis.RollupClient), which also satisfies
// this shape.
type SyncStatusFetcher interface {
	SyncStatus(ctx context.Context) (*eth.SyncStatus, error)
}

var _ SyncStatusFetcher = (*sources.RollupClient)(nil)

// WaitForAmsterdamOnL1 polls the L1 latest header until it contains a
// non-nil SlotNumber, then returns that header. Mirrors the first loop
// of TestSafeHeadAdvancesAfterGlamsterdam (glamsterdam_test.go).
func WaitForAmsterdamOnL1(ctx context.Context, lgr log.Logger, fetch LatestL1Header, poll time.Duration) (*types.Header, error) {
	if poll <= 0 {
		poll = 2 * time.Second
	}
	lgr.Info("waiting for Amsterdam (Glamsterdam) activation on L1")
	for {
		h, err := fetch(ctx)
		if err == nil && h != nil && h.SlotNumber != nil {
			lgr.Info("Amsterdam active on L1", "block", h.Number.Uint64(), "slot", *h.SlotNumber, "hash", h.Hash())
			return h, nil
		}
		if err != nil {
			lgr.Debug("L1 header poll failed; will retry", "err", err)
		}
		if err := sleepCtx(ctx, poll); err != nil {
			return nil, err
		}
	}
}

// WaitForSafeHeadPastL1 polls the rollup-node SyncStatus until the safe L2
// block's L1Origin number reaches target, then returns that safe block ref.
// Mirrors the second loop of TestSafeHeadAdvancesAfterGlamsterdam.
func WaitForSafeHeadPastL1(ctx context.Context, lgr log.Logger, rollup SyncStatusFetcher, target uint64, poll time.Duration) (eth.L2BlockRef, error) {
	if poll <= 0 {
		poll = 2 * time.Second
	}
	lgr.Info("waiting for L2 safe head to advance past Amsterdam L1 origin", "targetL1Block", target)
	for {
		st, err := rollup.SyncStatus(ctx)
		if err == nil && st.SafeL2.L1Origin.Number >= target {
			lgr.Info("L2 safe head reached post-Amsterdam L1 origin",
				"safeL2", st.SafeL2.Number, "safeL2Hash", st.SafeL2.Hash, "l1Origin", st.SafeL2.L1Origin.Number)
			return st.SafeL2, nil
		}
		if err != nil {
			lgr.Debug("syncStatus poll failed; will retry", "err", err)
		} else {
			lgr.Debug("safe head not yet past Amsterdam",
				"safeL2", st.SafeL2.Number, "safeL1Origin", st.SafeL2.L1Origin.Number, "target", target)
		}
		if err := sleepCtx(ctx, poll); err != nil {
			return eth.L2BlockRef{}, err
		}
	}
}

// CheckSafeHeadTraffic asserts that the L2 block at safeHash carries at least
// threshold gas-used. Mirrors the third loop of
// TestSafeHeadAdvancesAfterGlamsterdam (threshold = gasLimit/2 + 1 there).
// Passing a zero threshold skips the assertion and just reports gas used.
func CheckSafeHeadTraffic(ctx context.Context, lgr log.Logger, gasUsed L2BlockGasUsed, safeHash common.Hash, threshold uint64) error {
	used, err := gasUsed(ctx, safeHash)
	if err != nil {
		return fmt.Errorf("fetch safe L2 block %s: %w", safeHash, err)
	}
	lgr.Info("safe L2 block traffic", "hash", safeHash, "gasUsed", used, "threshold", threshold)
	if threshold > 0 && used < threshold {
		return fmt.Errorf("safe block %s gas used %d below threshold %d", safeHash, used, threshold)
	}
	return nil
}

// SpamL2 sends a stream of low-gas transactions from key to
// predeploys.L1BlockAddr, throughput-controlled by ticksPerSecond. Returns a
// cancel function the caller must invoke to stop the spammer. The CLI uses
// this to push the L2 toward the gas threshold; callers that already have a
// busy chain can skip it.
//
// Unlike the test's loadtest.SpammerFunc (which depends on the devtest
// harness), this is a plain goroutine that does its own nonce bookkeeping
// against a single account. One-account-one-goroutine intentionally — the
// CLI is for verification, not load generation.
func SpamL2(ctx context.Context, lgr log.Logger, l2 *ethclient.Client, key *ecdsa.PrivateKey, ticksPerSecond float64, gasLimit uint64) (cancel func(), err error) {
	chainID, err := l2.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch L2 chain id: %w", err)
	}
	signer := types.LatestSignerForChainID(chainID)
	from := crypto.PubkeyToAddress(key.PublicKey)
	nonce, err := l2.PendingNonceAt(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("fetch pending nonce for %s: %w", from, err)
	}
	gasTip, err := l2.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch suggested gas tip cap: %w", err)
	}
	head, err := l2.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("fetch L2 head: %w", err)
	}
	baseFee := head.BaseFee
	if baseFee == nil {
		baseFee = big.NewInt(0)
	}
	// gasFeeCap = 2*baseFee + tip — gives us some headroom across blocks.
	gasFeeCap := new(big.Int).Add(new(big.Int).Mul(baseFee, big.NewInt(2)), gasTip)
	to := predeploys.L1BlockAddr

	if ticksPerSecond <= 0 {
		ticksPerSecond = 50 // 50 tps default — enough to fill a 60M gas, 2s block at 50k gas/tx
	}
	interval := time.Duration(float64(time.Second) / ticksPerSecond)
	spamCtx, stop := context.WithCancel(ctx)
	var wg sync.WaitGroup
	var sent atomic.Uint64
	var fail atomic.Uint64
	wg.Add(1)
	go func() {
		defer wg.Done()
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-spamCtx.Done():
				return
			case <-t.C:
			}
			tx := types.NewTx(&types.DynamicFeeTx{
				ChainID:   chainID,
				Nonce:     nonce,
				GasTipCap: new(big.Int).Set(gasTip),
				GasFeeCap: new(big.Int).Set(gasFeeCap),
				Gas:       gasLimit,
				To:        &to,
				Value:     big.NewInt(1),
			})
			signed, err := types.SignTx(tx, signer, key)
			if err != nil {
				fail.Add(1)
				lgr.Debug("sign tx failed", "err", err)
				continue
			}
			if err := l2.SendTransaction(spamCtx, signed); err != nil {
				fail.Add(1)
				lgr.Debug("send tx failed", "nonce", nonce, "err", err)
				// Re-sync nonce on persistent failures.
				if n, nerr := l2.PendingNonceAt(spamCtx, from); nerr == nil {
					nonce = n
				}
				continue
			}
			sent.Add(1)
			nonce++
		}
	}()

	return func() {
		stop()
		wg.Wait()
		lgr.Info("spam stopped", "sent", sent.Load(), "failed", fail.Load())
	}, nil
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return errors.Join(ctx.Err(), errors.New("interrupted while waiting"))
	case <-t.C:
		return nil
	}
}
