// Package sync is responsible for reconciling L1 and L2.
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
//  1. l2block.number == l2block.l2parent.block.number + 1
//  2. l2block.l1Origin.number >= l2block.l2parent.l1Origin.number
//  3. l2block.l1Origin is in the canonical chain on L1
//  4. l1_rollup_genesis is an ancestor of l2block.l1Origin
//
// During normal operation, both the L1 and L2 canonical chains can change, due to a re-organisation
// or due to an extension (new L1 or L2 block).
//
// In particular, in the case of L1 extension, the L2 unsafe head will generally remain the same,
// but in the case of an L1 re-org, we need to search for the new safe and unsafe L2 block.
package sync

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type L1Chain interface {
	L1BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L1BlockRef, error)
	L1BlockRefByNumber(ctx context.Context, number uint64) (eth.L1BlockRef, error)
	L1BlockRefByHash(ctx context.Context, hash common.Hash) (eth.L1BlockRef, error)
}

type L2Chain interface {
	L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error)
	L2BlockRefByLabel(ctx context.Context, label eth.BlockLabel) (eth.L2BlockRef, error)
}

var ReorgFinalizedErr = errors.New("cannot reorg finalized block")
var WrongChainErr = errors.New("wrong chain")
var TooDeepReorgErr = errors.New("reorg is too deep")

const MaxReorgSeqWindows = 5

type FindHeadsResult struct {
	Unsafe    eth.L2BlockRef
	Safe      eth.L2BlockRef
	Finalized eth.L2BlockRef
}

// currentHeads returns the current finalized, safe and unsafe heads of the execution engine.
// If nothing has been marked finalized yet, the finalized head defaults to the genesis block.
// If nothing has been marked safe yet, the safe head defaults to the finalized block.
func currentHeads(ctx context.Context, cfg *rollup.Config, l2 L2Chain) (*FindHeadsResult, error) {
	finalized, err := l2.L2BlockRefByLabel(ctx, eth.Finalized)
	if errors.Is(err, ethereum.NotFound) {
		// default to genesis if we have not finalized anything before.
		finalized, err = l2.L2BlockRefByHash(ctx, cfg.Genesis.L2.Hash)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find the finalized L2 block: %w", err)
	}

	safe, err := l2.L2BlockRefByLabel(ctx, eth.Safe)
	if errors.Is(err, ethereum.NotFound) {
		safe = finalized
	} else if err != nil {
		return nil, fmt.Errorf("failed to find the safe L2 block: %w", err)
	}

	unsafe, err := l2.L2BlockRefByLabel(ctx, eth.Unsafe)
	if err != nil {
		return nil, fmt.Errorf("failed to find the L2 head block: %w", err)
	}
	return &FindHeadsResult{
		Unsafe:    unsafe,
		Safe:      safe,
		Finalized: finalized,
	}, nil
}

