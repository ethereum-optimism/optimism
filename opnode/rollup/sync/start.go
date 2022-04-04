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
//     2. l2block.l1Origin.height >= l2block.l2parent.l1Origin.height
//     3. l2block.l1Origin is in the canonical chain on L1
//     4. l1_rollup_genesis is an ancestor of l2block.l1Origin
//
//
// During normal operation, both the L1 and L2 canonical chains can change, due to a reorg
// or an extension (new block).
//     - L1 reorg
//     - L1 extension
//     - L2 reorg
//     - L2 extension
//
// When one of these changes occurs, the rollup node needs to determine what the new L2 Heads should be.
// In a simple extension case, the L2 head remains the same, but in the case of a re-org on L1, it needs
// to find the unsafe and safe blocks.
//
// Unsafe Block: The highest L2 block. If the L1 Attributes is ahead of the L1 head, it is assumed to be valid,
// if not, it walks back until it finds the first L2 block whose L1 Origin is canonical in the L1 chain.
// Safe Block: The highest L2 block whose sequence window has not changed during a reorg.
//
// The safe block can be found by walking back one sequence window from the "latest" L2 block. The latest L2
// block is the first L2 block whose L1 Origin is canonical in L1. If the unsafe block is ahead of the L1
// chain, the latest block and unsafe block are not the same.
package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
)

type L1Chain interface {
	L1HeadBlockRef(ctx context.Context) (eth.L1BlockRef, error)
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
}

type L2Chain interface {
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

var WrongChainErr = errors.New("wrong chain")
var TooDeepReorgErr = errors.New("reorg is too deep")

const MaxReorgDepth = 500

// isCanonical returns true if the supplied block ID is canonical in the L1 chain.
// It will suppress ethereum.NotFound errors
func isCanonical(ctx context.Context, l1 L1Chain, block eth.BlockID) (bool, error) {
	canonical, err := l1.L1BlockRefByNumber(ctx, block.Number)
	if err != nil && !errors.Is(err, ethereum.NotFound) {
		return false, err
	} else if err != nil {
		return false, nil
	}
	return canonical.Hash == block.Hash, nil
}

// FindL2Heads walks back from the supplied L2 blocks and finds the unsafe and safe L2 blocks.
// Unsafe Block: The highest L2 block. If the L1 Attributes is ahead of the L1 head, it is assumed to be valid,
// if not, it walks back until it finds the first L2 block whose L1 Origin is canonical in the L1 chain.
// Safe Block: The highest L2 block whose sequence window has not changed during a reorg.
func FindL2Heads(ctx context.Context, start eth.L2BlockRef, seqWindowSize int,
	l1 L1Chain, l2 L2Chain, genesis *rollup.Genesis) (unsafe eth.L2BlockRef, safe eth.L2BlockRef, err error) {
	reorgDepth := 0
	var prevL1OriginHash common.Hash
	// First check if the L1 Origin of the start block is ahead of the current L1 head
	// If so, we assume that this should be the next unsafe head for the sequencing window
	// We still need to walk back the safe head because we don't know where the reorg started.
	l1Head, err := l1.L1HeadBlockRef(ctx)
	if err != nil {
		return eth.L2BlockRef{}, eth.L2BlockRef{}, err
	}
	l2Ahead := start.L1Origin.Number > l1Head.Number
	var latest eth.L2BlockRef

	// Walk L2 chain until we find the "latest" L2 block. This the first L2 block whose L1 Origin is canonical.
	for n := start; ; {
		// Check if l1Origin is canonical when we get to a new epoch
		if prevL1OriginHash != n.L1Origin.Hash {
			if ok, err := isCanonical(ctx, l1, n.L1Origin); err != nil {
				return eth.L2BlockRef{}, eth.L2BlockRef{}, err
			} else if ok {
				latest = n
				break
			}
			prevL1OriginHash = n.L1Origin.Hash
		}
		// Don't walk past genesis. If we were at the L2 genesis, but could not find the L1 genesis
		// pointed to from it, we are on the wrong L1 chain.
		if n.Hash == genesis.L2.Hash || n.Number == genesis.L2.Number {
			return eth.L2BlockRef{}, eth.L2BlockRef{}, WrongChainErr
		}
		// Pull L2 parent for next iteration
		n, err = l2.L2BlockRefByHash(ctx, n.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.ParentHash, err)
		}
		reorgDepth++
		if reorgDepth >= MaxReorgDepth {
			return eth.L2BlockRef{}, eth.L2BlockRef{}, TooDeepReorgErr
		}
	}
	depth := 1 // SeqWindowSize is a length, but we are counting elements in the window.
	prevL1OriginHash = latest.L1Origin.Hash
	// Walk from the latest block back 1 Sequence Window of L1 Origins to determine the safe L2 block.
	for n := latest; ; {
		// Advance depth if new origin
		if n.L1Origin.Hash != prevL1OriginHash {
			depth++
			prevL1OriginHash = n.L1Origin.Hash
		}
		// Walked sufficiently far
		if depth == seqWindowSize {
			if l2Ahead {
				return start, n, nil
			} else {
				return latest, n, nil
			}

		}
		// Genesis is always safe.
		if n.Hash == genesis.L2.Hash || n.Number == genesis.L2.Number {
			safe = eth.L2BlockRef{Hash: genesis.L2.Hash, Number: genesis.L2.Number, Time: genesis.L2Time, L1Origin: genesis.L1}
			if l2Ahead {
				return start, safe, nil
			} else {
				return latest, safe, nil
			}

		}
		// Pull L2 parent for next iteration
		n, err = l2.L2BlockRefByHash(ctx, n.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, eth.L2BlockRef{}, fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.ParentHash, err)
		}
	}

}
