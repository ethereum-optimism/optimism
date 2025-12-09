package logsdb

// Invalidator invalidates read handles.
// Minimal interface to allow persistence components to request read invalidation.
type Invalidator interface {
	TryInvalidate(rule InvalidationRule) (release func(), err error)
}

// InvalidationRule describes a single invalidation operation.
type InvalidationRule interface {
	Apply(target invalidationTarget)
}

// InvalidationRules is a list of InvalidationRule.
type InvalidationRules []InvalidationRule

func (rules InvalidationRules) Apply(target invalidationTarget) {
	for _, r := range rules {
		r.Apply(target)
	}
}

// Compile-time check
var _ InvalidationRule = (InvalidationRules)(nil)

// invalidationTarget is implemented by read handles; kept minimal here.
type invalidationTarget interface {
	invalidateDerived(timestamp uint64)
	invalidateSource(blockNum uint64)
}

// DerivedInvalidation invalidates reads depending on a derived timestamp.
type DerivedInvalidation struct {
	Timestamp uint64
}

var _ InvalidationRule = DerivedInvalidation{}

func (s DerivedInvalidation) Apply(target invalidationTarget) {
	target.invalidateDerived(s.Timestamp)
}

// SourceInvalidation invalidates reads depending on a source block number.
type SourceInvalidation struct {
	Number uint64
}

var _ InvalidationRule = SourceInvalidation{}

func (s SourceInvalidation) Apply(target invalidationTarget) {
	target.invalidateSource(s.Number)
}


