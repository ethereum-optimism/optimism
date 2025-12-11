package sequencing

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-node/rollup/derive"
	"github.com/ethereum-optimism/optimism/op-node/rollup/engine"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/event"
)

type L1Blocks interface {
	derive.L1BlockRefByHashFetcher
	derive.L1BlockRefByNumberFetcher
}

type L1OriginSelector struct {
	ctx  context.Context
	log  log.Logger
	cfg  *rollup.Config
	spec *rollup.ChainSpec

	recoverMode atomic.Bool

	l1 L1Blocks

	// Internal cache of L1 origins for faster access.
	currentOrigin eth.L1BlockRef
	nextOrigin    eth.L1BlockRef

	mu sync.Mutex
}

func NewL1OriginSelector(ctx context.Context, log log.Logger, cfg *rollup.Config, l1 L1Blocks) *L1OriginSelector {
	return &L1OriginSelector{
		ctx:  ctx,
		log:  log,
		cfg:  cfg,
		spec: rollup.NewChainSpec(cfg),
		l1:   l1,
	}
}

func (los *L1OriginSelector) SetRecoverMode(enabled bool) {
	los.recoverMode.Store(enabled)
}

func (los *L1OriginSelector) ResetOrigins() {
	los.reset()
}

func (los *L1OriginSelector) OnEvent(ctx context.Context, ev event.Event) bool {
	switch x := ev.(type) {
	case engine.ForkchoiceUpdateEvent:
		los.onForkchoiceUpdate(x.UnsafeL2Head)
	case rollup.ResetEvent:
		los.ResetOrigins()
	default:
		return false
	}
	return true
}

// FindL1Origin determines what the L1 Origin for the next L2 Block should be.
func (los *L1OriginSelector) FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {

	// Get cached values for currentOrigin and nextOrigin
	currentOrigin, nextOrigin, err := los.CurrentAndNextOrigin(ctx, l2Head)
	if err != nil {
		return eth.L1BlockRef{}, err
	}

	// If in recover mode, get next origin synchronously
	matchAutoDerivation := los.recoverMode.Load()
	if (matchAutoDerivation && nextOrigin == eth.L1BlockRef{}) {
		nextOrigin, err = los.fetch(ctx, currentOrigin.Number+1)
		if errors.Is(err, ethereum.NotFound) {
			// We caught up to tip and no longer want to match auto derivation
			matchAutoDerivation = false
		} else if err != nil {
			return eth.L1BlockRef{}, err
		}
	}

	// TODO harmonize how we represent "no data for next origin"
	var nextOriginPtr *eth.L1BlockRef
	if (nextOrigin != eth.L1BlockRef{}) {
		nextOriginPtr = &nextOrigin
	}

	// defer to pure function
	o, err := FindL1OriginOfNextL2Block(los.cfg,
		&l2Head,
		&currentOrigin,
		nextOriginPtr,
		matchAutoDerivation)

	if err != nil {
		return *o, err
	}
	return eth.BlockRef{}, err
}

// CurrentAndNextOrigin returns the current cached values for the current L1 origin for the supplied l2Head, and it's successor.
// It only performs a fetch to L1 if the cache is invalid.
// The cache can be update asynchronously by other methods on L1OriginSelector.
func (los *L1OriginSelector) CurrentAndNextOrigin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, eth.L1BlockRef, error) {
	los.mu.Lock()
	defer los.mu.Unlock()

	if l2Head.L1Origin == los.currentOrigin.ID() {
		// Most likely outcome: the L2 head is still on the current origin.
	} else if l2Head.L1Origin == los.nextOrigin.ID() {
		// If the L2 head has progressed to the next origin, update the current and next origins.
		los.currentOrigin = los.nextOrigin
		los.nextOrigin = eth.L1BlockRef{}
	} else {
		// If for some reason the L2 head is not on the current or next origin, we need to find the
		// current origin block and reset the next origin.
		// This is most likely to occur on the first block after a restart.

		// Grab a reference to the current L1 origin block. This call is by hash and thus easily cached.
		currentOrigin, err := los.l1.L1BlockRefByHash(ctx, l2Head.L1Origin.Hash)
		if err != nil {
			return eth.L1BlockRef{}, eth.L1BlockRef{}, err
		}

		los.currentOrigin = currentOrigin
		los.nextOrigin = eth.L1BlockRef{}
	}

	return los.currentOrigin, los.nextOrigin, nil
}

func (los *L1OriginSelector) maybeSetNextOrigin(nextOrigin eth.L1BlockRef) {
	los.mu.Lock()
	defer los.mu.Unlock()

	// Set the next origin if it is the subsequent block by number.
	// On reorgs, this might not be the immediate child of the current origin
	// since the hash is not checked.
	if nextOrigin.Number == los.currentOrigin.Number+1 {
		los.nextOrigin = nextOrigin
	}
}

func (los *L1OriginSelector) onForkchoiceUpdate(unsafeL2Head eth.L2BlockRef) {
	// Only allow a relatively small window for fetching the next origin, as this is performed
	// on a best-effort basis.
	ctx, cancel := context.WithTimeout(los.ctx, 500*time.Millisecond)
	defer cancel()

	currentOrigin, nextOrigin, err := los.CurrentAndNextOrigin(ctx, unsafeL2Head)
	if err != nil {
		los.log.Error("Failed to get current and next L1 origin on forkchoice update", "err", err)
		return
	}

	los.tryFetchNextOrigin(ctx, currentOrigin, nextOrigin)
}

// tryFetchNextOrigin schedules a fetch for the next L1 origin block if it is not already set.
// This method always closes the channel, even if the next origin is already set.
func (los *L1OriginSelector) tryFetchNextOrigin(ctx context.Context, currentOrigin, nextOrigin eth.L1BlockRef) {
	// If the next origin is already set, we don't need to do anything.
	if nextOrigin != (eth.L1BlockRef{}) {
		return
	}

	// If the current origin is not set, we can't schedule the next origin check.
	if currentOrigin == (eth.L1BlockRef{}) {
		return
	}

	if _, err := los.fetch(ctx, currentOrigin.Number+1); err != nil {
		if errors.Is(err, ethereum.NotFound) {
			los.log.Debug("No next potential L1 origin found")
		} else {
			los.log.Error("Failed to get next origin", "err", err)
		}
	}
}

func (los *L1OriginSelector) fetch(ctx context.Context, number uint64) (eth.L1BlockRef, error) {
	// Attempt to find the next L1 origin block, where the next origin is the immediate child of
	// the current origin block.
	// The L1 source can be shimmed to hide new L1 blocks and enforce a sequencer confirmation distance.
	nextOrigin, err := los.l1.L1BlockRefByNumber(ctx, number)
	if err != nil {
		return eth.L1BlockRef{}, err
	}

	los.maybeSetNextOrigin(nextOrigin)

	return nextOrigin, nil
}

func (los *L1OriginSelector) reset() {
	los.mu.Lock()
	defer los.mu.Unlock()

	los.currentOrigin = eth.L1BlockRef{}
	los.nextOrigin = eth.L1BlockRef{}
}
