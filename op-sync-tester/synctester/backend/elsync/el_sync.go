package elsync

import (
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var _ eth.ELSyncPolicy = (*WindowSyncPolicy)(nil)

// WindowSyncPolicy implements eth.ELSyncPolicy by maintaining a sliding window
// of recently observed payload numbers (block heights) and determining when the
// execution layer (EL) should be considered fully synced.
//
// Conceptually, the policy tracks the most recent payload numbers in a sorted,
// duplicate-free cache. When a new payload number is reported, it is inserted
// into the cache while removing any entries greater than or equal to it -
// simulating a reorg or an out-of-order unsafe payload insertion. The cache size
// is capped by maxSize.
//
// The EL is considered SYNCING until the following conditions are met:
//  1. At least cnt payload numbers have been observed, and
//  2. The last cnt numbers in the cache form a consecutive sequence ending
//     exactly at the most recent number.
//
// Once both conditions hold, ELSyncStatus(num) returns ExecutionValid.
// Otherwise, it returns ExecutionSyncing.
//
// Example:
//
//	With cnt=3 and maxSize=5, the policy reports SYNCING until it has seen
//	three consecutive payloads (for example 10, 11, 12). After that, it reports
//	VALID for subsequent payloads unless a reorg or out-of-order insertion
//	causes the cache to break the consecutive sequence.
type WindowSyncPolicy struct {
	cache   []uint64
	cnt     uint64
	maxSize uint64
}

func DefaultELSyncPolicy() eth.ELSyncPolicy {
	return NewWindowSyncPolicy(2, 5)
}

func NewWindowSyncPolicy(cnt, maxSize uint64) *WindowSyncPolicy {
	if cnt == 0 || maxSize == 0 {
		panic(fmt.Sprintf("cache max size: %d or window size: %d is not positive", maxSize, cnt))
	}
	if cnt > maxSize {
		panic(fmt.Sprintf("cache max size: %d less than window size: %d", maxSize, cnt))
	}
	return &WindowSyncPolicy{cnt: cnt, maxSize: maxSize}
}

func (e *WindowSyncPolicy) insertNum(num uint64) {
	if len(e.cache) == 0 {
		e.cache = append(e.cache, num)
		return
	}
	maxNum := slices.Max(e.cache)
	if maxNum >= num {
		e.cache = slices.DeleteFunc(e.cache, func(v uint64) bool {
			return v >= num
		})
	}
	e.cache = append(e.cache, num)
	slices.Sort(e.cache)
	if e.maxSize < uint64(len(e.cache)) {
		e.cache = e.cache[1:]
	}
	// Invariant: cache is sorted, no duplicates and size not larger than maxSize
}

func (e *WindowSyncPolicy) ELSyncStatus(num uint64) eth.ExecutePayloadStatus {
	e.insertNum(num)
	if uint64(len(e.cache)) < e.cnt {
		return eth.ExecutionSyncing
	}
	if e.cache[len(e.cache)-1] != num {
		return eth.ExecutionSyncing
	}
	for i := range e.cnt {
		if e.cache[uint64(len(e.cache))-1-i] != num-i {
			return eth.ExecutionSyncing
		}
	}
	return eth.ExecutionValid
}
