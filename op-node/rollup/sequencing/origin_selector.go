package sequencing

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	log  log.Logger
	cfg  *rollup.Config
	spec *rollup.ChainSpec

	l1 L1Blocks
}

func NewL1OriginSelector(log log.Logger, cfg *rollup.Config, l1 L1Blocks) *L1OriginSelector {
	return &L1OriginSelector{
		log:  log,
		cfg:  cfg,
		spec: rollup.NewChainSpec(cfg),
		l1:   l1,
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
	msd := los.spec.MaxSequencerDrift(currentOrigin.Time)
	log := los.log.New("current", currentOrigin, "current_time", currentOrigin.Time,
		"l2_head", l2Head, "l2_head_time", l2Head.Time, "max_seq_drift", msd)

	seqDrift := l2Head.Time + los.cfg.BlockTime - currentOrigin.Time

	// If we are past the sequencer depth, we may want to advance the origin, but need to still
	// check the time of the next origin.
	pastSeqDrift := seqDrift > msd
	if pastSeqDrift {
		log.Warn("Next L2 block time is past the sequencer drift + current origin time")
		seqDrift = msd
	}

	// Calculate the maximum time we can spend attempting to fetch the next L1 origin block.
	// Time spent fetching this information is time not spent building the next L2 block, so
	// we generally prioritize keeping this value small, allowing for a nonzero failure rate.
	// As the next L2 block time approaches the max sequencer drift, increase our tolerance for
	// slower L1 fetches in order to avoid falling too far behind.
	fetchTimeout := time.Second + (9*time.Second*time.Duration(seqDrift))/time.Duration(msd)
	fetchCtx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	// Attempt to find the next L1 origin block, where the next origin is the immediate child of
	// the current origin block.
	// The L1 source can be shimmed to hide new L1 blocks and enforce a sequencer confirmation distance.
	nextOrigin, err := los.l1.L1BlockRefByNumber(fetchCtx, currentOrigin.Number+1)
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
