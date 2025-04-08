package db

import (
	"fmt"
	"sync"
	"sync/atomic"
)

var ErrInvalidHandle = fmt.Errorf("read handle is invalid due to chain reorg")

// Design Rationale:
// This approach was chosen over simpler read-write locks for several reasons:
// 1. Fine-grained invalidation: Only operations depending on rewound blocks are affected
// 2. Non-blocking reads: Rewinds don't block unrelated read operations 
// 3. Cross-chain consistency: Handles from multiple chains can be validated together
// 4. Better concurrency: Atomic operations instead of locks provide higher throughput
// 5. Explicit validation: Operations can choose when to verify consistency
//
// Why not just use ValidateBlocksDidntChange?
// While a simple block validation at the end of each query would be conceptually simpler,
// the handle approach provides built-in tracking of dependencies across chains with less code
// duplication. It also enables a more reactive programming model where invalidation happens
// proactively rather than being discovered only at validation time.
type ReadHandle struct {
	blockNum uint64
	handleID uint64 // Unique ID for debugging and tracking
	valid    atomic.Bool
	registry *ReadRegistry
}

func (h *ReadHandle) IsValid() bool {
	return h.valid.Load()
}

func (h *ReadHandle) UpdateBlock(blockNum uint64) bool {
	if !h.valid.Load() {
		return false
	}
	h.blockNum = blockNum
	return true
}

func (h *ReadHandle) Release() {
	if h.registry != nil {
		h.registry.releaseHandle(h.handleID)
	}
}

func (h *ReadHandle) Validate() error {
	if !h.valid.Load() {
		return ErrInvalidHandle
	}
	return nil
}

type ReadRegistry struct {
	nextHandleID  atomic.Uint64
	activeHandles sync.Map
}

func NewReadRegistry() *ReadRegistry {
	return &ReadRegistry{}
}

func (r *ReadRegistry) AcquireHandle(blockNum uint64) *ReadHandle {
	handle := &ReadHandle{
		blockNum: blockNum,
		handleID: r.nextHandleID.Add(1),
		registry: r,
	}
	handle.valid.Store(true)
	r.activeHandles.Store(handle.handleID, handle)
	return handle
}

// InvalidateHandlesAfter invalidates all handles that depend on blocks with numbers >= blockNum
func (r *ReadRegistry) InvalidateHandlesAfter(blockNum uint64) {
	r.activeHandles.Range(func(key, value interface{}) bool {
		handle := value.(*ReadHandle)
		if handle.blockNum >= blockNum {
			handle.valid.Store(false)
		}
		return true
	})
}

func (r *ReadRegistry) releaseHandle(handleID uint64) {
	r.activeHandles.Delete(handleID)
}