// FindL2Heads walks back from `start` (the previous unsafe L2 block) and finds
// the finalized, unsafe and safe L2 blocks.
//
//   - The *unsafe L2 block*: This is the highest L2 block whose L1 origin is a *plausible*
//     extension of the canonical L1 chain (as known to the op-node).
//   - The *safe L2 block*: This is the highest L2 block whose epoch's sequencing window is
//     complete within the canonical L1 chain (as known to the op-node).
//   - The *finalized L2 block*: This is the L2 block which is known to be fully derived from
//     finalized L1 block data.
//
// Plausible: meaning that the blockhash of the L2 block's L1 origin
// (as reported in the L1 Attributes deposit within the L2 block) is not canonical at another height in the L1 chain,
// and the same holds for all its ancestors.
func FindL2Heads(ctx context.Context, cfg *rollup.Config, l1 L1Chain, l2 L2Chain, lgr log.Logger) (result *FindHeadsResult, err error) {
	// Fetch current L2 forkchoice state
	result, err = currentHeads(ctx, cfg, l2)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch current L2 forkchoice state: %w", err)
	}

	lgr.Info("Loaded current L2 heads", "unsafe", result.Unsafe, "safe", result.Safe, "finalized", result.Finalized,
		"unsafe_origin", result.Unsafe.L1Origin, "safe_origin", result.Safe.L1Origin)

	// Remember original unsafe block to determine reorg depth
	prevUnsafe := result.Unsafe

	// Current L2 block.
	n := result.Unsafe

	var highestL2WithCanonicalL1Origin eth.L2BlockRef // the highest L2 block with confirmed canonical L1 origin
	var l1Block eth.L1BlockRef                        // the L1 block at the height of the L1 origin of the current L2 block n.
	var ahead bool                                    // when "n", the L2 block, has a L1 origin that is not visible in our L1 chain source yet

	ready := false // when we found the block after the safe head, and we just need to return the parent block.

	// Each loop iteration we traverse further from the unsafe head towards the finalized head.
	// Once we pass the previous safe head and we have seen enough canonical L1 origins to fill a sequence window worth of data,
	// then we return the last L2 block of the epoch before that as safe head.
	// Each loop iteration we traverse a single L2 block, and we check if the L1 origins are consistent.
	for {
		// Fetch L1 information if we never had it, or if we do not have it for the current origin.
		// Optimization: as soon as we have a previous L1 block, try to traverse L1 by hash instead of by number, to fill the cache.
		if n.L1Origin.Hash == l1Block.ParentHash {
			b, err := l1.L1BlockRefByHash(ctx, n.L1Origin.Hash)
			if err != nil {
				// Exit, find-sync start should start over, to move to an available L1 chain with block-by-number / not-found case.
				return nil, fmt.Errorf("failed to retrieve L1 block: %w", err)
			}
			lgr.Info("Walking back L1Block by hash", "curr", l1Block, "next", b, "l2block", n)
			l1Block = b
			ahead = false
		} else if l1Block == (eth.L1BlockRef{}) || n.L1Origin.Hash != l1Block.Hash {
			b, err := l1.L1BlockRefByNumber(ctx, n.L1Origin.Number)
			// if L2 is ahead of L1 view, then consider it a "plausible" head
			notFound := errors.Is(err, ethereum.NotFound)
			if err != nil && !notFound {
				return nil, fmt.Errorf("failed to retrieve block %d from L1 for comparison against %s: %w", n.L1Origin.Number, n.L1Origin.Hash, err)
			}
			l1Block = b
			ahead = notFound
			lgr.Info("Walking back L1Block by number", "curr", l1Block, "next", b, "l2block", n)
		}

		lgr.Trace("walking sync start", "l2block", n)

		// Don't walk past genesis. If we were at the L2 genesis, but could not find its L1 origin,
		// the L2 chain is building on the wrong L1 branch.
		if n.Number == cfg.Genesis.L2.Number {
			// Check L2 traversal against L2 Genesis data, to make sure the engine is on the correct chain, instead of attempting sync with different L2 destination.
			if n.Hash != cfg.Genesis.L2.Hash {
				return nil, fmt.Errorf("%w L2: genesis: %s, got %s", WrongChainErr, cfg.Genesis.L2, n)
			}
			// Check L1 comparison against L1 Genesis data, to make sure the L1 data is from the correct chain, instead of attempting sync with different L1 source.
			if !ahead && l1Block.Hash != cfg.Genesis.L1.Hash {
				return nil, fmt.Errorf("%w L1: genesis: %s, got %s", WrongChainErr, cfg.Genesis.L1, l1Block)
			}
		}
		// Check L2 traversal against finalized data
		if (n.Number == result.Finalized.Number) && (n.Hash != result.Finalized.Hash) {
			return nil, fmt.Errorf("%w: finalized %s, got: %s", ReorgFinalizedErr, result.Finalized, n)
		}
		// Check we are not reorging L2 incredibly deep
		if n.L1Origin.Number+(MaxReorgSeqWindows*cfg.SeqWindowSize) < prevUnsafe.L1Origin.Number {
			// If the reorg depth is too large, something is fishy.
			// This can legitimately happen if L1 goes down for a while. But in that case,
			// restarting the L2 node with a bigger configured MaxReorgDepth is an acceptable
			// stopgap solution.
			return nil, fmt.Errorf("%w: traversed back to L2 block %s, but too deep compared to previous unsafe block %s", TooDeepReorgErr, n, prevUnsafe)
		}

		// If we don't have a usable unsafe head, then set it
		if result.Unsafe == (eth.L2BlockRef{}) {
			result.Unsafe = n
		}

		if ahead {
			// keep the unsafe head if we can't tell if its L1 origin is canonical or not yet.
		} else if l1Block.Hash == n.L1Origin.Hash {
			// if L2 matches canonical chain, even if unsafe,
			// then we can start finding a span of L1 blocks to cover the sequence window,
			// which may help avoid rewinding the existing safe head unnecessarily.
			if highestL2WithCanonicalL1Origin == (eth.L2BlockRef{}) {
				highestL2WithCanonicalL1Origin = n
			}
		} else {
			// L1 origin not ahead of L1 head nor canonical, discard previous candidate and keep looking.
			result.Unsafe = eth.L2BlockRef{}
			highestL2WithCanonicalL1Origin = eth.L2BlockRef{}
		}

		// If the L2 block is at least as old as the previous safe head, and we have seen at least a full sequence window worth of L1 blocks to confirm
		if n.Number <= result.Safe.Number && n.L1Origin.Number+cfg.SeqWindowSize < highestL2WithCanonicalL1Origin.L1Origin.Number && n.SequenceNumber == 0 {
			ready = true
		}

		// Don't traverse further than the finalized head to find a safe head
		if n.Number == result.Finalized.Number {
			lgr.Info("Hit finalized L2 head, returning immediately", "unsafe", result.Unsafe, "safe", result.Safe,
				"finalized", result.Finalized, "unsafe_origin", result.Unsafe.L1Origin, "safe_origin", result.Safe.L1Origin)
			result.Safe = n
			return result, nil
		}

		// Pull L2 parent for next iteration
		parent, err := l2.L2BlockRefByHash(ctx, n.ParentHash)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch L2 block by hash %v: %w", n.ParentHash, err)
		}

		// Check the L1 origin relation
		if parent.L1Origin != n.L1Origin {
			// sanity check that the L1 origin block number is coherent
			if parent.L1Origin.Number+1 != n.L1Origin.Number {
				return nil, fmt.Errorf("l2 parent %s of %s has L1 origin %s that is not before %s", parent, n, parent.L1Origin, n.L1Origin)
			}
			// sanity check that the later sequence number is 0, if it changed between the L2 blocks
			if n.SequenceNumber != 0 {
				return nil, fmt.Errorf("l2 block %s has parent %s with different L1 origin %s, but non-zero sequence number %d", n, parent, parent.L1Origin, n.SequenceNumber)
			}
			// if the L1 origin is known to be canonical, then the parent must be too
			if l1Block.Hash == n.L1Origin.Hash && l1Block.ParentHash != parent.L1Origin.Hash {
				return nil, fmt.Errorf("parent L2 block %s has origin %s but expected %s", parent, parent.L1Origin, l1Block.ParentHash)
			}
		} else {
			if parent.SequenceNumber+1 != n.SequenceNumber {
				return nil, fmt.Errorf("sequence number inconsistency %d <> %d between l2 blocks %s and %s", parent.SequenceNumber, n.SequenceNumber, parent, n)
			}
		}

		n = parent

		// once we found the block at seq nr 0 that is more than a full seq window behind the common chain post-reorg, then use the parent block as safe head.
		if ready {
			result.Safe = n
			return result, nil
		}
	}
}
