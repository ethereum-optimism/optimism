// Package sync is responsible for reconciling L1 and L2.
//
// The Ethereum chain is a DAG of blocks with the root block being the genesis block. At any given
// time, the head (or tip) of the chain can change if an offshoot/branch of the chain has a higher
// total difficulty or PoS attestation weight.
// This is known as a re-organization of the canonical chain.
// Each block points to a parent block and the node is responsible for deciding which block is the
// head and thus the mapping from block number to canonical block.
//
// The Optimism (L2) chain has similar properties, but also retains references to the Ethereum (L1)
// chain. Each L2 block retains a reference to an L1 block (its "L1 origin", i.e. L1 block
// associated with the epoch that the L2 block belongs to) and to its parent L2 block.
// The L2 chain node must satisfy the following validity rules:
//
//  1. l2block.number == l2block.l2parent.block.number + 1
//  2. l2block.l1Origin.number == l2block.l2parent.l1Origin.number
//     OR l2block.l1Origin.number == l2block.l2parent.l1Origin.number + 1
//  3. l2block.l1Origin is in the canonical chain on L1
//  4. l1_rollup_genesis is an ancestor of l2block.l1Origin
//
// During normal operation, both the L1 and L2 canonical chains can change, due to a re-organisation
// or due to an extension (new L1 or L2 block).
//
// When one of these changes occurs, the rollup node needs to determine what the new L2 sync status
// should be. We track the following attributes:
//
//   - The *unsafe L2 block*: This is the highest L2 block whose L1 origin is a plausible (1)
//     extension of the canonical L1 chain (as known to the op-node).
//   - The *safe L2 block*: This is the highest L2 block which is certain to be fully derived from the L1 chain.
//     Being derived from the L1 chain requires inclusion in L1, not just references of L1 in L2.
//     Inclusion is guaranteed by rewinding back a full sequence window before the L2 block that has
//     a canonical origin and is before or equal the previous safe L2 block.
//   - The *finalized L2 block*: This is the highest L2 block which is fully derived from finalized L1 data.
//     This block does not change upon a reorg, assuming L1 cannot reorg finalized data.
//     Additionally, the safe block and unsafe block cannot rewind past the finalized L2 block.
//   - The *starting L1 origin*: This is the L1 origin to restart the derivation process at.
//     This may be behind the L1 origin of the safe L2 block,
//     since the safe L2 block cannot be reset further than the finalized L2 block.
//
// (1) Plausible meaning that the blockhash of the L2 block's L1 origin (as reported in the L1)
package sync

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum-optimism/optimism/op-node/rollup"
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

const MaxReorgDepth = 1000

var ReorgFinalizedErr = errors.New("cannot reorg finalized block")
var WrongChainErr = errors.New("wrong chain")
var TooDeepReorgErr = errors.New("reorg is too deep")

type FindSyncStart struct {
	cfg *rollup.Config

	l1 L1Chain
	l2 L2Chain

	finalized eth.L2BlockRef
	safeMaybe eth.L2BlockRef
	safe      eth.L2BlockRef
	unsafe    eth.L2BlockRef

	// currentL2 is used to traverse from unsafe L2 head all the way down to the L2 block that we can fully derive from L1,
	// to use its origin as starting point. Along the traversal we update the other values.
	currentL2 eth.L2BlockRef
	// currentL1 represents the L1 data, when available
	currentL1 eth.L1BlockRef
	// currentL1 may not always exist, so we traverse the chain by needed number,
	// and get the full reference data when we can
	currentL1Needed uint64

	startL1 eth.L1BlockRef
}

func NewFindSyncStart(cfg *rollup.Config, l1 L1Chain, l2 L2Chain) *FindSyncStart {
	return &FindSyncStart{
		cfg: cfg,
		l1:  l1,
		l2:  l2,
	}
}

