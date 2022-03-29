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
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
)

type L1Chain interface {
	L1BlockRefByNumber(ctx context.Context, l1Num uint64) (eth.L1BlockRef, error)
	L1HeadBlockRef(ctx context.Context) (eth.L1BlockRef, error)
}

type L2Chain interface {
	L2BlockRefByNumber(ctx context.Context, l2Num *big.Int) (eth.L2BlockRef, error)
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
}

var WrongChainErr = errors.New("wrong chain")
var TooDeepReorgErr = errors.New("reorg is too deep")

const MaxReorgDepth = 500

// FindSafeL2Head takes the supplied L2 start block and walks the L2 chain until it finds the first L2 block reachable from the supplied
// block that is also canonical.
func FindSafeL2Head(ctx context.Context, start eth.BlockID, l1 L1Chain, l2 L2Chain, genesis *rollup.Genesis) (eth.L2BlockRef, error) {
	// Starting point
	l2Head, err := l2.L2BlockRefByHash(ctx, start.Hash)
	if err != nil {
		return eth.L2BlockRef{}, fmt.Errorf("failed to fetch L2 head: %w", err)
	}
	reorgDepth := 0
	// Walk L2 chain from L2 head to first L2 block which has a L1 Parent that is canonical. May walk to L2 genesis
	for n := l2Head; ; {
		l1header, err := l1.L1BlockRefByNumber(ctx, n.L1Origin.Number)
		if err != nil {
			// Generic error, bail out.
			if !errors.Is(err, ethereum.NotFound) {
				return eth.L2BlockRef{}, fmt.Errorf("failed to fetch L1 block %v: %w", n.L1Origin.Number, err)
			}
			// L1 block not found, keep walking chain
		} else {
			// L1 Block found, check if matches & should keep walking the chain
			if l1header.Hash == n.L1Origin.Hash {
				return n, nil
			}
		}

		// Don't walk past genesis. If we were at the L2 genesis, but could not find the L1 genesis
		// pointed to from it, we are on the wrong L1 chain.
		if n.Hash == genesis.L2.Hash || n.Number == genesis.L2.Number {
			return eth.L2BlockRef{}, WrongChainErr
		}

		// Pull L2 parent for next iteration
		n, err = l2.L2BlockRefByHash(ctx, n.ParentHash)
		if err != nil {
			return eth.L2BlockRef{}, fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.ParentHash, err)
		}
		reorgDepth++
		if reorgDepth >= MaxReorgDepth {
			return eth.L2BlockRef{}, TooDeepReorgErr
		}
	}
}
