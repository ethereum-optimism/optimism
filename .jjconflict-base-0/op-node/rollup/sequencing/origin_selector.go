package sequencing

import (
	"context"
	"errors"
	"fmt"
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
// It wraps the FindL1OriginOfNextL2Block function and handles caching and network requests.
func (los *L1OriginSelector) FindL1Origin(ctx context.Context, l2Head eth.L2BlockRef) (eth.L1BlockRef, error) {
	recoverMode := los.recoverMode.Load()
	// Get cached values for currentOrigin and nextOrigin
	currentOrigin, nextOrigin, err := los.CurrentAndNextOrigin(ctx, l2Head)
	if err != nil {
		return eth.L1BlockRef{}, err
	}
	// Try to find the L1 origin given the current data in cache
	o, err := los.findL1OriginOfNextL2Block(
		l2Head,
		currentOrigin,
		nextOrigin,
		recoverMode)

	// If the cache doesn't have the next origin, but we now
	// know we definitely need it, fetch it and try again.
	if errors.Is(err, ErrNextL1OriginRequired) {
		nextOrigin, err = los.fetch(ctx, currentOrigin.Number+1)
		if err == nil || (recoverMode && errors.Is(err, ethereum.NotFound)) {
			// If we got the origin, or we are in recover mode and the origin is not found
			// (because we recovered the l1 origin up to the l1 tip)
			// try again with matchAutoDerivation = false.
			return los.findL1OriginOfNextL2Block(
				l2Head,
				currentOrigin,
				nextOrigin,
				false)
		} else {
			return eth.L1BlockRef{}, ErrNextL1OriginRequired
		}
	}
	return o, err
}

// CurrentAndNextOrigin returns the current cached values for the current L1 origin for the supplied l2Head, and its successor.
// It only performs a fetch to L1 if the cache is invalid.
// The cache can be updated asynchronously by other methods on L1OriginSelector.
// The returned currentOrigin should _always_ be non-empty, because it is populated from l2Head whose
// l1Origin is first specified in the rollup.Config.Genesis.L1 and progressed to non-empty values thereafter.
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

var (
	ErrInvalidL1Origin      = fmt.Errorf("origin-selector: currentL1Origin.Hash != l2Head.L1Origin.Hash")
	ErrNextL1OriginOrphaned = fmt.Errorf("origin-selector: nextL1Origin.ParentHash != currentL1Origin.Hash")
	ErrNextL1OriginRequired = fmt.Errorf("origin-selector: nextL1Origin not supplied but required to satisfy constraints")
)

// FindL1OriginOfNextL2Block finds the L1 origin of the next L2 block.
// It returns an error if there is no way to build a block satisfying
// derivation constraints with the supplied data.
// You can pass an empty nextL1Origin if it is not yet available
// removing the need for block building to wait on the result of network calls.
// This method is designed to be pure (it only reads the cfg property of the receiver)
// and should not have any side effects.
func (los *L1OriginSelector) findL1OriginOfNextL2Block(
	l2Head eth.L2BlockRef,
	currentL1Origin eth.L1BlockRef, nextL1Origin eth.L1BlockRef,
	matchAutoDerivation bool) (eth.L1BlockRef, error) {

	if (currentL1Origin == eth.L1BlockRef{}) {
		// This would indicate a programming error, since the currentL1Origin
		// should _always_ be available.
		// The first value (for block 1) is specified in rollup.Config.Genesis.L1
		// and it is then only updated to non-empty values.
		panic("origin-selector: currentL1Origin is empty")
	}
	if l2Head.L1Origin.Hash != currentL1Origin.Hash {
		return currentL1Origin, ErrInvalidL1Origin
	}
	if (nextL1Origin != eth.L1BlockRef{} && nextL1Origin.ParentHash != currentL1Origin.Hash) {
		return nextL1Origin, ErrNextL1OriginOrphaned
	}

	l2BlockTime := los.cfg.BlockTime
	maxDrift := rollup.NewChainSpec(los.cfg).MaxSequencerDrift(currentL1Origin.Time)
	nextL2BlockTime := l2Head.Time + l2BlockTime
	driftCurrent := int64(nextL2BlockTime) - int64(currentL1Origin.Time)

	if (nextL1Origin == eth.L1BlockRef{}) {
		if matchAutoDerivation {
			// See https://github.com/ethereum-optimism/optimism/blob/ce9fa62d0c0325304fc37d91d87aa2e16a7f8356/op-node/rollup/derive/base_batch_stage.go#L186-L205
			// We need the next L1 origin to decide whether we can eagerly adopt it.
			// NOTE: This can cause unsafe block production to slow to the rate of L1 block production, if the L1 origin is caught up to the L1 Head.
			// Code higher up the call stack should ensure that matchAutoDerivation is false under such conditions.
			return eth.L1BlockRef{}, ErrNextL1OriginRequired
		} else {
			// If we don't yet have the nextL1Origin, stick with the current L1 origin unless doing so would exceed the maximum drift.
			if driftCurrent > int64(maxDrift) {
				// Return an error so the caller knows it needs to fetch the next l1 origin now.
				return eth.L1BlockRef{}, fmt.Errorf("%w: drift of next L2 block would exceed maximum %d unless nextl1Origin is adopted", ErrNextL1OriginRequired, maxDrift)
			}
			return currentL1Origin, nil
		}
	}

	driftNext := int64(nextL2BlockTime) - int64(nextL1Origin.Time)

	// Progress to l1OriginChild if doing so would respect the requirement
	// that L2 blocks cannot point to a future L1 block (negative drift).
	if driftNext >= 0 {
		return nextL1Origin, nil
	} else {
		// If we cannot adopt the l1OriginChild, use the current l1 origin.
		return currentL1Origin, nil
	}
}
