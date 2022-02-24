package sync

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum-optimism/optimistic-specs/opnode/eth"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup"
	"github.com/ethereum-optimism/optimistic-specs/opnode/rollup/derive"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// Sync Source Interface TODOs:
// Relax block by hash???  would have to use by number interface though
// Merge v1 and v2
// Put node definitions in eth?

// SyncSource implements SyncReference with a L2 block sources and L1 hash-by-number source
type SyncSource struct {
	L1 interface {
		HeadBlockLink(ctx context.Context) (self eth.BlockID, parent eth.BlockID, err error)
		BlockLinkByNumber(ctx context.Context, num uint64) (self eth.BlockID, parent eth.BlockID, err error)
	}
	L2 interface {
		BlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error)
		BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	}
}

// RefByL1Num fetches the canonical L1 block hash and the parent for the given L1 block height.
func (src SyncSource) RefByL1Num(ctx context.Context, l1Num uint64) (self eth.BlockID, parent eth.BlockID, err error) {
	return src.L1.BlockLinkByNumber(ctx, l1Num)
}

// L1HeadRef fetches the canonical L1 block hash and the parent for the head of the L1 chain.
func (src SyncSource) L1HeadRef(ctx context.Context) (self eth.BlockID, parent eth.BlockID, err error) {
	return src.L1.HeadBlockLink(ctx)
}

// RefByL2Num fetches the L1 and L2 block IDs from the engine for the given L2 block height.
// Use a nil height to fetch the head.
func (src SyncSource) RefByL2Num(ctx context.Context, l2Num *big.Int, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error) {
	refL2Block, err2 := src.L2.BlockByNumber(ctx, l2Num) // nil for latest block
	if err2 != nil {
		err = fmt.Errorf("failed to retrieve L2 block: %v", err2)
		return
	}
	return derive.BlockReferences(refL2Block, genesis)
}

// RefByL2Hash fetches the L1 and L2 block IDs from the engine for the given L2 block hash.
func (src SyncSource) RefByL2Hash(ctx context.Context, l2Hash common.Hash, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error) {
	refL2Block, err2 := src.L2.BlockByHash(ctx, l2Hash)
	if err2 != nil {
		err = fmt.Errorf("failed to retrieve L2 block: %v", err2)
		return
	}
	return derive.BlockReferences(refL2Block, genesis)
}

// SyncReference helps inform the sync algorithm of the L2 sync-state and L1 canonical chain
type SyncReference interface {
	// RefByL1Num fetches the canonical L1 block hash and the parent for the given L1 block height.
	RefByL1Num(ctx context.Context, l1Num uint64) (self eth.BlockID, parent eth.BlockID, err error)

	// L1HeadRef fetches the canonical L1 block hash and the parent for the head of the L1 chain.
	L1HeadRef(ctx context.Context) (self eth.BlockID, parent eth.BlockID, err error)

	// RefByL2Num fetches the L1 and L2 block IDs from the engine for the given L2 block height.
	// Use a nil height to fetch the head.
	RefByL2Num(ctx context.Context, l2Num *big.Int, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error)

	// RefByL2Hash fetches the L1 and L2 block IDs from the engine for the given L2 block hash.
	RefByL2Hash(ctx context.Context, l2Hash common.Hash, genesis *rollup.Genesis) (refL1 eth.BlockID, refL2 eth.BlockID, parentL2 common.Hash, err error)
}

// SyncReferenceV2 wraps the return types of SyncReference to be easier to work with.
type SyncReferenceV2 interface {
	L1NodeByNumber(ctx context.Context, l1Num uint64) (L1Node, error)
	L1HeadNode(ctx context.Context) (L1Node, error)
	L2NodeByNumber(ctx context.Context, l2Num *big.Int, genesis *rollup.Genesis) (L2Node, error)
	L2NodeByHash(ctx context.Context, l2Hash common.Hash, genesis *rollup.Genesis) (L2Node, error)
}

type SyncSourceV2 struct {
	SyncReference
}

func (src SyncSourceV2) L1NodeByNumber(ctx context.Context, l1Num uint64) (L1Node, error) {
	self, parent, err := src.RefByL1Num(ctx, l1Num)
	if err != nil {
		return L1Node{}, err
	}
	return L1Node{self: self, parent: parent}, nil

}

func (src SyncSourceV2) L1HeadNode(ctx context.Context) (L1Node, error) {
	self, parent, err := src.L1HeadRef(ctx)
	if err != nil {
		return L1Node{}, err
	}
	return L1Node{self: self, parent: parent}, nil
}

func (src SyncSourceV2) L2NodeByNumber(ctx context.Context, l2Num *big.Int, genesis *rollup.Genesis) (L2Node, error) {
	l1Ref, l2ref, l2ParentHash, err := src.RefByL2Num(ctx, l2Num, genesis)
	if err != nil {
		return L2Node{}, err
	}
	return L2Node{self: l2ref, l1parent: l1Ref, l2parent: eth.BlockID{Hash: l2ParentHash, Number: l2ref.Number - 1}}, nil
}

func (src SyncSourceV2) L2NodeByHash(ctx context.Context, l2Hash common.Hash, genesis *rollup.Genesis) (L2Node, error) {
	l1Ref, l2ref, l2ParentHash, err := src.RefByL2Hash(ctx, l2Hash, genesis)
	if err != nil {
		return L2Node{}, err
	}
	return L2Node{self: l2ref, l1parent: l1Ref, l2parent: eth.BlockID{Hash: l2ParentHash, Number: l2ref.Number - 1}}, nil
}

type L2Node struct {
	self     eth.BlockID
	l2parent eth.BlockID
	l1parent eth.BlockID
}

type L1Node struct {
	self   eth.BlockID
	parent eth.BlockID
}
