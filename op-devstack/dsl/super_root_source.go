package dsl

import (
	"context"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-e2e/e2eutils/wait"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	serviceclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/sources"
)

// SuperRootSource is the super-root query surface the dispute-game DSL needs. It is
// satisfied by both op-supernode (interop, multi-chain) and op-node (non-interop,
// single-chain), letting the same dispute-game tests run against either source.
type SuperRootSource interface {
	QueryAPI() apis.SupernodeQueryAPI
	UserRPC() string
	SuperRootAtTimestamp(timestamp uint64) eth.SuperRootAtTimestampResponse
	AssertSuperRootAtTimestamp(l2SequenceNumber uint64, rootClaim common.Hash)
	AwaitValidatedTimestamp(timestamp uint64)
	AwaitFullyProcessedL1(targetL1 uint64)
}

var (
	_ SuperRootSource = (*Supernode)(nil)
	_ SuperRootSource = (*SuperRootQuerier)(nil)
	_ SuperRootSource = (*LagControlledSuperRootSource)(nil)
)

// SuperRootQuerier is a SuperRootSource backed by a fixed apis.SupernodeQueryAPI. It is
// used directly as the op-node-backed source (see NewOpNodeSuperRoots) and embedded by
// Supernode, which adds op-supernode-only controls on top.
type SuperRootQuerier struct {
	commonImpl
	api     apis.SupernodeQueryAPI
	userRPC string
}

// NewOpNodeSuperRoots wraps a single op-node's superroot_atTimestamp endpoint as a
// super-root source. Only superroot_atTimestamp is served by op-node; SyncStatus
// (supernode_syncStatus) is not available on this source.
func NewOpNodeSuperRoots(cl *L2CLNode) *SuperRootQuerier {
	return &SuperRootQuerier{
		commonImpl: commonFromT(cl.t),
		api:        sources.NewSuperNodeClient(cl.inner.ClientRPC()),
		userRPC:    cl.inner.UserRPC(),
	}
}

// QueryAPI returns the super-root query API for this source.
func (s *SuperRootQuerier) QueryAPI() apis.SupernodeQueryAPI {
	return s.api
}

// UserRPC returns the super-root source's RPC endpoint.
func (s *SuperRootQuerier) UserRPC() string {
	return s.userRPC
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

// LagControlledSuperRootSource wraps a live super-root source with an RPC endpoint
// whose CurrentL1 can be held back while all other response data keeps updating.
type LagControlledSuperRootSource struct {
	*SuperRootQuerier

	upstream SuperRootSource
	heldL1   eth.BlockID

	releaseOnce sync.Once
	released    atomic.Bool

	closeOnce sync.Once
	client    *sources.SuperNodeClient
	rpcServer *rpc.Server
	http      *httptest.Server
}

type lagControlledSuperRootAPI struct {
	source *LagControlledSuperRootSource
}

type lagControlledSupernodeAPI struct {
	source *LagControlledSuperRootSource
}

// NewLagControlledSuperRootSource captures and serves the upstream CurrentL1 as
// the visible progress boundary until Release is called.
func NewLagControlledSuperRootSource(t devtest.T, upstream SuperRootSource) *LagControlledSuperRootSource {
	t.Require().NotNil(upstream, "upstream super-root source is required")
	common := commonFromT(t)

	ctx, cancel := context.WithTimeout(t.Ctx(), 5*DefaultTimeout)
	defer cancel()
	var heldL1 eth.BlockID
	var lastReadErr error
	err := wait.For(ctx, time.Second, func() (bool, error) {
		resp, err := upstream.QueryAPI().SuperRootAtTimestamp(ctx, uint64(time.Now().Unix()))
		if err != nil {
			lastReadErr = err
			t.Logf("Super-root source progress unavailable while establishing lag: %v", err)
			return false, nil
		}
		heldL1 = resp.CurrentL1
		lastReadErr = nil
		return true, nil
	})
	t.Require().NoErrorf(err, "could not establish lag-controlled L1 boundary; last read error: %v", lastReadErr)
	source := &LagControlledSuperRootSource{
		upstream: upstream,
		heldL1:   heldL1,
	}
	source.rpcServer = rpc.NewServer()
	t.Require().NoError(source.rpcServer.RegisterName("superroot", &lagControlledSuperRootAPI{source: source}))
	t.Require().NoError(source.rpcServer.RegisterName("supernode", &lagControlledSupernodeAPI{source: source}))
	source.http = httptest.NewServer(source.rpcServer)
	t.Cleanup(source.Close)

	rpcClient, err := rpc.DialHTTP(source.http.URL)
	t.Require().NoError(err, "failed to connect to lag-controlled super-root source")
	source.client = sources.NewSuperNodeClient(serviceclient.NewBaseRPCClient(rpcClient))
	source.SuperRootQuerier = &SuperRootQuerier{
		commonImpl: common,
		api:        source.client,
		userRPC:    source.http.URL,
	}
	return source
}

// Release waits until the upstream has strictly progressed beyond the held
// block, then atomically exposes its live CurrentL1.
func (s *LagControlledSuperRootSource) Release() {
	s.releaseOnce.Do(func() {
		s.upstream.AwaitFullyProcessedL1(s.heldL1.Number)
		s.released.Store(true)
	})
}

// Close shuts down the local RPC client and server. It is safe to call more
// than once; test cleanup calls it automatically.
func (s *LagControlledSuperRootSource) Close() {
	s.closeOnce.Do(func() {
		if s.client != nil {
			s.client.Close()
		}
		if s.rpcServer != nil {
			s.rpcServer.Stop()
		}
		if s.http != nil {
			s.http.Close()
		}
	})
}

func (a *lagControlledSuperRootAPI) AtTimestamp(ctx context.Context, timestamp hexutil.Uint64) (eth.SuperRootAtTimestampResponse, error) {
	resp, err := a.source.upstream.QueryAPI().SuperRootAtTimestamp(ctx, uint64(timestamp))
	if err != nil {
		return eth.SuperRootAtTimestampResponse{}, err
	}
	if !a.source.released.Load() {
		resp.CurrentL1 = a.source.heldL1
	}
	return resp, nil
}

func (a *lagControlledSupernodeAPI) SyncStatus(ctx context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	resp, err := a.source.upstream.QueryAPI().SyncStatus(ctx)
	if err != nil {
		return eth.SuperNodeSyncStatusResponse{}, err
	}
	if !a.source.released.Load() {
		resp.CurrentL1 = a.source.heldL1
	}
	return resp, nil
}
