package l2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum/go-ethereum/common"
)

func FindSyncStart(ctx context.Context, reference SyncReference, genesis *Genesis) (nextRefL1 eth.BlockID, refL2 eth.BlockID, err error) {
	// 10 seconds for the whole thing (TODO: or do we want individual timeouts?)
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	var refL1 eth.BlockID
	var parentL2 common.Hash
	// Start at L2 head
	refL1, refL2, parentL2, err = reference.RefByL2Num(ctx, nil, genesis)
	if err != nil {
		err = fmt.Errorf("failed to fetch L2 head: %v", err)
		return
	}
	// Check if L1 source has the block
	var currentL1 eth.BlockID
	currentL1, _, err = reference.RefByL1Num(ctx, refL1.Number)
	if err != nil {
		err = fmt.Errorf("failed to lookup block %d in L1: %v", refL1.Number, err)
		return
	}
	if currentL1 == refL1 {
		// L1 node has head-block of execution-engine, so we should fetch the L1 block that builds on top.
		var ontoL1 eth.BlockID
		nextRefL1, ontoL1, err = reference.RefByL1Num(ctx, refL1.Number+1)
		if err != nil {
			// If refL1 is the head block, then we might not have a next block to build on the head
			if errors.Is(err, ethereum.NotFound) {
				// return the same as the engine head was already built on, no error.
				nextRefL1 = refL1
				refL2 = eth.BlockID{Hash: parentL2, Number: refL2.Number}
				if refL2.Number > 0 {
					refL2.Number -= 1
				}
				err = nil
				return
			}
			return
		}
		// The L1 source might rug us with a reorg between API calls, catch that.
		if ontoL1 != currentL1 {
			err = fmt.Errorf("the L1 source reorged, the block for N+1 %s doesn't have the previously fetched block N %s as parent, but builds on %s instead", nextRefL1, currentL1, ontoL1)
		}
		return
	}

	// Search back: linear walk back from engine head. Should only be as deep as the reorg.
	for refL2.Number > 0 {
		// remember the canonical L1 block that builds on top of the L1 source block of the L2 parent block.
		nextRefL1 = currentL1
		refL1, refL2, parentL2, err = reference.RefByL2Hash(ctx, parentL2, genesis)
		if err != nil {
			// TODO: re-attempt look-up, now that we already traversed previous history?
			err = fmt.Errorf("failed to lookup block %s in L2: %v", refL2, err) // refL2 is previous parentL2
			return
		}
		// Check if L1 source has the block that derived the L2 block we are planning to build on
		currentL1, _, err = reference.RefByL1Num(ctx, refL1.Number)
		if err != nil {
			err = fmt.Errorf("failed to lookup block %d in L1: %v", refL1.Number, err)
			return
		}
		if currentL1 == refL1 {
			return
		}
		// TODO: after e.g. initial N steps, use binary search instead
		// (relies on block numbers, not great for tip of chain, but nice-to-have in deep reorgs)
	}
	// Enforce that we build on the desired genesis block.
	// The engine might be configured for a different chain or older testnet.
	if refL2 != genesis.L2 {
		err = fmt.Errorf("engine was anchored at unexpected block: %s, expected %s", refL2, genesis.L2)
		return
	}
	// we got the correct genesis, all good, but a lot to sync!
	return
}
