package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum/go-ethereum/common"
)

var WrongChainErr = errors.New("wrong chain")

// The ethereum chain is a DAG of blocks with the root block being the genesis block.
// At any given time, the head (or tip) of the chain can change if an offshoot of the chain
// has a higher difficulty. This is known as a re-organization of the canonical chain.
// Each block points to a parent block and the node is responsible for deciding which block is the head
// and thus the mapping from block number to canonical block.
//
// The optimism chain has similar properties, but also retains references to the ethereum chain.
// Each optimism block retains a reference to an ethereum (L1) block and to it's parent optimism (L2) block.
// The L2 chain node must satisfy the following validity rules
//     1. l2block.height == l2parent.block.height + 1
//     2. l2block.l1parent.height >= l2block.l2parent.l1parent.height
//     3. l2block.l1parent is in the canonical chain on L1
//     4. l1_rollup_genesis is reachable from l2block.l1parent
//
//
// During normal operation, both the L1 and L2 canonical chains can change.
//     - L1 reorg
//     - L1 extension
//     - L2 reorg
//     - L2 extension
//
// When one of those actions occurs, the L2 chain head may need to be updated and we would like a portion of the L1 chain
// that we should use to derive the L2 chain from.
// These are two different functions, but doing the first requires walking both chains in which case it is easy to also
// return a portion of the chain.
//
// This function returns the highest possible l2block that is valid. This is the (possibly new) L2 chain head.
// It also returns a portion of the L1 chain starting just after l2block.l1parent.number.
//     - The L1 chain was canonical when the function was called.
//     - The L1 chain is continuous
//     - The L1 chain is from low to high
//     - The length is not defined.
// If err is not nil, the above return values are not well defined. An error will be returned in the following cases:
//     - Wrapped ethereum.NotFound if it could not find a block in L1 or L2. This error may be temporary.
//     - Wrapped WrongChainErr if the l1_rollup_genesis block is not reachable from the L2 chain.
//     - ??

func V3FindSyncStart(ctx context.Context, source SyncReferenceV2, genesis *rollup.Genesis) ([]eth.BlockID, eth.BlockID, error) {
	l2Head, err := findL2Head(ctx, source, genesis)
	if err != nil {
		return nil, eth.BlockID{}, err
	}
	l1blocks, err := findL1Range(ctx, source, l2Head.l1parent)
	if err != nil {
		return nil, eth.BlockID{}, fmt.Errorf("failed to fetch l1 range: %w", err)
	}

	return l1blocks, l2Head.self, nil

}

// findL2Head takes the current L2 Head and then finds the topmost L2 head that is valid
// In the case that there are no re-orgs,
func findL2Head(ctx context.Context, source SyncReferenceV2, genesis *rollup.Genesis) (L2Node, error) {
	// Starting point
	l2Head, err := source.L2NodeByNumber(ctx, nil, genesis)
	if err != nil {
		return L2Node{}, fmt.Errorf("failed to fetch L2 head: %w", err)
	}
	// Walk L2 chain from L2 head to first L2 block which has a L1 Parent that is canonical. May walk to L2 genesis
	for n := l2Head; ; {
		l1header, err := source.L1NodeByNumber(ctx, n.l1parent.Number)
		if err != nil {
			// Generic error, bail out.
			if !errors.Is(err, ethereum.NotFound) {
				return L2Node{}, fmt.Errorf("failed to fetch L1 block %v: %w", n.l1parent.Number, err)
			}
			// L1 block not found, keep walking chain
		} else {
			// L1 Block found, check if matches & should keep walking the chain
			if l1header.self.Hash == n.l1parent.Hash {
				return n, nil
			}
		}

		// Don't walk past genesis. If we were at the L2 genesis, but could not find the L1 genesis
		// pointed to from it, we are on the wrong L1 chain.
		if n.self.Hash == genesis.L2.Hash || n.self.Number == genesis.L2.Number {
			return L2Node{}, WrongChainErr
		}

		// Pull L2 parent for next iteration
		n, err = source.L2NodeByHash(ctx, n.l2parent.Hash, genesis)
		if err != nil {
			return L2Node{}, fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.l2parent.Hash, err)
		}
	}
}

