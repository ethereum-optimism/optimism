package sequencing

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// FindL1OriginOfNextL2Block finds the L1 origin of the next L2 block.
// It returns an error if there is no way to build a block satisfying
// derivation constraints with the supplied l1OriginChild.
// You can pass a nil pointer for l1OriginChild if it is temporarily
// unavailable.
// TODO: is there a way we can insist on having the l1OriginChild?
// TODO: does the PR introducing async l1 origin fetching provide a hint we should not do that?
func FindL1OriginOfNextL2Block(
	l2Head *eth.L2BlockRef,
	l1Origin *eth.L1BlockRef, l1OriginChild *eth.L1BlockRef,
	matchAutoDerivation bool) (*eth.L1BlockRef, error) {

	if l1Origin == nil {
		panic("supplied l1 origin is nil")
	}

	if l2Head.L1Origin.Hash != l1Origin.Hash {
		// TODO maybe return sentinel error for reorg detected?
		panic("supplied l1 origin is not the origin of the supplied l2 head")
	}

	if l1OriginChild != nil && l1OriginChild.ParentHash != l1Origin.Hash {
		panic("supplied l1 origin child is not a child of the supplied l1 origin")
	}

	l2BlockTime := uint64(2) // TODO generalize this
	maxDrift := uint64(1800) // TODO generalize this
	nextL2BlockTime := l2Head.Time + l2BlockTime

	driftStick := nextL2BlockTime - l1Origin.Time
	driftTwist := nextL2BlockTime - l1OriginChild.Time

	if driftTwist > maxDrift {
		return nil, fmt.Errorf("drift of next L2 block exceeds maximum %d even when progressing to l1OriginChild", maxDrift)
	}

	if matchAutoDerivation {
		// THIS SECTION SHOULD MATCH PROTOCOL RULES EXACTLY
		// e.g. https://github.com/ethereum-optimism/optimism/blob/3b22b347f73774c0bf639aade750c10c9dc703d5/op-node/rollup/derive/base_batch_stage.go#L162-L206
		if l1OriginChild == nil {
			return nil, fmt.Errorf("l1OriginChild is nil but required to match auto derivation, try again later")
		}
		// Progress to l1OriginChild if doing so would respect the requirement
		// that L2 blocks cannot point to a future L1 block.
		if nextL2BlockTime >= l1OriginChild.Time {
			return l1OriginChild, nil
		}
		// If we cannot adopt the l1OriginChild, use the current l1 origin.
		if nextL2BlockTime < l1OriginChild.Time {
			return l1Origin, nil
		}
	} else {
		// THIS SECTION IS partly POLICY, and can be modified/optimized
		// by exploiting the freedom allowed by the nonzero sequencer drift.
		// Progress to l1OriginChild if it exists
		// and if doing so would respect the requirement
		// that L2 blocks cannot point to a future L1 block.
		if l1OriginChild != nil && nextL2BlockTime >= l1OriginChild.Time {
			return l1OriginChild, nil
		}
		if l1OriginChild != nil && nextL2BlockTime < l1OriginChild.Time {
			// This is the only time we are allowed to violate the max drift condition
			// in order to preserve the l2time >= l1time invariant
			return l1Origin, nil
		}
		// Otherwise, stick with the current L1 origin unless doing so would exceed the maximum drift.
		if driftStick > maxDrift {
			// We are allowed to violate this condition ONLY when
			return nil, fmt.Errorf("cannot progress to l1OriginChild and drift of next L2 block would exceed maximum %d even when progressing to l1Origin", maxDrift)
		}
		return l1Origin, nil
	}

}
