package proofs

import (
	"time"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/dsl"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// SuperRootsSource is a minimal abstraction for "super-roots" needed by the proofs DSL.
// It can be backed by either an op-supervisor (legacy) or an op-supernode (replacement).
type SuperRootsSource interface {
	// SafeTimestamp retrieves the current safe timestamp from the source
	SafeTimestamp() uint64
	// AwaitMinVerifiedTimestamp blocks until the source has verified at least up to the given timestamp
	AwaitMinVerifiedTimestamp(timestamp uint64)
	// SuperV1AtTimestamp retrieves the super root at the given timestamp
	SuperV1AtTimestamp(timestamp uint64) *eth.SuperV1
}

func NewSuperRootsFromSupervisor(supervisor *dsl.Supervisor) SuperRootsSource {
	return &supervisorSuperRootsSource{supervisor: supervisor}
}

func NewSuperRootsFromSupernode(t devtest.T, supernode stack.Supernode) SuperRootsSource {
	return &supernodeSuperRootsSource{
		t:         t,
		require:   require.New(t),
		supernode: supernode,
	}
}

type supervisorSuperRootsSource struct {
	supervisor *dsl.Supervisor
}

func (s *supervisorSuperRootsSource) SafeTimestamp() uint64 {
	return s.supervisor.FetchSyncStatus().SafeTimestamp
}

func (s *supervisorSuperRootsSource) AwaitMinVerifiedTimestamp(timestamp uint64) {
	s.supervisor.AwaitMinCrossSafeTimestamp(timestamp)
}

func (s *supervisorSuperRootsSource) SuperV1AtTimestamp(timestamp uint64) *eth.SuperV1 {
	super, err := s.supervisor.FetchSuperRootAtTimestamp(timestamp).ToSuper()
	s.supervisor.Escape().T().Require().NoError(err, "Failed to parse super root at timestamp %v", timestamp)
	superV1, ok := super.(*eth.SuperV1)
	s.supervisor.Escape().T().Require().Truef(ok, "Unsupported super type %T", super)
	return superV1
}

type supernodeSuperRootsSource struct {
	t         devtest.T
	require   *require.Assertions
	supernode stack.Supernode
}

func (s *supernodeSuperRootsSource) SafeTimestamp() uint64 {
	s.t.Error("not yet implemented: supernodeSuperRootsSource.SafeTimestamp")
	s.t.FailNow()
	return 0
}

func (s *supernodeSuperRootsSource) AwaitMinVerifiedTimestamp(timestamp uint64) {
	s.t.Require().Eventually(func() bool {
		resp, err := s.supernode.QueryAPI().SuperRootAtTimestamp(s.t.Ctx(), timestamp)
		s.require.NoError(err, "Failed to fetch supernode status (superroot_atTimestamp)")
		return resp.Data != nil
	}, 2*time.Minute, 1*time.Second)
}

func (s *supernodeSuperRootsSource) SuperV1AtTimestamp(timestamp uint64) *eth.SuperV1 {
	var resp eth.SuperRootAtTimestampResponse
	s.t.Require().Eventually(func() bool {
		r, err := s.supernode.QueryAPI().SuperRootAtTimestamp(s.t.Ctx(), timestamp)
		if err != nil {
			return false
		}
		resp = r
		return resp.Data != nil
	}, 2*time.Minute, 1*time.Second)
	s.require.NotNil(resp.Data, "Super root data must be present at timestamp %v", timestamp)
	superV1, ok := resp.Data.Super.(*eth.SuperV1)
	s.require.Truef(ok, "Unsupported super type %T", resp.Data.Super)
	return superV1
}
