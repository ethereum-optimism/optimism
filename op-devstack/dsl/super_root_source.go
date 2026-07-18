package dsl

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// SuperRootSource is the super-root query surface the dispute-game DSL needs. It is
// satisfied by both op-supernode (interop, multi-chain) and op-node (non-interop,
// single-chain), letting the same dispute-game tests run against either source.
type SuperRootSource interface {
	QueryAPI() apis.SupernodeQueryAPI
	SuperRootAtTimestamp(timestamp uint64) eth.SuperRootAtTimestampResponse
	AssertSuperRootAtTimestamp(l2SequenceNumber uint64, rootClaim common.Hash)
	AwaitValidatedTimestamp(timestamp uint64)
	AwaitFullyProcessedL1(targetL1 uint64)
}

var (
	_ SuperRootSource = (*Supernode)(nil)
	_ SuperRootSource = (*SuperRootQuerier)(nil)
)

// SuperRootQuerier is a SuperRootSource backed by a fixed apis.SupernodeQueryAPI. It is
// used directly as the op-node-backed source (see NewOpNodeSuperRoots) and embedded by
// Supernode, which adds op-supernode-only controls on top.
type SuperRootQuerier struct {
	commonImpl
	api apis.SupernodeQueryAPI
}

// NewOpNodeSuperRoots wraps a single op-node's superroot_atTimestamp endpoint as a
// super-root source. Only superroot_atTimestamp is served by op-node; SyncStatus
// (supernode_syncStatus) is not available on this source.
func NewOpNodeSuperRoots(cl *L2CLNode) *SuperRootQuerier {
	return &SuperRootQuerier{
		commonImpl: commonFromT(cl.t),
		api:        sources.NewSuperNodeClient(cl.inner.ClientRPC()),
	}
}

// QueryAPI returns the super-root query API for this source.
func (s *SuperRootQuerier) QueryAPI() apis.SupernodeQueryAPI {
	return s.api
}

// SuperRootAtTimestamp fetches the super-root at the given timestamp.
func (s *SuperRootQuerier) SuperRootAtTimestamp(timestamp uint64) eth.SuperRootAtTimestampResponse {
	ctx, cancel := context.WithTimeout(s.ctx, DefaultTimeout)
	defer cancel()
	resp, err := s.api.SuperRootAtTimestamp(ctx, timestamp)
	s.require.NoError(err, "failed to get super-root at timestamp %d", timestamp)
	return resp
}

// AssertSuperRootAtTimestamp asserts that the super-root at the given timestamp matches the expected root claim.
func (s *SuperRootQuerier) AssertSuperRootAtTimestamp(l2SequenceNumber uint64, rootClaim common.Hash) {
	resp := s.SuperRootAtTimestamp(l2SequenceNumber)
	s.require.NotNilf(resp.Data, "super root does not exist at time %d", l2SequenceNumber)
	superRoot := eth.SuperRoot(resp.Data.Super)
	s.require.Equal(superRoot[:], rootClaim[:])
}

// AwaitValidatedTimestamp waits for the super-root at the given timestamp to be available.
func (s *SuperRootQuerier) AwaitValidatedTimestamp(timestamp uint64) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		resp, err := s.api.SuperRootAtTimestamp(ctx, timestamp)
		if err != nil {
			return false, nil // Ignore transient errors.
		}
		return resp.Data != nil, nil
	})
	s.require.NoError(err, "super-root at timestamp %d was not validated in time", timestamp)
}

// AwaitFullyProcessedL1 waits until the source has fully processed the given L1 block
// number. SuperRootAtTimestamp's CurrentL1 names the block currently being processed
// (L1[<CurrentL1] is fully processed), so this returns once CurrentL1.Number > targetL1.
func (s *SuperRootQuerier) AwaitFullyProcessedL1(targetL1 uint64) {
	ctx, cancel := context.WithTimeout(s.ctx, 5*DefaultTimeout)
	defer cancel()
	err := wait.For(ctx, 1*time.Second, func() (bool, error) {
		resp, err := s.api.SuperRootAtTimestamp(ctx, uint64(time.Now().Unix()))
		if err != nil {
			return false, nil // Ignore transient errors.
		}
		return resp.CurrentL1.Number > targetL1, nil
	})
	s.require.NoError(err, "source did not fully process L1 block %d in time", targetL1)
}
