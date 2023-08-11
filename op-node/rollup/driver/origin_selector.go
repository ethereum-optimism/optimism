package driver

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type L1Blocks interface {
	derive.L1BlockRefByHashFetcher
	derive.L1BlockRefByNumberFetcher
}

type L1OriginSelector struct {
	log log.Logger
	cfg *rollup.Config

	l1 L1Blocks
}

func NewL1OriginSelector(log log.Logger, cfg *rollup.Config, l1 L1Blocks) *L1OriginSelector {
	return &L1OriginSelector{
		log: log,
		cfg: cfg,
		l1:  l1,
	}
}

// FindL1Origin determines what the next L1 Origin should be.
// The L1 Origin is either the L2 Head's Origin, or the following L1 block
// if the next L2 block's time is greater than or equal to the L2 Head's Origin.
func (los *L1OriginSelector) FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
	// Grab a reference to the current L1 origin block. This call is by hash and thus easily cached.
	currentOrigin, err := los.l1.L1BlockRefByHash(ctx, l2Head.L1Origin.Hash)
	if err != nil {
		return eth.L1BlockRef{}, err
	}
	log := los.log.New("current", currentOrigin, "current_time", currentOrigin.Time,
		"l2_head", l2Head, "l2_head_time", l2Head.Time)

	// If we are past the sequencer depth, we may want to advance the origin, but need to still
	// check the time of the next origin.
	pastSeqDrift := l2Head.Time+los.cfg.BlockTime > currentOrigin.Time+los.cfg.MaxSequencerDrift
	if pastSeqDrift {
		log.Warn("Next L2 block time is past the sequencer drift + current origin time")
	}

	// Attempt to find the next L1 origin block, where the next origin is the immediate child of
	// the current origin block.
	// The L1 source can be shimmed to hide new L1 blocks and enforce a sequencer confirmation distance.
	nextOrigin, err := los.l1.L1BlockRefByNumber(ctx, currentOrigin.Number+1)
	if err != nil {
		if pastSeqDrift {
			return eth.L1BlockRef{}, fmt.Errorf("cannot build next L2 block past current L1 origin %s by more than sequencer time drift, and failed to find next L1 origin: %w", currentOrigin, err)
		}
		if errors.Is(err, ethereum.NotFound) {
			log.Debug("No next L1 block found, repeating current origin")
		} else {
			log.Error("Failed to get next origin. Falling back to current origin", "err", err)
		}
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
