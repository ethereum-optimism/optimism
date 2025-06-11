package reads

import (
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
)

// NoopHandle implements Handle without offering actual protection.
// This can be used when access is known to be synchronous and safe.
type NoopHandle struct{}

func (n NoopHandle) DependOnDerivedTime(timestamp uint64) {
}

func (n NoopHandle) DependOnSourceBlock(blockNum uint64) {
}

func (n NoopHandle) invalidateDerived(timestamp uint64) {
}

func (n NoopHandle) invalidateSource(blockNum uint64) {
}

func (n NoopHandle) Err() error {
	return nil
}

func (n NoopHandle) IsValid() bool {
	return true
}

func (n NoopHandle) Release() {
}

var _ Handle = NoopHandle{}

// NoopRegistry implements Acquirer and Invalidator without offering actual protection.
// This can be used when access is known to be synchronous and safe.
type NoopRegistry struct{}

func (n NoopRegistry) TryInvalidate(rule InvalidationRule) (release func(), err error) {
	return func() {}, nil
}

func (n NoopRegistry) AcquireHandle() Handle {
	return NoopHandle{}
}

var _ Acquirer = (*NoopRegistry)(nil)
var _ Invalidator = (*NoopRegistry)(nil)

// InvalidHandle is a test-util, to present a read-handle that is always invalid, to mock an inconsistency.
type InvalidHandle struct{}

func (i InvalidHandle) DependOnDerivedTime(timestamp uint64) {
}

func (i InvalidHandle) DependOnSourceBlock(blockNum uint64) {
}

func (i InvalidHandle) invalidateDerived(timestamp uint64) {
}

func (i InvalidHandle) invalidateSource(blockNum uint64) {
}

func (i InvalidHandle) Err() error {
	return types.ErrInvalidatedRead
}

func (i InvalidHandle) IsValid() bool {
	return false
}

func (i InvalidHandle) Release() {}

var _ Handle = InvalidHandle{}

type TestInvalidator struct {
	Invalidated                 bool
	InvalidatedDerivedTimestamp uint64
	InvalidatedSourceNum        uint64
}

func (t *TestInvalidator) invalidateDerived(timestamp uint64) {
	t.Invalidated = true
	t.InvalidatedDerivedTimestamp = timestamp
}

func (t *TestInvalidator) invalidateSource(blockNum uint64) {
	t.Invalidated = true
	t.InvalidatedSourceNum = blockNum
}

func (t *TestInvalidator) TryInvalidate(rule InvalidationRule) (release func(), err error) {
	rule.Apply(t)
	return func() {}, nil
}

var _ Invalidator = (*TestInvalidator)(nil)
