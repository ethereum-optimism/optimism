package confdepth

import (
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// l1HeadBuffer is a cache for L1 block references, which contains a series blocks with a valid chain of parent hashes.
type l1HeadBuffer struct {
	rb             *ringbuffer[eth.L1BlockRef]
	minBlockNumber uint64
}

func newL1HeadBuffer(size int) *l1HeadBuffer {
	return &l1HeadBuffer{rb: newRingBuffer[eth.L1BlockRef](size)}
}

// Get returns the L1 block reference for the given block number, if it exists in the cache.
func (lhb *l1HeadBuffer) Get(num uint64) (eth.L1BlockRef, bool) {
	return lhb.rb.Get(int(num - lhb.minBlockNumber))
}

// Insert inserts a new L1 block reference into the cache, and removes any entries that are invalidated by a reorg.
// If the parent hash of the new head doesn't match the hash of the previous head, all entries after the new head are removed
// as the chain cannot be validated.
func (lhb *l1HeadBuffer) Insert(l1Head eth.L1BlockRef) {
	// First, check if the L1 head is in the cache.
	// If the hash doesn't match the one in the cache, we have a reorg and need to remove all entries after the new head.
	if ref, ok := lhb.Get(l1Head.Number); ok {
		if ref.Hash != l1Head.Hash {
			// Reorg detected, invalidate all entries after the new head.
			for {
				ref, ok := lhb.rb.Pop()
				if !ok || ref.Number == l1Head.Number {
					break
				}
			}
			lhb.rb.Push(ref)
		}
	} else if ref, ok := lhb.Get(l1Head.Number - 1); ok && ref.Hash == l1Head.ParentHash {
		// Parent hash matches, so we can safely add the new head to the cache.
		lhb.rb.Push(l1Head)
	} else {
		// Parent not found or doesn't match, so invalidate the entire cache.
		lhb.rb.Reset()
		lhb.rb.Push(l1Head)
	}

	start, _ := lhb.rb.Start()
	lhb.minBlockNumber = start.Number
}
