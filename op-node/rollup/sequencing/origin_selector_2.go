package sequencing

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var (
	ErrInvalidL1Origin      = fmt.Errorf("origin-selector: currentL1Origin.Hash != l2Head.L1Origin.Hash")
	ErrNextL1OriginOrphaned = fmt.Errorf("origin-selector: nextL1Origin.ParentHash != currentL1Origin.Hash")
	ErrNextL1OriginRequired = fmt.Errorf("origin-selector: nextL1Origin not supplied but required to satisfy constraints")
)

// FindL1OriginOfNextL2Block finds the L1 origin of the next L2 block.
// It returns an error if there is no way to build a block satisfying
// derivation constraints with the supplied l1OriginChild.
// You can pass a nil pointer for l1OriginChild if it is not yet available,
func FindL1OriginOfNextL2Block(
	cfg *rollup.Config,
	l2Head *eth.L2BlockRef,
	currentL1Origin *eth.L1BlockRef, nextL1Origin *eth.L1BlockRef,
	matchAutoDerivation bool) (*eth.L1BlockRef, error) {

	if currentL1Origin == nil {
		panic("origin-selector: currentl1Origin is nil")
	}
	if l2Head.L1Origin.Hash != currentL1Origin.Hash {
		return nil, ErrInvalidL1Origin
	}
	if nextL1Origin != nil && nextL1Origin.ParentHash != currentL1Origin.Hash {
		return nil, ErrNextL1OriginOrphaned
	}

	l2BlockTime := cfg.BlockTime
	maxDrift := rollup.NewChainSpec(cfg).MaxSequencerDrift(currentL1Origin.Time)
	nextL2BlockTime := l2Head.Time + l2BlockTime
	driftCurrent := nextL2BlockTime - currentL1Origin.Time

	if nextL1Origin == nil {
		if matchAutoDerivation {
			// This can cause unsafe block production to slow to the rate of L1 block production, if the L1 origin is caught up to the L1 Head.
			// Code higher up the call stack should ensure that matchAutoDerivation is false under such conditions.
			return nil, ErrNextL1OriginRequired
		} else {
			// If we don't yet have the nextL1Origin, stick with the current L1 origin unless doing so would exceed the maximum drift.
			if driftCurrent > maxDrift {
				// Return an error so the caller knows it needs to fetch the next l1 origin now.
				return nil, fmt.Errorf("%w: drift of next L2 block would exceed maximum %d unless nextl1Origin is adopted", ErrNextL1OriginRequired, maxDrift)
			}
			return currentL1Origin, nil
		}
	}

	// Progress to l1OriginChild if doing so would respect the requirement
	// that L2 blocks cannot point to a future L1 block.
	if nextL2BlockTime >= nextL1Origin.Time {
		// Adopt it if
		return nextL1Origin, nil
	} else {
		// If we cannot adopt the l1OriginChild, use the current l1 origin.
		return currentL1Origin, nil
	}
}