func (fss *FindSyncStart) Step(ctx context.Context) error {
	if fss.finalized == (eth.L2BlockRef{}) {
		finalized, err := fss.l2.L2BlockRefByLabel(ctx, eth.Finalized)
		if errors.Is(err, ethereum.NotFound) {
			// default to genesis if we have not finalized anything before.
			finalized, err = fss.l2.L2BlockRefByHash(ctx, fss.cfg.Genesis.L2.Hash)
		}
		if err != nil {
			return fmt.Errorf("failed to find the finalized L2 block: %w", err)
		}
		fss.finalized = finalized
		return nil
	}

	if fss.safeMaybe == (eth.L2BlockRef{}) {
		safe, err := fss.l2.L2BlockRefByLabel(ctx, eth.Safe)
		if errors.Is(err, ethereum.NotFound) {
			safe = fss.finalized
		} else if err != nil {
			return fmt.Errorf("failed to find the safe L2 block: %w", err)
		}
		fss.safeMaybe = safe
		return nil
	}

	if fss.unsafe == (eth.L2BlockRef{}) {
		unsafe, err := fss.l2.L2BlockRefByLabel(ctx, eth.Unsafe)
		if err != nil {
			return fmt.Errorf("failed to find the L2 head block: %w", err)
		}
		fss.unsafe = unsafe
		fss.currentL2 = unsafe
		fss.currentL1Needed = unsafe.L1Origin.Number
		return nil
	}

	if fss.currentL1 == (eth.L1BlockRef{}) || fss.currentL1Needed != fss.currentL1.Number {
		currentL1, err := fss.l1.L1BlockRefByNumber(ctx, fss.currentL1Needed)
		if err == ethereum.NotFound {
			if fss.currentL1Needed == fss.cfg.Genesis.L1.Number {
				return fmt.Errorf("rollup data starting at L1 block %s is not available on provided L1 node, maybe the L1 node is syncing", fss.cfg.Genesis.L1)
			}
			// Plausible still, the L1 source may be lagging behind the L1 origins in the L2 unsafe blocks.
			fss.currentL1Needed -= 1
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to get L1 block %d: %w", fss.currentL1Needed, err)
		} else {
			fss.currentL1 = currentL1
			return nil
		}
	}

	// Check traversal against Genesis data
	if (fss.currentL2.Number == fss.cfg.Genesis.L2.Number) && (fss.currentL2.Hash != fss.cfg.Genesis.L2.Hash) {
		return fmt.Errorf("%w L2: genesis: %s, got %s", WrongChainErr, fss.cfg.Genesis.L2, fss.currentL2)
	}

	if (fss.currentL1.Number == fss.cfg.Genesis.L1.Number) && (fss.currentL1.Hash != fss.cfg.Genesis.L1.Hash) {
		return fmt.Errorf("%w L1: genesis: %s, got %s", WrongChainErr, fss.cfg.Genesis.L1, fss.currentL1)
	}

	// Check traversal against finalized data
	if (fss.currentL2.Number == fss.finalized.Number) && (fss.currentL2.Hash != fss.finalized.Hash) {
		return fmt.Errorf("%w: finalized %s, got: %s", ReorgFinalizedErr, fss.finalized, fss.currentL2)
	}

	// Check we are not reorging incredibly deep
	if fss.currentL2.Number+MaxReorgDepth < fss.safe.Number {
		return fmt.Errorf("%w: traversed back to L2 block %s, but too deep compared to previous safe block %s", TooDeepReorgErr, fss.currentL2, fss.safe)
	}

	// if the L2 chain references a yet unknown L1 origin, then traverse back L2 to find a L1 reference within view
	if fss.currentL2.L1Origin.Number > fss.currentL1.Number {
		parentL2, err := fss.l2.L2BlockRefByHash(ctx, fss.currentL2.ParentHash)
		if err != nil {
			return fmt.Errorf("failed to retrieve parent %s of L2 block %s with origin %s to get towards older L1 origin %s: %w",
				fss.currentL2.ParentHash, fss.currentL2, fss.currentL2.L1Origin, fss.currentL1, err)
		}
		fss.currentL2 = parentL2
		fss.currentL1Needed = fss.currentL2.L1Origin.Number
		return nil
	}

	// if the origin of the current block is not canonical, then we have to revert to the parent of this L2 block
	if fss.currentL1.Hash != fss.currentL2.L1Origin.Hash {
		// and traverse back further
		parentL2, err := fss.l2.L2BlockRefByHash(ctx, fss.currentL2.ParentHash)
		if err != nil {
			return fmt.Errorf("failed to retrieve parent %s of L2 block %s to reorg different L1 origin %s (expected %s): %w",
				fss.currentL2.ParentHash, fss.currentL2, fss.currentL2.L1Origin, fss.currentL1, err)
		}
		fss.currentL2 = parentL2
		fss.currentL1Needed = fss.currentL2.L1Origin.Number

		// it not being canonical also means we have to reorg whatever we previously determined as potential L2 chain
		fss.unsafe = parentL2
		if parentL2.Number <= fss.safeMaybe.Number {
			fss.safeMaybe = parentL2
		}
		return nil
	}

	// At this point we've initialized everything, and found a L2 block with known L1 origin.
	// However, we still have to traverse back a full sequence window,
	// to ensure the L2 block not just has the correct origin, but was also derived from L1 data, and thus still safe.
	// The confirmed L2 data on the L1 chain can then be reconciled against the now temporarily unsafe L2 chain,
	// this is not necessarily a reorg.

	// initialize the default the safe head to what we know
	if fss.safe == (eth.L2BlockRef{}) {
		fss.safe = fss.safeMaybe
	}

	// Keep resetting back the safe head until the L1 origin is deep enough to guarantee it was fully derived from L1
	if fss.currentL2.L1Origin.Number+fss.cfg.SeqWindowSize > fss.safeMaybe.L1Origin.Number && fss.currentL2.Number > fss.cfg.Genesis.L2.Number {
		// and traverse back further
		parentL2, err := fss.l2.L2BlockRefByHash(ctx, fss.currentL2.ParentHash)
		if err != nil {
			return fmt.Errorf("failed to retrieve parent %s of L2 block %s with origin %s to find block %d blocks before last safe L1 block %s: %w",
				fss.currentL2.ParentHash, fss.currentL2, fss.currentL2.L1Origin, fss.cfg.SeqWindowSize, fss.safeMaybe.L1Origin, err)
		}
		fss.currentL2 = parentL2
		fss.currentL1Needed = fss.currentL2.L1Origin.Number
		// don't reset back the safe head beyond the finalized head
		if parentL2.Number >= fss.finalized.Number {
			fss.safe = parentL2
		}
		return nil
	}

	if fss.startL1 == (eth.L1BlockRef{}) {
		ref, err := fss.l1.L1BlockRefByHash(ctx, fss.currentL2.L1Origin.Hash)
		if err != nil {
			return fmt.Errorf("failed to find full L1 block contents for L1 starting point: %w", err)
		}
		fss.startL1 = ref
	}

	// finished
	return io.EOF
}

func (fss *FindSyncStart) Result() (finalized, safe, unsafe eth.L2BlockRef, startL1 eth.L1BlockRef) {
	return fss.finalized, fss.safe, fss.unsafe, fss.startL1
}
