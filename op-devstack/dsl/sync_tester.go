package dsl

import "github.com/ethereum-optimism/optimism/op-devstack/stack"

// SyncTester wraps a stack.SyncTester interface for DSL operations
type SyncTester struct {
	commonImpl
	inner stack.SyncTester
}

// NewSyncTester creates a new Sync Tester DSL wrapper
func NewSyncTester(inner stack.SyncTester) *SyncTester {
	return &SyncTester{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

// Escape returns the underlying stack.SyncTester
func (s *SyncTester) Escape() stack.SyncTester {
	return s.inner
}
