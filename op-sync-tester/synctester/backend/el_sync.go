package backend

import (
	"fmt"
	"slices"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

var _ eth.ELSyncPolicy = (*WindowSyncPolicy)(nil)

type WindowSyncPolicy struct {
	cache   []uint64
	cnt     int
	maxSize int
}

func NewWindowSyncPolicy(cnt, maxSize int) *WindowSyncPolicy {
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
	if e.maxSize < len(e.cache) {
		e.cache = e.cache[1:]
	}
	// Invariant: cache is sorted, no duplicates and size not larger then maxSize
}

func (e *WindowSyncPolicy) ELSyncStatus(num uint64) eth.ExecutePayloadStatus {
	e.insertNum(num)
	if len(e.cache) < e.cnt {
		return eth.ExecutionSyncing
	}
	if e.cache[len(e.cache)-1] != num {
		return eth.ExecutionSyncing
	}
	for i := range e.cnt {
		if e.cache[len(e.cache)-1-i] != num-uint64(i) {
			return eth.ExecutionSyncing
		}
	}
	return eth.ExecutionValid
}
