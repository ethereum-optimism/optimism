package filter

import (
	"context"
	"fmt"
	"time"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

const reorgRecoveryInterval = 500 * time.Millisecond

func (b *Backend) runReorgRecovery(ctx context.Context) {
	defer b.reorgRecoveryWg.Done()

	ticker := time.NewTicker(reorgRecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.tryResolveReorgs(ctx)
		}
	}
}

func (b *Backend) tryResolveReorgs(ctx context.Context) {
	for chainID, ingester := range b.chains {
		if ingester.Error() == nil {
			continue
		}

		blockID, timestamp, err := b.recoverChainReorg(ctx, chainID, ingester)
		if err != nil {
			b.log.Warn("Failed to auto-resolve reorg", "chain", chainID, "target", eth.Finalized, "err", err)
			continue
		}

		b.crossValidator.ResetCrossValidatedTimestamp(timestamp)
		ingester.ClearError()
		b.log.Info("Auto-resolved reorg-triggered failsafe",
			"chain", chainID,
			"target", eth.Finalized,
			"block", blockID.Number,
			"hash", blockID.Hash,
			"timestamp", timestamp)
	}
}

func (b *Backend) recoverChainReorg(
	ctx context.Context,
	chainID eth.ChainID,
	ingester ChainIngester,
) (eth.BlockID, uint64, error) {
	errState := ingester.Error()
	if errState == nil {
		return eth.BlockID{}, 0, fmt.Errorf("cannot auto-resolve reorg: chain %s has no ingester error", chainID)
	}
	if errState.Reason != ErrorReorg {
		return eth.BlockID{}, 0, reorgResolutionSkipped(errState.Reason)
	}

	blockID, timestamp, err := ingester.RewindToFinalized(ctx)
	if err != nil {
		return eth.BlockID{}, 0, fmt.Errorf("failed to rewind to finalized: %w", err)
	}
	return blockID, timestamp, nil
}

func reorgResolutionSkipped(reason IngesterErrorReason) error {
	return fmt.Errorf("cannot auto-resolve ingester error reason %s", reason)
}
