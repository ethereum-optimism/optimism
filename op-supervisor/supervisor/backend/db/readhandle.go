package db

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/log"
)

var ErrInvalidHandle = fmt.Errorf("read handle is invalid due to chain reorg")

// Design Rationale:
// This approach was chosen over simpler read-write locks for two main reasons:
// 1. Fine-grained invalidation: Only operations depending on rewound blocks are affected
// 2. Non-blocking reads: Rewinds don't block unrelated read operations
type ReadHandle struct {
	blockNum uint64
	handleID uint64
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
		h.registry.logger.Debug("Read handle validation failed",
			"handleID", h.handleID,
			"blockNum", h.blockNum)
		return ErrInvalidHandle
	}
	return nil
}

type ReadRegistry struct {
	nextHandleID  atomic.Uint64
	activeHandles sync.Map
	logger        log.Logger
}

func NewReadRegistry(logger log.Logger) *ReadRegistry {
	return &ReadRegistry{
		logger: logger,
	}
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
	var invalidated []uint64
	r.activeHandles.Range(func(key, value interface{}) bool {
		handle := value.(*ReadHandle)
		if handle.blockNum >= blockNum {
			handle.valid.Store(false)
			invalidated = append(invalidated, handle.handleID)
		}
		return true
	})

	if len(invalidated) > 0 {
		r.logger.Debug("Invalidated read handles",
			"threshold", blockNum,
			"count", len(invalidated),
			"handleIDs", invalidated)
	}
}

func (r *ReadRegistry) releaseHandle(handleID uint64) {
	r.activeHandles.Delete(handleID)
	r.logger.Trace("Released read handle", "handleID", handleID)
}
