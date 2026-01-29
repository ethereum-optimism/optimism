package l2

import (
	"fmt"

	"github.com/ethereum-optimism/optimism/op-service/bigs"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type BlockByHashFn func(hash common.Hash) *types.Block

type CanonicalBlockHeaderOracle struct {
	head                 *types.Header
	hashByNum            map[uint64]common.Hash
	earliestIndexedBlock *types.Header
	blockByHashFn        BlockByHashFn
}

func NewCanonicalBlockHeaderOracle(head *types.Header, blockByHashFn BlockByHashFn) *CanonicalBlockHeaderOracle {
	return &CanonicalBlockHeaderOracle{
		head: head,
		hashByNum: map[uint64]common.Hash{
			bigs.Uint64Strict(head.Number): head.Hash(),
		},
		earliestIndexedBlock: head,
		blockByHashFn:        blockByHashFn,
	}
}

func (o *CanonicalBlockHeaderOracle) CurrentHeader() *types.Header {
	return o.head
}

// GetHeaderByNumber walks back from the current head to the requested block number
func (o *CanonicalBlockHeaderOracle) GetHeaderByNumber(n uint64) *types.Header {
	if bigs.Uint64Strict(o.head.Number) < n {
		return nil
	}

	if bigs.Uint64Strict(o.earliestIndexedBlock.Number) <= n {
		// guaranteed to be cached during lookup
		hash, ok := o.hashByNum[n]
		if !ok {
			panic(fmt.Sprintf("block %v was not indexed when earliest block number is %v", n, o.earliestIndexedBlock.Number))
		}
		return o.blockByHashFn(hash).Header()
	}

	h := o.earliestIndexedBlock
	for bigs.Uint64Strict(h.Number) > n {
		hash := h.ParentHash
		h = o.blockByHashFn(hash).Header()
		o.hashByNum[bigs.Uint64Strict(h.Number)] = hash
	}
	o.earliestIndexedBlock = h
	return h
}

func (o *CanonicalBlockHeaderOracle) SetCanonical(head *types.Header) common.Hash {
	oldHead := o.head
	o.head = head

	// Remove canonical hashes after the new header
	for n := bigs.Uint64Strict(head.Number) + 1; n <= bigs.Uint64Strict(oldHead.Number); n++ {
		delete(o.hashByNum, n)
	}

	// Add new canonical blocks to the block by number cache
	// Since the original head is added to the block number cache and acts as the finalized block,
	// at some point we must reach the existing canonical chain and can stop updating.
	h := o.head
	for {
		newHash := h.Hash()
		prevHash, ok := o.hashByNum[bigs.Uint64Strict(h.Number)]
		if ok && prevHash == newHash {
			// Connected with the existing canonical chain so stop updating
			break
		}
		o.hashByNum[bigs.Uint64Strict(h.Number)] = newHash
		if bigs.Uint64Strict(h.Number) == 0 {
			// Reachable if there aren't any cached blocks at or before the common ancestor
			break
		}
		h = o.blockByHashFn(h.ParentHash).Header()
	}
	o.earliestIndexedBlock = h
	return head.Hash()
}
