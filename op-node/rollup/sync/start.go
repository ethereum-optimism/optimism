// The sync package is responsible for reconciling L1 and L2.
//
// The Ethereum chain is a DAG of blocks with the root block being the genesis block. At any given
// time, the head (or tip) of the chain can change if an offshoot/branch of the chain has a higher
// total difficulty. This is known as a re-organization of the canonical chain. Each block points to
// a parent block and the node is responsible for deciding which block is the head and thus the
// mapping from block number to canonical block.
//
// The Optimism (L2) chain has similar properties, but also retains references to the Ethereum (L1)
// chain. Each L2 block retains a reference to an L1 block (its "L1 origin", i.e. L1 block
// associated with the epoch that the L2 block belongs to) and to its parent L2 block. The L2 chain
// node must satisfy the following validity rules:
//
//     1. l2block.number == l2block.l2parent.block.number + 1
//     2. l2block.l1Origin.number >= l2block.l2parent.l1Origin.number
//     3. l2block.l1Origin is in the canonical chain on L1
//     4. l1_rollup_genesis is an ancestor of l2block.l1Origin
//
// During normal operation, both the L1 and L2 canonical chains can change, due to a re-organisation
// or due to an extension (new L1 or L2 block).
//
// When one of these changes occurs, the rollup node needs to determine what the new L2 head blocks
// should be. We track two L2 head blocks:
//
//     - The *unsafe L2 block*: This is the highest L2 block whose L1 origin is a plausible (1)
//       extension of the canonical L1 chain (as known to the op-node).
//     - The *safe L2 block*: This is the highest L2 block whose epoch's sequencing window is
//       complete within the canonical L1 chain (as known to the op-node).
//
// (1) Plausible meaning that the blockhash of the L2 block's L1 origin (as reported in the L1
//     Attributes deposit within the L2 block) is not canonical at another height in the L1 chain,
//     and the same holds for all its ancestors.
//
// In particular, in the case of L1 extension, the L2 unsafe head will generally remain the same,
// but in the case of an L1 re-org, we need to search for the new safe and unsafe L2 block.
package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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

// isCanonical returns the following values:
// - `aheadOrCanonical: true if the supplied block is ahead of the known head of the L1 chain,
//    or canonical in the L1 chain.
// - `canonical`: true if the block is canonical in the L1 chain.
func isAheadOrCanonical(ctx context.Context, l1 L1Chain, block eth.BlockID) (aheadOrCanonical bool, canonical bool, err error) {
	if l1Head, err := l1.L1HeadBlockRef(ctx); err != nil {
		return false, false, err
	} else if block.Number > l1Head.Number {
		return true, false, nil
	} else if canonical, err := l1.L1BlockRefByNumber(ctx, block.Number); err != nil {
		return false, false, err
	} else {
		canonical := canonical.Hash == block.Hash
		return canonical, canonical, nil
	}
}

