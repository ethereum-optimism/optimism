package reads

// Handle represents a read handle, to detect chain rewinds during multi-operation reads.
type Handle interface {
	// DependOnDerivedTime registers the given derived block timestamp as a dependency,
	// updating the current range if needed.
	DependOnDerivedTime(timestamp uint64)
	// DependOnSourceBlock registers the given derived block number as a dependency,
	// updating the current range if needed.
	DependOnSourceBlock(blockNum uint64)

	// Err is a convenience method to return a types.ErrInvalidatedRead whenever the read handle is not valid
	Err() error

	// IsValid inspects the dependencies we have seen so far, and the invalidations we have seen,
	// and determines if the reads thus far are still valid.
	// Once a Handle is invalid, it never becomes valid again.
	IsValid() bool

	// Release releases the read-handle: once released we no longer apply updates to the read handle.
	// rangeHandles must be released, otherwise they are tracked forever.
	Release()

	invalidationTarget
}

type invalidationTarget interface {
	// Used internally, to apply invalidation to a read handle
	invalidateDerived(timestamp uint64)
	// Used internally, to apply invalidation to a read handle
	invalidateSource(blockNum uint64)
}

// Acquirer creates read handles
type Acquirer interface {
	// AcquireHandle creates a new read handle.
	// Once the handle is no longer used, it must be released with readHandle.Release, or memory leaks.
	AcquireHandle() Handle
}

// Invalidator invalidates read handles
type Invalidator interface {
	TryInvalidate(rule InvalidationRule) (release func(), err error)
}

type InvalidationRule interface {
	Apply(h invalidationTarget)
}

type InvalidationRules []InvalidationRule

func (rules InvalidationRules) Apply(h invalidationTarget) {
	for _, r := range rules {
		r.Apply(h)
	}
}

var _ InvalidationRule = (InvalidationRules)(nil)

type DerivedInvalidation struct {
	Timestamp uint64
}

var _ InvalidationRule = DerivedInvalidation{}

func (s DerivedInvalidation) Apply(h invalidationTarget) {
	h.invalidateDerived(s.Timestamp)
}

type SourceInvalidation struct {
	Number uint64
}

var _ InvalidationRule = SourceInvalidation{}

func (s SourceInvalidation) Apply(h invalidationTarget) {
	h.invalidateSource(s.Number)
}