func findL1Range(ctx context.Context, source SyncReferenceV2, begin eth.BlockID) ([]eth.BlockID, error) {
	if _, err := source.L1NodeByNumber(ctx, begin.Number); err != nil {
		return nil, fmt.Errorf("failed to fetch L1 block %v %v: %w", begin.Number, begin.Hash, err)
	}
	// TODO: Check hash here (even if slightly redudant)
	l1head, err := source.L1HeadNode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch head L1 block: %w", err)
	}
	maxBlocks := uint64(100)
	if l1head.self.Number-begin.Number <= maxBlocks {
		fmt.Println("Capping max blocks")
		maxBlocks = l1head.self.Number - begin.Number
		fmt.Println(maxBlocks)
	}

	prevHash := begin.Hash
	var res []eth.BlockID
	for i := begin.Number + 1; i < begin.Number+maxBlocks+1; i++ {
		fmt.Println(i, i)
		n, err := source.L1NodeByNumber(ctx, i)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch L1 block %v: %w", i, err)
		}
		fmt.Println(prevHash, n.parent.Hash)
		if n.parent.Hash != prevHash {
			return nil, errors.New("re-organization occurred while attempting to get l1 range")
		}
		prevHash = n.self.Hash
		res = append(res, n.self)
	}

	return res, nil
}

// FindSyncStart finds nextL1s: the L1 blocks needed next for sync, to derive into a L2 block on top of refL2.
// If the L1 reorgs then this will find the common history to build on top of and then follow the first step of the reorg.
func FindSyncStart(ctx context.Context, reference SyncReference, genesis *rollup.Genesis) (nextRefL1, refL2 eth.BlockID, err error) {
	var refL1 eth.BlockID    // the L1 block the refL2 was derived from
	var parentL2 common.Hash // the parent of refL2
	// Start at L2 head
	refL1, refL2, parentL2, err = reference.RefByL2Num(ctx, nil, genesis)
	if err != nil {
		err = fmt.Errorf("failed to fetch L2 head: %v", err)
		return
	}
	// Check if L1 source has the block
	var currentL1 eth.BlockID // the expected L1 block at the height of refL1
	currentL1, _, err = reference.RefByL1Num(ctx, refL1.Number)
	if err != nil {
		if !errors.Is(err, ethereum.NotFound) {
			err = fmt.Errorf("failed to lookup block %d in L1: %w", refL1.Number, err)
			return
		}
		// If the L1 did not find the block, it might be out of sync.
		// We cannot sync from L1 in this case, but we still traverse back to
		// make sure we are not just in a reorg to a L1 chain with fewer blocks.
		err = nil
		currentL1 = eth.BlockID{} // empty = not found
	}
	if currentL1 == refL1 {
		// L1 node has head-block of execution-engine, so we should fetch the L1 block that builds on top.
		var ontoL1 eth.BlockID // ontoL1 is the parent, to make sure we got a nextRefL1 that connects as expected.
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
			err = fmt.Errorf("failed to lookup block %s in L2: %w", refL2, err) // refL2 is previous parentL2
			return
		}
		// Check if L1 source has the block that derived the L2 block we are planning to build on
		currentL1, _, err = reference.RefByL1Num(ctx, refL1.Number)
		if err != nil {
			if !errors.Is(err, ethereum.NotFound) {
				err = fmt.Errorf("failed to lookup block %d in L1: %w", refL1.Number, err)
				return
			}
			// again, if L1 does not have the block, then we just search if we are reorging.
			err = nil
			currentL1 = eth.BlockID{} // empty = not found
		}
		if currentL1 == refL1 {
			// check if we had a L1 block to build on top of the common chain with
			if nextRefL1 == (eth.BlockID{}) {
				err = ethereum.NotFound
			}
			return
		}
		// TODO: after e.g. initial N steps, use binary search instead
		// (relies on block numbers, not great for tip of chain, but nice-to-have in deep reorgs)
	}
	// Enforce that we build on the desired genesis block.
	// The engine might be configured for a different chain or older testnet.
	if refL2 != genesis.L2 {
		err = fmt.Errorf("unexpected L2 genesis block: %s, expected %s, %w", refL2, genesis.L2, WrongChainErr)
		return
	}
	if currentL1 != genesis.L1 {
		err = fmt.Errorf("unexpected L1 anchor block: %s, expected %s, %w", currentL1, genesis.L1, WrongChainErr)
		return
	}
	// we got the correct genesis, all good, but a lot to sync!
	return
}
