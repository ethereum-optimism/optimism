package dsl

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-devstack/stack"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	opclient "github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

// The point of these tests is that nothing below imports op-supernode. The
// interop test-control surface the DSL drives supernodes through is satisfied
// here by a plain struct holding plain data, which is exactly what a supernode
// in another process would do behind an RPC client. If this file ever needs a
// real supernode to compile, the surface has regressed to an in-process handle.

func TestSupernodePauseAndResumeGoThroughTheTestAPI(t *testing.T) {
	sn, ctrl := newStubSupernode(t)

	sn.PauseInterop(4200)
	require.Equal(t, uint64(4200), ctrl.api.pausedAt.Load())

	sn.ResumeInterop()
	require.Equal(t, uint64(0), ctrl.api.pausedAt.Load())
}

func TestSupernodeStatusReadsComeFromOneSnapshot(t *testing.T) {
	sn, ctrl := newStubSupernode(t)
	ctrl.api.status = eth.SupernodeInteropStatus{
		BackfillAttempts:           3,
		BackfillCompleted:          true,
		ActivationTimestamp:        1000,
		VerificationStartTimestamp: 1200,
		FirstVerifiableTimestamp:   1300,
	}

	require.Equal(t, int32(3), sn.BackfillAttempts())
	require.Equal(t, uint64(1000), sn.ActivationTimestamp())
	require.Equal(t, uint64(1200), sn.VerificationStartTimestamp())

	sn.AwaitBackfillCompleted()
	sn.AwaitVerificationStartsAt(1200)
	sn.AwaitBackfillAttempts(3)
}

func TestSupernodeAwaitBackfillCompletedPolls(t *testing.T) {
	sn, ctrl := newStubSupernode(t)
	ctrl.api.status.BackfillCompleted = false
	ctrl.api.onStatus = func(calls int) {
		if calls >= 2 {
			ctrl.api.status.BackfillCompleted = true
			ctrl.api.status.VerificationStartTimestamp = 777
		}
	}

	sn.AwaitVerificationStartsAt(777)
	require.GreaterOrEqual(t, ctrl.api.statusCalls.Load(), int32(2))
}

func TestSupernodeAssertBackfillCoversUsesSealedBlockRange(t *testing.T) {
	sn, ctrl := newStubSupernode(t)
	chain := eth.ChainIDFromUInt64(901)
	ctrl.api.status = eth.SupernodeInteropStatus{
		BackfillCompleted:        true,
		ActivationTimestamp:      1000,
		FirstVerifiableTimestamp: 1100,
	}
	// Activation at 1000, depth 60s, latest seal at 1090: the expected lower
	// bound is the activation timestamp, since the chain is younger than depth.
	ctrl.api.sealed = map[eth.ChainID]eth.SupernodeSealedBlocks{
		chain: {
			First:     eth.SupernodeSealedBlock{ID: eth.BlockID{Hash: common.Hash{0x1}, Number: 10}, Timestamp: 1000},
			Latest:    eth.SupernodeSealedBlock{ID: eth.BlockID{Hash: common.Hash{0x2}, Number: 55}, Timestamp: 1090},
			HasBlocks: true,
		},
	}

	sn.AssertBackfillCovers(60*time.Second, 2, chain)
	require.Equal(t, int32(1), ctrl.api.sealedCalls.Load())
}

// --- stubs -----------------------------------------------------------------

func newStubSupernode(t *testing.T) (*Supernode, *stubTestControl) {
	dt := devtest.SerialT(t)
	ctrl := &stubTestControl{api: &stubInteropTestAPI{}}
	return NewSupernodeWithTestControl(&stubStackSupernode{t: dt}, ctrl), ctrl
}

type stubTestControl struct {
	api *stubInteropTestAPI
}

var _ stack.SupernodeTestControl = (*stubTestControl)(nil)

func (c *stubTestControl) InteropTestAPI() apis.SupernodeInteropTestAPI { return c.api }
func (c *stubTestControl) RestartWithFreshDataDir() error               { return nil }
func (c *stubTestControl) Stop()                                        {}
func (c *stubTestControl) Start()                                       {}

type stubInteropTestAPI struct {
	pausedAt    atomic.Uint64
	status      eth.SupernodeInteropStatus
	statusCalls atomic.Int32
	// onStatus, when set, runs before each status read is answered and is
	// passed the number of reads so far, so a test can let the verifier
	// "progress" between polls.
	onStatus    func(calls int)
	sealed      map[eth.ChainID]eth.SupernodeSealedBlocks
	sealedCalls atomic.Int32
}

var _ apis.SupernodeInteropTestAPI = (*stubInteropTestAPI)(nil)

func (a *stubInteropTestAPI) PauseInterop(_ context.Context, ts uint64) error {
	a.pausedAt.Store(ts)
	return nil
}

func (a *stubInteropTestAPI) ResumeInterop(_ context.Context) error {
	a.pausedAt.Store(0)
	return nil
}

func (a *stubInteropTestAPI) InteropStatus(_ context.Context) (eth.SupernodeInteropStatus, error) {
	calls := int(a.statusCalls.Add(1))
	if a.onStatus != nil {
		a.onStatus(calls)
	}
	return a.status, nil
}

func (a *stubInteropTestAPI) InteropSealedBlocks(_ context.Context, chainID eth.ChainID) (eth.SupernodeSealedBlocks, error) {
	a.sealedCalls.Add(1)
	sealed, ok := a.sealed[chainID]
	if !ok {
		return eth.SupernodeSealedBlocks{}, errors.New("unknown chain")
	}
	return sealed, nil
}

type stubStackSupernode struct {
	t devtest.T
}

var _ stack.Supernode = (*stubStackSupernode)(nil)

func (s *stubStackSupernode) T() devtest.T                     { return s.t }
func (s *stubStackSupernode) Logger() log.Logger               { return s.t.Logger() }
func (s *stubStackSupernode) Name() string                     { return "stub-supernode" }
func (s *stubStackSupernode) Label(string) string              { return "" }
func (s *stubStackSupernode) SetLabel(string, string)          {}
func (s *stubStackSupernode) ClientRPC() opclient.RPC          { return nil }
func (s *stubStackSupernode) QueryAPI() apis.SupernodeQueryAPI { return nil }
func (s *stubStackSupernode) UserRPC() string                  { return "" }