// FindL2Heads walks back from `start` (the previous unsafe L2 block) and finds the unsafe and safe
// L2 blocks.
//
//     - The *unsafe L2 block*: This is the highest L2 block whose L1 origin is a plausible (1)
//       extension of the canonical L1 chain (as known to the op-node).
//     - The *safe L2 block*: This is the highest L2 block whose epoch's sequencing window is
//       complete within the canonical L1 chain (as known to the op-node).
//
// (1) Plausible meaning that the blockhash of the L2 block's L1 origin (as reported in the L1
//     Attributes deposit within the L2 block) is not canonical at another height in the L1 chain,
//     and the same holds for all its ancestors.
func FindL2Heads(ctx context.Context, start eth.L2BlockRef, seqWindowSize uint64,
	l1 L1Chain, l2 L2Chain, genesis *rollup.Genesis) (unsafe eth.L2BlockRef, safe eth.L2BlockRef, err error) {

	// Loop 1. Walk the L2 chain backwards until we find an L2 block whose L1 origin is canonical.

	// Current L2 block.
	n := start

	// Number of blocks between n and start.
	reorgDepth := 0

	// Blockhash of L1 origin hash for the L2 block during the previous iteration, 0 for first
	// iteration. When this changes as we walk the L2 chain backwards, it means we're seeing a different
	// (earlier) epoch.
	var prevL1OriginHash common.Hash

	// The highest L2 ancestor of `start` (or `start` itself) whose ancestors are not (yet) known
	// to have a non-canonical L1 origin. Empty if no such candidate is known yet. Guaranteed to be
	// set after exiting from Loop 1.
	var highestPlausibleCanonicalOrigin eth.L2BlockRef

	for {
		// Check if l1Origin is canonical when we get to a new epoch.
		if prevL1OriginHash != n.L1Origin.Hash {
			prevL1OriginHash = n.L1Origin.Hash

			if plausible, canonical, err := isAheadOrCanonical(ctx, l1, n.L1Origin); err != nil {
				return eth.L2BlockRef{}, eth.L2BlockRef{}, err
			} else if !plausible {
				// L1 origin nor ahead of L1 head nor canonical, discard previous candidate and
				// keep looking.
				highestPlausibleCanonicalOrigin = eth.L2BlockRef{}
			} else {
				if highestPlausibleCanonicalOrigin == (eth.L2BlockRef{}) {
					// No highest plausible candidate, make L2 block new candidate.
					highestPlausibleCanonicalOrigin = n
				}
				if canonical {
					break
				}
			}
		}

		// Don't walk past genesis. If we were at the L2 genesis, but could not find its L1 origin,
		// the L2 chain is building on the wrong L1 branch.
		if n.Hash == genesis.L2.Hash || n.Number == genesis.L2.Number {
			return eth.L2BlockRef{}, eth.L2BlockRef{}, WrongChainErr
		}

		// Pull L2 parent for next iteration
		n, err = l2.L2BlockRefByHash(ctx, n.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, eth.L2BlockRef{},
				fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.ParentHash, err)
		}

		reorgDepth++
		if reorgDepth >= MaxReorgDepth {
			// If the reorg depth is too large, something is fishy.
			// This can legitimately happen if L1 goes down for a while. But in that case,
			// restarting the L2 node with a bigger configured MaxReorgDepth is an acceptable
			// stopgap solution.
			// Currently this can also happen if the L2 node is down for a while, but in the future
			// state sync should prevent this issue.
			return eth.L2BlockRef{}, eth.L2BlockRef{}, TooDeepReorgErr
		}
	}

	// Loop 2. Walk from the L1 origin of the `n` block (*) back to the L1 block that starts the
	// sequencing window ending at that block. Instead of iterating on L1 blocks, we actually
	// iterate on L2 blocks, because we want to find the safe L2 head, i.e. the highest L2 block
	// whose L1 origin is the start of the sequencing window.

	// (*) `n` being at this stage the highest L2 block whose L1 origin is canonical.

	// Depth counter: we need to walk back `seqWindowSize` L1 blocks in order to find the start
	// of the sequencing window.
	depth := uint64(1)

	// Before entering the loop: `prevL1OriginHash == n.L1Origin.Hash`
	// The original definitions of `n` and `prevL1OriginHash` still hold.
	for {
		// Advance depth if we change to a different (earlier) epoch.
		if n.L1Origin.Hash != prevL1OriginHash {
			depth++
			prevL1OriginHash = n.L1Origin.Hash
		}

		// Found an L2 block whose L1 origin is the start of the sequencing window.
		// Note: We also ensure that we are on the block number with the 0 seq number.
		// This is a little hacky, but kinda works. The issue is about where the
		// batch queue should start building.
		if depth == seqWindowSize && n.SequenceNumber == 0 {
			return highestPlausibleCanonicalOrigin, n, nil
		}

		// Genesis is always safe.
		if n.Hash == genesis.L2.Hash || n.Number == genesis.L2.Number {
			safe = eth.L2BlockRef{Hash: genesis.L2.Hash, Number: genesis.L2.Number,
				Time: genesis.L2Time, L1Origin: genesis.L1, SequenceNumber: 0}
			return highestPlausibleCanonicalOrigin, safe, nil
		}

		// Pull L2 parent for next iteration.
		n, err = l2.L2BlockRefByHash(ctx, n.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, eth.L2BlockRef{},
				fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.ParentHash, err)
		}
	}
}
