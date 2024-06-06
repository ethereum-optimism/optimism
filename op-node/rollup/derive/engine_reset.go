package derive

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/sync"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type ResetL2 interface {
	sync.L2Chain
	SystemConfigL2Fetcher
}

// ResetEngine walks the L2 chain backwards until it finds a plausible unsafe head,
// and an L2 safe block that is guaranteed to still be from the L1 chain.
func ResetEngine(ctx context.Context, log log.Logger, cfg *rollup.Config, ec ResetEngineControl, l1 sync.L1Chain, l2 ResetL2, syncCfg *sync.Config, safeHeadNotifs SafeHeadListener) error {
	result, err := sync.FindL2Heads(ctx, cfg, l1, l2, log, syncCfg)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find the L2 Heads to start from: %w", err))
	}
	finalized, safe, unsafe := result.Finalized, result.Safe, result.Unsafe
	l1Origin, err := l1.L1BlockRefByHash(ctx, safe.L1Origin.Hash)
	if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to fetch the new L1 progress: origin: %v; err: %w", safe.L1Origin, err))
	}
	if safe.Time < l1Origin.Time {
		return NewResetError(fmt.Errorf("cannot reset block derivation to start at L2 block %s with time %d older than its L1 origin %s with time %d, time invariant is broken",
			safe, safe.Time, l1Origin, l1Origin.Time))
	}

	ec.SetUnsafeHead(unsafe)
	ec.SetSafeHead(safe)
	ec.SetPendingSafeL2Head(safe)
	ec.SetFinalizedHead(finalized)
	ec.SetBackupUnsafeL2Head(eth.L2BlockRef{}, false)
	ec.ResetBuildingState()

	log.Debug("Reset of Engine is completed", "safeHead", safe, "unsafe", unsafe, "safe_timestamp", safe.Time,
		"unsafe_timestamp", unsafe.Time, "l1Origin", l1Origin)

	if safeHeadNotifs != nil {
		if err := safeHeadNotifs.SafeHeadReset(safe); err != nil {
			return err
		}
		if safeHeadNotifs.Enabled() && safe.Number == cfg.Genesis.L2.Number && safe.Hash == cfg.Genesis.L2.Hash {
			// The rollup genesis block is always safe by definition. So if the pipeline resets this far back we know
			// we will process all safe head updates and can record genesis as always safe from L1 genesis.
			// Note that it is not safe to use cfg.Genesis.L1 here as it is the block immediately before the L2 genesis
			// but the contracts may have been deployed earlier than that, allowing creating a dispute game
			// with a L1 head prior to cfg.Genesis.L1
			l1Genesis, err := l1.L1BlockRefByNumber(ctx, 0)
			if err != nil {
				return fmt.Errorf("failed to retrieve L1 genesis: %w", err)
			}
			if err := safeHeadNotifs.SafeHeadUpdated(safe, l1Genesis.ID()); err != nil {
				return err
			}
		}
	}
	return nil
}
