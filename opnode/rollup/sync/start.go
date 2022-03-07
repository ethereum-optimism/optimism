// The sync package is responsible for reconciling L1 and L2.
//
//
// The ethereum chain is a DAG of blocks with the root block being the genesis block.
// At any given time, the head (or tip) of the chain can change if an offshoot of the chain
// has a higher number. This is known as a re-organization of the canonical chain.
// Each block points to a parent block and the node is responsible for deciding which block is the head
// and thus the mapping from block number to canonical block.
//
// The optimism chain has similar properties, but also retains references to the ethereum chain.
// Each optimism block retains a reference to an L1 block and to its parent L2 block.
// The L2 chain node must satisfy the following validity rules
//     1. l2block.height == l2parent.block.height + 1
//     2. l2block.l1parent.height >= l2block.l2parent.l1parent.height
//     3. l2block.l1parent is in the canonical chain on L1
//     4. l1_rollup_genesis is reachable from l2block.l1parent
//
//
// During normal operation, both the L1 and L2 canonical chains can change, due to a reorg
// or an extension (new block).
//     - L1 reorg
//     - L1 extension
//     - L2 reorg
//     - L2 extension
//
// When one of these changes occurs, the rollup node needs to determine what the new L2 Head should be.
// In a simple extension case, the L2 head remains the same, but in the case of a re-org on L1, it needs
// to find the first L2 block where the l1parent is in the L1 canonical chain.
// In the case of a re-org, it is also helpful to obtain the L1 blocks after the L1 base to re-start the
// chain derivation process.

package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
)

var WrongChainErr = errors.New("wrong chain")
var TooDeepReorgErr = errors.New("reorg is too deep")
var MaxReorgDepth = 500
var MaxBlocksInL1Range = uint64(100)

// FindSyncStart finds the L2 head and the chain of L1 blocks after the L1 base block.
// Note: The ChainSource should memoize calls as the L1 and L2 chains will be walked multiple times.
// The L2 Head is the highest possible l2block such that it is valid (see above rules).
// It also returns a portion of the L1 chain starting just after l2block.l1parent.number.
//     - The returned L1 blocks were canonical when the function was called.
//     - The returned L1 block are contiguous and ordered from low to high.
//     - The first block (if len > 0) has height l2block.l1parent.number + 1.
//     - The length of the array may be any value, including 0.
// If err is not nil, the above return values are not well defined. An error will be returned in the following cases:
//     - Wrapped ethereum.NotFound if it could not find a block in L1 or L2. This error may be temporary.
//     - Wrapped WrongChainErr if the l1_rollup_genesis block is not reachable from the L2 chain.
func FindSyncStart(ctx context.Context, source ChainSource, genesis *rollup.Genesis) ([]eth.BlockID, eth.BlockID, error) {
	l2Head, err := FindSafeL2Head(ctx, source, genesis)
	if err != nil {
		return nil, eth.BlockID{}, err
	}
	l1blocks, err := FindL1Range(ctx, source, l2Head.L1Origin)
	if err != nil {
		return nil, eth.BlockID{}, fmt.Errorf("failed to fetch l1 range: %w", err)
	}

	return l1blocks, l2Head.Self, nil

}

// FindSafeL2Head takes the current L2 Head and then finds the topmost L2 head that is valid
// In the case that there are no re-orgs, this is just the L2 head. Otherwise it has to walk back
// until it finds the first L2 block that is based on a canonical L1 block.
func FindSafeL2Head(ctx context.Context, source ChainSource, genesis *rollup.Genesis) (eth.L2BlockRef, error) {
	// Starting point
	l2Head, err := source.L2NodeByNumber(ctx, nil)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("failed to fetch L2 head: %w", err)
	}
	reorgDepth := 0
	// Walk L2 chain from L2 head to first L2 block which has a L1 Parent that is canonical. May walk to L2 genesis
	for n := l2Head; ; {
		l1header, err := source.L1NodeByNumber(ctx, n.L1Origin.Number)
		if err != nil {
			// Generic error, bail out.
			if !errors.Is(err, ethereum.NotFound) {
				return eth.L2BlockRef{}, fmt.Errorf("failed to fetch L1 block %v: %w", n.L1Origin.Number, err)
			}
			// L1 block not found, keep walking chain
		} else {
			// L1 Block found, check if matches & should keep walking the chain
			if l1header.Self.Hash == n.L1Origin.Hash {
				return n, nil
			}
		}

		// Don't walk past genesis. If we were at the L2 genesis, but could not find the L1 genesis
		// pointed to from it, we are on the wrong L1 chain.
		if n.Self.Hash == genesis.L2.Hash || n.Self.Number == genesis.L2.Number {
			return eth.L2BlockRef{}, WrongChainErr
		}

		// Pull L2 parent for next iteration
		n, err = source.L2NodeByHash(ctx, n.Parent.Hash)
		if err != nil {
			return eth.L2BlockRef{}, fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.Parent.Hash, err)
		}
		reorgDepth++
		if reorgDepth >= MaxReorgDepth {
			return eth.L2BlockRef{}, TooDeepReorgErr
		}
	}
}

// FindL1Range returns a range of L1 block beginning just after `begin`.
func FindL1Range(ctx context.Context, source ChainSource, begin eth.BlockID) ([]eth.BlockID, error) {
	// Ensure that we start on the expected chain.
	if canonicalBegin, err := source.L1NodeByNumber(ctx, begin.Number); err != nil {
		return nil, fmt.Errorf("failed to fetch L1 block %v %v: %w", begin.Number, begin.Hash, err)
	} else {
		if canonicalBegin.Self != begin {
			return nil, fmt.Errorf("Re-org at begin block. Expected: %v. Actual: %v", begin, canonicalBegin.Self)
		}
	}

	l1head, err := source.L1HeadNode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch head L1 block: %w", err)
	}
	maxBlocks := MaxBlocksInL1Range
	// Cap maxBlocks if there are less than maxBlocks between `begin` and the head of the chain.
	if l1head.Self.Number-begin.Number <= maxBlocks {
		maxBlocks = l1head.Self.Number - begin.Number
	}

	if maxBlocks == 0 {
		return nil, nil
	}

	prevHash := begin.Hash
	var res []eth.BlockID
	for i := begin.Number + 1; i < begin.Number+maxBlocks+1; i++ {
		n, err := source.L1NodeByNumber(ctx, i)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch L1 block %v: %w", i, err)
		}
		// TODO(Joshua): Look into why this fails around the genesis block
		if n.Parent.Number != 0 && n.Parent.Hash != prevHash {
			return nil, errors.New("re-organization occurred while attempting to get l1 range")
		}
		prevHash = n.Self.Hash
		res = append(res, n.Self)
	}

	return res, nil
}
