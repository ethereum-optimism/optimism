package reads

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/log"

	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// readHandle maintains a lower and upper range view over a chain.
// This is invalidated through the Registry whenever the chain changes.
type readHandle struct {

	// highestDerivedTimestamp: highest timestamp that this read depends on.
	// Set to block number that the read handle depends on.
	// This may be changed at any time, to adjust the read-handle.
	// If nil, then there is no dependency on derived time.
	highestDerivedTimestamp *uint64

	// lowestInvalidatedDerivedTimestamp: lowest timestamp that was invalidated while this read handle was active.
	// If nil, then nothing was invalidated.
	lowestInvalidatedDerivedTimestamp *uint64

	highestSourceNum           *uint64
	lowestInvalidatedSourceNum *uint64

	// Note: we may add a lowestDependencyNum and a highestPrunedNum in the future, to handle DB pruning.

	// the read handle is locked when inspecting it and making changes to it
	mu sync.Mutex

	registry *Registry
}

func (h *readHandle) DependOnDerivedTime(timestamp uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.registry == nil {
		panic(fmt.Errorf("cannot depend on derived timestamp %d, read-handle has already been released", timestamp))
	}
	prev := h.highestDerivedTimestamp
	if prev == nil || timestamp > *prev {
		h.highestDerivedTimestamp = &timestamp
	}
}

func (h *readHandle) DependOnSourceBlock(blockNum uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.registry == nil {
		panic(fmt.Errorf("cannot depend on source block %d, read-handle has already been released", blockNum))
	}
	prev := h.highestSourceNum
	if prev == nil || blockNum > *prev {
		h.highestSourceNum = &blockNum
	}
}

func (h *readHandle) invalidateDerived(timestamp uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.registry == nil { // if already released, then we can ignore the invalidation signal
		return
	}
	prev := h.lowestInvalidatedDerivedTimestamp
	if prev == nil || timestamp < *prev {
		h.lowestInvalidatedDerivedTimestamp = &timestamp
	}
}

func (h *readHandle) invalidateSource(blockNum uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.registry == nil { // if already released, then we can ignore the invalidation signal
		return
	}
	prev := h.lowestInvalidatedSourceNum
	if prev == nil || blockNum < *prev {
		h.lowestInvalidatedSourceNum = &blockNum
	}
}

var _ Handle = (*readHandle)(nil)

// Err is a convenience method to return a types.ErrInvalidatedRead whenever the read handle is not valid
func (h *readHandle) Err() error {
	if h.IsValid() {
		return nil
	}
	return types.ErrInvalidatedRead
}

// IsValid inspects the dependencies we have seen so far, and the invalidations we have seen,
// and determines if the reads thus far are still valid.
func (h *readHandle) IsValid() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if hi, lo := h.highestDerivedTimestamp, h.lowestInvalidatedDerivedTimestamp; hi != nil && lo != nil && *lo <= *hi {
		return false
	}
	if hi, lo := h.highestSourceNum, h.lowestInvalidatedSourceNum; hi != nil && lo != nil && *lo <= *hi {
		return false
	}
	// If nothing we depend on was invalidated,
	// then this read-handle is still considered valid.
	return true
}

// Release releases the read-handle: once released we no longer apply updates to the read handle.
// rangeHandles must be released, otherwise they are tracked forever.
func (h *readHandle) Release() {
	defer h.registry.releaseHandle(h) // release from registry once readHandle is no longer locked
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.registry == nil {
		panic("read-handle was already released")
	}
	h.registry = nil
}

// Registry maintains a set of read handles (readHandle).
// Whenever the chain reorgs, the active handles are updated, to track what has been reorged.
// Any handles that have an overlapping usage and invalidation range are invalid and return an error.
type Registry struct {
	mu sync.Mutex

	activeHandles map[*readHandle]struct{}
	invalidating  InvalidationRule

	logger log.Logger
}

var _ Acquirer = (*Registry)(nil)
var _ Invalidator = (*Registry)(nil)

func NewRegistry(logger log.Logger) *Registry {
	return &Registry{
		logger:        logger,
		activeHandles: make(map[*readHandle]struct{}),
	}
}

// AcquireHandle creates a new read handle.
// Once the handle is no longer used, it must be released with readHandle.Release, or memory leaks.
func (r *Registry) AcquireHandle() Handle {
	r.mu.Lock()
	defer r.mu.Unlock()
	handle := &readHandle{registry: r}
	if r.invalidating != nil {
		r.invalidating.Apply(handle)
	}
	r.activeHandles[handle] = struct{}{}
	return handle
}

// TryInvalidate invalidates all handles that depend on blocks with numbers >= blockNum.
// Any new read-handles will automatically be invalidated if they depend on blockNum or later.
// The invalidation of new reads (starting at the given number) will stop once it has been released by calling release.
func (r *Registry) TryInvalidate(rule InvalidationRule) (release func(), err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.invalidating != nil {
		return nil, types.ErrAlreadyInvalidatingRead
	}
	r.invalidating = rule
	for handle := range r.activeHandles { // invalidate all existing handles
		rule.Apply(handle)
	}
	return r.releaseInvalidator, nil
}

// releaseInvalidator is returned by TryInvalidate to release the invalidation rule
func (r *Registry) releaseInvalidator() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.invalidating == nil {
		panic("not currently invalidating")
	}
	r.invalidating = nil
}

// releaseHandle is called by Handle.Release() to unregister itself
func (r *Registry) releaseHandle(handle *readHandle) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.activeHandles, handle)
	r.logger.Trace("Released read handle", "handle", handle, "valid", handle.IsValid())
}
