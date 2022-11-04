package driver

import (
	"context"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
)

type L1Blocks interface {
	derive.L1BlockRefByHashFetcher
	derive.L1BlockRefByNumberFetcher
}

type L1OriginSelector struct {
	log log.Logger
	cfg *rollup.Config

	l1                  L1Blocks
	sequencingConfDepth uint64
}

func NewL1OriginSelector(log log.Logger, cfg *rollup.Config, l1 L1Blocks, sequencingConfDepth uint64) *L1OriginSelector {
	return &L1OriginSelector{
		log:                 log,
		cfg:                 cfg,
		l1:                  l1,
		sequencingConfDepth: sequencingConfDepth,
	}
}

// FindL1Origin determines what the next L1 Origin should be.
// The L1 Origin is either the L2 Head's Origin, or the following L1 block
// if the next L2 block's time is greater than or equal to the L2 Head's Origin.
func (los *L1OriginSelector) FindL1Origin(ctx context.Context, l1Head eth.L1BlockRef, l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
	// If we are at the head block, don't do a lookup.
	if l2Head.L1Origin.Hash == l1Head.Hash {
		return l1Head, nil
	}

	// Grab a reference to the current L1 origin block.
	currentOrigin, err := los.l1.L1BlockRefByHash(ctx, l2Head.L1Origin.Hash)
	if err != nil {
		return eth.L1BlockRef{}, err
	}

	// If we are past the sequencer depth, we may want to advance the origin, but need to still
	// check the time of the next origin.
	pastSeqDrift := l2Head.Time+los.cfg.BlockTime > currentOrigin.Time+los.cfg.MaxSequencerDrift
	if pastSeqDrift {
		log.Info("Next L2 block time is past the sequencer drift + current origin time",
			"current", currentOrigin, "current_time", currentOrigin.Time,
			"l1_head", l1Head, "l1_head_time", l1Head.Time,
			"l2_head", l2Head, "l2_head_time", l2Head.Time,
			"depth", los.sequencingConfDepth)
	}

	if !pastSeqDrift && currentOrigin.Number+1+los.sequencingConfDepth > l1Head.Number {
		// TODO: we can decide to ignore confirmation depth if we would be forced
		//  to make an empty block (only deposits) by staying on the current origin.
		log.Info("sequencing with old origin to preserve conf depth",
			"current", currentOrigin, "current_time", currentOrigin.Time,
			"l1_head", l1Head, "l1_head_time", l1Head.Time,
			"l2_head", l2Head, "l2_head_time", l2Head.Time,
			"depth", los.sequencingConfDepth)
		return currentOrigin, nil
	}

	// Attempt to find the next L1 origin block, where the next origin is the immediate child of
	// the current origin block.
	nextOrigin, err := los.l1.L1BlockRefByNumber(ctx, currentOrigin.Number+1)
	if err != nil {
		// TODO: this could result in a bad origin being selected if we are past the seq
		// drift & should instead advance to the next origin.
		log.Error("Failed to get next origin. Falling back to current origin", "err", err)
		return currentOrigin, nil
	}

	// If the next L2 block time is greater than the next origin block's time, we can choose to
	// start building on top of the next origin. Sequencer implementation has some leeway here and
	// could decide to continue to build on top of the previous origin until the Sequencer runs out
	// of slack. For simplicity, we implement our Sequencer to always start building on the latest
	// L1 block when we can.
	if l2Head.Time+los.cfg.BlockTime >= nextOrigin.Time {
		return nextOrigin, nil
	}

	return currentOrigin, nil
}
