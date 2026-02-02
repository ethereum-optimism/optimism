package dsl

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
)

// Supernode wraps a stack.Supernode interface for DSL operations
type Supernode struct {
	commonImpl
	inner stack.Supernode
}

// NewSupernode creates a new Supernode DSL wrapper
func NewSupernode(inner stack.Supernode) *Supernode {
	return &Supernode{
		commonImpl: commonFromT(inner.T()),
		inner:      inner,
	}
}

func (s *Supernode) ID() stack.SupernodeID {
	return s.inner.ID()
}

func (s *Supernode) String() string {
	return s.inner.ID().String()
}

// Escape returns the underlying stack.Supernode
func (s *Supernode) Escape() stack.Supernode {
	return s.inner
}

// QueryAPI returns the supernode's query API
func (s *Supernode) QueryAPI() apis.SupernodeQueryAPI {
	return s.inner.QueryAPI()
}

// SuperRootAtTimestamp fetches the super-root at the given timestamp
func (s *Supernode) SuperRootAtTimestamp(timestamp uint64) eth.SuperRootAtTimestampResponse {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	resp, err := s.inner.QueryAPI().SuperRootAtTimestamp(ctx, timestamp)
	s.require.NoError(err, "failed to get super-root at timestamp %d", timestamp)
	return resp
}

// AssertSuperRootAtTimestamp asserts that the super-root at the given timestamp matches the expected root claim
func (s *Supernode) AssertSuperRootAtTimestamp(l2SequenceNumber uint64, rootClaim common.Hash) {
	resp := s.SuperRootAtTimestamp(l2SequenceNumber)
	s.require.NotNilf(resp.Data, "super root does not exist at time %d", l2SequenceNumber)
	superRoot := eth.SuperRoot(resp.Data.Super)
	s.require.Equal(superRoot[:], rootClaim[:])
}
