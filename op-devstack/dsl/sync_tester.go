package dsl

import (
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

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

func (s *SyncTester) ListSessions() []string {
	sessionIDs, err := s.inner.API().ListSessions(s.ctx)
	s.t.Require().NoError(err)
	return sessionIDs
}

func (s *SyncTester) GetSession(sessionID string) *eth.SyncTesterSession {
	session, err := s.inner.APIWithSession(sessionID).GetSession(s.ctx)
	s.t.Require().NoError(err)
	return session
}

func (s *SyncTester) DeleteSession(sessionID string) {
	err := s.inner.APIWithSession(sessionID).DeleteSession(s.ctx)
	s.t.Require().NoError(err)
}

func (s *SyncTester) ChainID(sessionID string) eth.ChainID {
	chainID, err := s.inner.APIWithSession(sessionID).ChainID(s.ctx)
	s.t.Require().NoError(err, "should be able to get chain ID from SyncTester")
	return chainID
}
