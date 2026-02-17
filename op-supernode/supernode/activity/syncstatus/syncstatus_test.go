package syncstatus

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// mockChainContainer implements cc.ChainContainer with only SyncStatus wired up.
type mockChainContainer struct {
	id         eth.ChainID
	syncStatus *eth.SyncStatus
	syncErr    error
}

var _ cc.ChainContainer = (*mockChainContainer)(nil)

func (m *mockChainContainer) Start(context.Context) error   { return nil }
func (m *mockChainContainer) Stop(context.Context) error    { return nil }
func (m *mockChainContainer) Pause(context.Context) error   { return nil }
func (m *mockChainContainer) Resume(context.Context) error  { return nil }
func (m *mockChainContainer) ID() eth.ChainID               { return m.id }
func (m *mockChainContainer) BlockAtTimestamp(context.Context, uint64, eth.BlockLabel) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}
func (m *mockChainContainer) SyncStatus(context.Context) (*eth.SyncStatus, error) {
	return m.syncStatus, m.syncErr
}
func (m *mockChainContainer) VerifiedAt(context.Context, uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockChainContainer) L1ForL2(context.Context, eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockChainContainer) OptimisticAt(context.Context, uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}
func (m *mockChainContainer) OutputRootAtL2BlockNumber(context.Context, uint64) (eth.Bytes32, error) {
	return eth.Bytes32{}, nil
}
func (m *mockChainContainer) OptimisticOutputAtTimestamp(context.Context, uint64) (*eth.OutputResponse, error) {
	return nil, nil
}
func (m *mockChainContainer) RewindEngine(context.Context, uint64) error { return nil }
func (m *mockChainContainer) RegisterVerifier(activity.VerificationActivity) {}
func (m *mockChainContainer) FetchReceipts(context.Context, eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}
func (m *mockChainContainer) BlockTime() uint64                                       { return 2 }
func (m *mockChainContainer) InvalidateBlock(context.Context, uint64, common.Hash) (bool, error) {
	return false, nil
}
func (m *mockChainContainer) IsDenied(uint64, common.Hash) (bool, error) { return false, nil }
func (m *mockChainContainer) SetResetCallback(cc.ResetCallback)          {}

// mockVerifier implements activity.VerificationActivity for testing.
type mockVerifier struct {
	name      string
	currentL1 eth.BlockID
	lastTS    uint64
	inited    bool
}

var _ activity.VerificationActivity = (*mockVerifier)(nil)

func (m *mockVerifier) Name() string                             { return m.name }
func (m *mockVerifier) CurrentL1() eth.BlockID                   { return m.currentL1 }
func (m *mockVerifier) VerifiedAtTimestamp(uint64) (bool, error) { return true, nil }
func (m *mockVerifier) LatestVerifiedTimestamp() (uint64, bool)  { return m.lastTS, m.inited }
func (m *mockVerifier) Reset(eth.ChainID, uint64)               {}

func TestSyncStatus_HappyPath(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)
	status := &eth.SyncStatus{
		CurrentL1: eth.L1BlockRef{Number: 100},
		SafeL2:    eth.L2BlockRef{Number: 50},
	}

	chains := map[eth.ChainID]cc.ChainContainer{
		chainID: &mockChainContainer{id: chainID, syncStatus: status},
	}
	verifiers := []activity.VerificationActivity{
		&mockVerifier{name: "interop", currentL1: eth.BlockID{Number: 99}, lastTS: 1000, inited: true},
	}

	ss := New(chains, verifiers)
	resp, err := ss.syncStatus(context.Background())
	require.NoError(t, err)

	require.Len(t, resp.Chains, 1)
	require.Equal(t, status, resp.Chains[chainID])

	require.Len(t, resp.Verifiers, 1)
	require.Equal(t, "interop", resp.Verifiers[0].Name)
	require.Equal(t, uint64(99), resp.Verifiers[0].CurrentL1.Number)
	require.Equal(t, uint64(1000), resp.Verifiers[0].LatestVerifiedTimestamp)
	require.True(t, resp.Verifiers[0].Initialized)
}

func TestSyncStatus_MultipleChains(t *testing.T) {
	chain1 := eth.ChainIDFromUInt64(10)
	chain2 := eth.ChainIDFromUInt64(8453)
	status1 := &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 100}}
	status2 := &eth.SyncStatus{CurrentL1: eth.L1BlockRef{Number: 200}}

	chains := map[eth.ChainID]cc.ChainContainer{
		chain1: &mockChainContainer{id: chain1, syncStatus: status1},
		chain2: &mockChainContainer{id: chain2, syncStatus: status2},
	}

	ss := New(chains, nil)
	resp, err := ss.syncStatus(context.Background())
	require.NoError(t, err)

	require.Len(t, resp.Chains, 2)
	require.Equal(t, status1, resp.Chains[chain1])
	require.Equal(t, status2, resp.Chains[chain2])
	require.Empty(t, resp.Verifiers)
}

func TestSyncStatus_MultipleVerifiers(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainID: &mockChainContainer{id: chainID, syncStatus: &eth.SyncStatus{}},
	}
	verifiers := []activity.VerificationActivity{
		&mockVerifier{name: "interop", currentL1: eth.BlockID{Number: 50}, lastTS: 500, inited: true},
		&mockVerifier{name: "fault-proof", currentL1: eth.BlockID{Number: 60}, lastTS: 600, inited: true},
	}

	ss := New(chains, verifiers)
	resp, err := ss.syncStatus(context.Background())
	require.NoError(t, err)

	require.Len(t, resp.Verifiers, 2)
	require.Equal(t, "interop", resp.Verifiers[0].Name)
	require.Equal(t, uint64(500), resp.Verifiers[0].LatestVerifiedTimestamp)
	require.Equal(t, "fault-proof", resp.Verifiers[1].Name)
	require.Equal(t, uint64(600), resp.Verifiers[1].LatestVerifiedTimestamp)
}

func TestSyncStatus_ChainSyncStatusError(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainID: &mockChainContainer{id: chainID, syncErr: fmt.Errorf("node not ready")},
	}

	ss := New(chains, nil)
	_, err := ss.syncStatus(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "node not ready")
}

func TestSyncStatus_EmptyState(t *testing.T) {
	chains := map[eth.ChainID]cc.ChainContainer{}

	ss := New(chains, nil)
	resp, err := ss.syncStatus(context.Background())
	require.NoError(t, err)

	require.Empty(t, resp.Chains)
	require.Empty(t, resp.Verifiers)
}

func TestSyncStatus_UninitializedVerifier(t *testing.T) {
	chainID := eth.ChainIDFromUInt64(420)
	chains := map[eth.ChainID]cc.ChainContainer{
		chainID: &mockChainContainer{id: chainID, syncStatus: &eth.SyncStatus{}},
	}
	verifiers := []activity.VerificationActivity{
		&mockVerifier{name: "interop", inited: false, lastTS: 0},
	}

	ss := New(chains, verifiers)
	resp, err := ss.syncStatus(context.Background())
	require.NoError(t, err)

	require.Len(t, resp.Verifiers, 1)
	require.False(t, resp.Verifiers[0].Initialized)
	require.Equal(t, uint64(0), resp.Verifiers[0].LatestVerifiedTimestamp)
}
