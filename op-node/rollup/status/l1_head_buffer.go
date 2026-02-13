package status

import (
	"sync"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// l1HeadBuffer is a thread-safe cache for L1 block references, which contains a series blocks with a valid chain of parent hashes.
type l1HeadBuffer struct {
	rb             *ringbuffer[eth.L1BlockRef]
	minBlockNumber uint64
	mu             sync.RWMutex
}

func newL1HeadBuffer(size int) *l1HeadBuffer {
	return &l1HeadBuffer{rb: newRingBuffer[eth.L1BlockRef](size)}
}

// Get returns the L1 block reference for the given block number, if it exists in the cache.
func (lhb *l1HeadBuffer) Get(num uint64) (eth.L1BlockRef, bool) {
	lhb.mu.RLock()
	defer lhb.mu.RUnlock()

	return lhb.get(num)
}

func (lhb *l1HeadBuffer) get(num uint64) (eth.L1BlockRef, bool) {
	return lhb.rb.Get(int(num - lhb.minBlockNumber))
}

// Insert inserts a new L1 block reference into the cache, and removes any entries that are invalidated by a reorg.
// If the parent hash of the new head doesn't match the hash of the previous head, all entries after the new head are removed
// as the chain cannot be validated.
func (lhb *l1HeadBuffer) Insert(l1Head eth.L1BlockRef) {
	lhb.mu.Lock()
	defer lhb.mu.Unlock()

	if ref, ok := lhb.get(l1Head.Number - 1); ok && ref.Hash == l1Head.ParentHash {
		// Parent hash is found, so we can safely add the new head to the cache after the parent.
		// Remove any L1 refs from the cache after or conflicting with the new head.
		if ref, ok := lhb.rb.End(); ok && ref.Number >= l1Head.Number {
			for ref, ok = lhb.rb.Pop(); ok && ref.Number > l1Head.Number; ref, ok = lhb.rb.Pop() {
			}
		}
	} else {
		// Parent not found or doesn't match, so invalidate the entire cache.
		lhb.rb.Reset()
	}

	lhb.rb.Push(l1Head)

	start, _ := lhb.rb.Start()
	lhb.minBlockNumber = start.Number
}
