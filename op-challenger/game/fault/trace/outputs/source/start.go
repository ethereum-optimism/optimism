package source

import (
	"context"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L2Source interface {
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
	L2BlockRefByNumber(ctx context.Context, num uint64) (eth.L2BlockRef, error)
}

// FindGuaranteedSafeHead finds a L2 block where the L1 origin is the most recent L1 block closes to l1BlockNum
// where the block is guaranteed to now be safe because the sequencer window has expired.
// That is: block.origin.Number + sequencerWindowSize < l1BlockNum
// Note that the derivation rules guarantee that there is at least 1 L2 block for each L1 block.
// Otherwise deposits from the skipped L1 block would be missed.
func FindGuaranteedSafeHead(ctx context.Context, rollupCfg *rollup.Config, l1BlockNum uint64, l2Client L2Source) (eth.BlockID, error) {
	if l1BlockNum <= rollupCfg.SeqWindowSize {
		// The sequencer window hasn't completed yet, so the only guaranteed safe block is L2 genesis
		return rollupCfg.Genesis.L2, nil
	}
	safeHead, err := l2Client.L2BlockRefByLabel(ctx, eth.Safe)
	if err != nil {
		return eth.BlockID{}, fmt.Errorf("failed to load local safe head: %w", err)
	}
	safeL1BlockNum := l1BlockNum - rollupCfg.SeqWindowSize - 1
	start := rollupCfg.Genesis.L2.Number
	end := safeHead.Number
	for start <= end {
		mid := (start + end) / 2
		l2Block, err := l2Client.L2BlockRefByNumber(ctx, mid)
		if err != nil {
			return eth.BlockID{}, fmt.Errorf("failed to retrieve l2 block %v: %w", mid, err)
		}
		if l2Block.L1Origin.Number == safeL1BlockNum {
			return l2Block.ID(), nil
		} else if l2Block.L1Origin.Number < safeL1BlockNum {
			start = mid + 1
		} else {
			end = mid - 1
		}
	}
	return rollupCfg.Genesis.L2, nil
}
