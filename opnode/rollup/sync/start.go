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
	l2Head, err := FindL2Head(ctx, source, genesis)
	if err != nil {
		return nil, eth.BlockID{}, err
	}
	l1blocks, err := FindL1Range(ctx, source, l2Head.l1parent)
	if err != nil {
		return nil, eth.BlockID{}, fmt.Errorf("failed to fetch l1 range: %w", err)
	}

	return l1blocks, l2Head.self, nil

}

// findL2Head takes the current L2 Head and then finds the topmost L2 head that is valid
// In the case that there are no re-orgs,
func FindL2Head(ctx context.Context, source SyncReferenceV2, genesis *rollup.Genesis) (L2Node, error) {
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

func FindL1Range(ctx context.Context, source SyncReferenceV2, begin eth.BlockID) ([]eth.BlockID, error) {
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
		// TODO(Joshua): Look into why this fails around the genesis block
		if n.parent.Number != 0 && n.parent.Hash != prevHash {
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
	l1s, refL2, err := V3FindSyncStart(ctx, SyncSourceV2{reference}, genesis)
	if err != nil && len(l1s) > 0 {
		nextRefL1 = l1s[0]
	}
	return
}
