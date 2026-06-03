package dsl

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum-optimism/optimism/op-service/clock"
	"github.com/ethereum-optimism/optimism/op-service/retry"
)

// crossUnsafeHead is the JSON shape returned by op-reth's eth_crossUnsafeHead.
type crossUnsafeHead struct {
	Number hexutil.Uint64 `json:"number"`
	Hash   common.Hash    `json:"hash"`
}

// queryCrossUnsafeHead fetches op-reth's runtime cross-unsafe head (eth_crossUnsafeHead).
// Only available on op-reth nodes configured with --rollup.cross-unsafe-head-source-rpc;
// other execution clients will error.
func (el *L2ELNode) queryCrossUnsafeHead() (uint64, common.Hash) {
	ctx, cancel := context.WithTimeout(el.ctx, DefaultTimeout)
	defer cancel()
	var head crossUnsafeHead
	err := el.inner.EthClient().RPC().CallContext(ctx, &head, "eth_crossUnsafeHead")
	el.require.NoError(err, "eth_crossUnsafeHead call failed")
	return uint64(head.Number), head.Hash
}

// CrossUnsafeHeadReachedFn waits for the runtime cross-unsafe head to reach (>=) target with a
// non-zero hash. Reaching a block proves op-reth runtime-validated it (and everything below it)
// against the configured source chain.
func (el *L2ELNode) CrossUnsafeHeadReachedFn(target uint64, attempts int) CheckFunc {
	return func() error {
		logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "target", target)
		logger.Info("expecting cross-unsafe head to reach target")
		return retry.Do0(el.ctx, attempts, &retry.FixedStrategy{Dur: 2 * time.Second},
			func() error {
				number, hash := el.queryCrossUnsafeHead()
				logger.Info("cross-unsafe head", "number", number, "hash", hash, "target", target)
				if number >= target && hash != emptyHash {
					return nil
				}
				return fmt.Errorf("cross-unsafe head %d has not reached target %d", number, target)
			})
	}
}

// CrossUnsafeHeadStaysBelowFn asserts the runtime cross-unsafe head never reaches (>=) limit over
// the polling window (attempts * 2s). Used to prove op-reth refuses to advance past a block whose
// executing message cannot be validated against the source chain.
func (el *L2ELNode) CrossUnsafeHeadStaysBelowFn(limit uint64, attempts int) CheckFunc {
	return func() error {
		logger := el.log.With("name", el.inner.Name(), "chain", el.ChainID(), "limit", limit)
		logger.Info("expecting cross-unsafe head to stay below limit")
		for range attempts {
			if err := clock.SystemClock.SleepCtx(el.ctx, 2*time.Second); err != nil { // nosemgrep: flake-sleep-in-test -- asserting absence of progress; no chain event to wait on
				return err
			}
			number, _ := el.queryCrossUnsafeHead()
			logger.Info("cross-unsafe head", "number", number, "limit", limit)
			if number >= limit {
				return fmt.Errorf("cross-unsafe head %d reached forbidden limit %d", number, limit)
			}
		}
		return nil
	}
}
