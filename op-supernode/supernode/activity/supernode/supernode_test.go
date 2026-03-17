package supernode

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supernode/supernode/activity"
	cc "github.com/ethereum-optimism/optimism/op-supernode/supernode/chain_container"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethlog "github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockCC struct {
	status      *eth.SyncStatus
	verifierL1s []eth.BlockID

	verifiedErr   error
	outputErr     error
	syncStatusErr error
}

func (m *mockCC) Start(ctx context.Context) error  { return nil }
func (m *mockCC) Stop(ctx context.Context) error   { return nil }
func (m *mockCC) Pause(ctx context.Context) error  { return nil }
func (m *mockCC) Resume(ctx context.Context) error { return nil }

func (m *mockCC) RegisterVerifier(v activity.VerificationActivity) {}
func (m *mockCC) VerifierCurrentL1s() []eth.BlockID {
	return m.verifierL1s
}

func (m *mockCC) LocalSafeBlockAtTimestamp(ctx context.Context, ts uint64) (eth.L2BlockRef, error) {
	return eth.L2BlockRef{}, nil
}

func (m *mockCC) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	if m.syncStatusErr != nil {
		return nil, m.syncStatusErr
	}
	if m.status == nil {
		return &eth.SyncStatus{}, nil
	}
	return m.status, nil
}

func (m *mockCC) SafeHeadAtL1(ctx context.Context, l1BlockNum uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}

func (m *mockCC) L1AtSafeHead(ctx context.Context, l2 eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}

func (m *mockCC) VerifiedAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	if m.verifiedErr != nil {
		return eth.BlockID{}, eth.BlockID{}, m.verifiedErr
	}
	return eth.BlockID{}, eth.BlockID{}, nil
}

func (m *mockCC) OptimisticAt(ctx context.Context, ts uint64) (eth.BlockID, eth.BlockID, error) {
	return eth.BlockID{}, eth.BlockID{}, nil
}

func (m *mockCC) OutputRootAtL2BlockNumber(ctx context.Context, l2BlockNum uint64) (eth.Bytes32, error) {
	if m.outputErr != nil {
		return eth.Bytes32{}, m.outputErr
	}
	return eth.Bytes32{}, nil
}

func (m *mockCC) OptimisticOutputAtTimestamp(ctx context.Context, ts uint64) (*eth.OutputResponse, error) {
	return &eth.OutputResponse{}, nil
}

func (m *mockCC) RewindEngine(ctx context.Context, timestamp uint64, invalidatedBlock eth.BlockRef) error {
	return nil
}

func (m *mockCC) L1ForL2(ctx context.Context, l2Block eth.BlockID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}

func (m *mockCC) FetchReceipts(ctx context.Context, blockID eth.BlockID) (eth.BlockInfo, types.Receipts, error) {
	return nil, nil, nil
}

func (m *mockCC) ID() eth.ChainID {
	return eth.ChainIDFromUInt64(10)
}

func (m *mockCC) BlockTime() uint64 { return 1 }

func (m *mockCC) InvalidateBlock(ctx context.Context, height uint64, payloadHash common.Hash, decisionTimestamp uint64) (bool, error) {
	return false, nil
}

func (m *mockCC) IsDenied(height uint64, payloadHash common.Hash) (bool, error) {
	return false, nil
}

func (m *mockCC) PruneDeniedAtOrAfterTimestamp(timestamp uint64) (map[uint64][]common.Hash, error) {
	return nil, nil
}

func (m *mockCC) SetResetCallback(cb cc.ResetCallback) {}

var _ cc.ChainContainer = (*mockCC)(nil)

func TestSupernode_SyncStatus_Succeeds(t *testing.T) {
	t.Parallel()
	chainA := eth.ChainIDFromUInt64(10)
	chainB := eth.ChainIDFromUInt64(420)

	chains := map[eth.ChainID]cc.ChainContainer{
		chainA: &mockCC{
			status: &eth.SyncStatus{
				CurrentL1:     eth.L1BlockRef{Number: 2000},
				UnsafeL2:      eth.L2BlockRef{Number: 120, Time: 220},
				CrossUnsafeL2: eth.L2BlockRef{Number: 118, Time: 205},
				SafeL2:        eth.L2BlockRef{Number: 110, Time: 170},
				LocalSafeL2:   eth.L2BlockRef{Number: 111, Time: 180},
				FinalizedL2:   eth.L2BlockRef{Number: 100, Time: 140},
			},
		},
		chainB: &mockCC{
			status: &eth.SyncStatus{
				CurrentL1:     eth.L1BlockRef{Number: 2000},
				UnsafeL2:      eth.L2BlockRef{Number: 130, Time: 230},
				CrossUnsafeL2: eth.L2BlockRef{Number: 128, Time: 215},
				SafeL2:        eth.L2BlockRef{Number: 112, Time: 175},
				LocalSafeL2:   eth.L2BlockRef{Number: 113, Time: 190},
				FinalizedL2:   eth.L2BlockRef{Number: 101, Time: 150},
			},
		},
	}

	s := New(gethlog.New(), chains)
	api := &api{a: s}
	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)

	require.Len(t, out.Chains, 2)
	require.Contains(t, out.Chains, chainA)
	require.Contains(t, out.Chains, chainB)
	require.Equal(t, []eth.ChainID{chainA, chainB}, out.ChainIDs)
	require.Equal(t, uint64(2000), out.CurrentL1.Number)
	require.Equal(t, out.Chains[chainA].CurrentL1.ID(), out.CurrentL1)
	require.Equal(t, uint64(170), out.SafeTimestamp)
	require.Equal(t, uint64(180), out.LocalSafeTimestamp)
	require.Equal(t, uint64(140), out.FinalizedTimestamp)

	statusA := out.Chains[chainA]
	require.Greater(t, statusA.UnsafeL2.Number, statusA.SafeL2.Number)
	require.Greater(t, statusA.SafeL2.Number, statusA.FinalizedL2.Number)
	require.Greater(t, statusA.LocalSafeL2.Number, statusA.SafeL2.Number)
}

func TestSupernode_SyncStatus_UsesMinimumCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 100, Hash: common.Hash{0x11}},
			},
		},
		eth.ChainIDFromUInt64(11): &mockCC{
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 101, Hash: common.Hash{0x22}},
			},
		},
	}
	s := New(gethlog.New(), chains)
	api := &api{a: s}
	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, eth.BlockID{Number: 100, Hash: common.Hash{0x11}}, out.CurrentL1)
}

func TestSupernode_SyncStatus_UsesMinimumVerifierCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 200, Hash: common.Hash{0x11}},
			},
			verifierL1s: []eth.BlockID{
				{Number: 150, Hash: common.Hash{0x33}},
				{Number: 175, Hash: common.Hash{0x44}},
			},
		},
		eth.ChainIDFromUInt64(11): &mockCC{
			status: &eth.SyncStatus{
				CurrentL1: eth.L1BlockRef{Number: 180, Hash: common.Hash{0x22}},
			},
			verifierL1s: []eth.BlockID{
				{Number: 190, Hash: common.Hash{0x55}},
			},
		},
	}
	s := New(gethlog.New(), chains)
	api := &api{a: s}
	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, eth.BlockID{Number: 150, Hash: common.Hash{0x33}}, out.CurrentL1)
}

func TestSupernode_SyncStatus_ErrorOnCurrentL1(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			syncStatusErr: assertErr(),
		},
	}
	s := New(gethlog.New(), chains)
	api := &api{a: s}
	_, err := api.SyncStatus(context.Background())
	require.Error(t, err)
}

func TestSupernode_SyncStatus_IgnoresUnsafeOutputRootErrors(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{
		eth.ChainIDFromUInt64(10): &mockCC{
			verifiedErr: fmt.Errorf("not available: %w", ethereum.NotFound),
			outputErr:   assertErr(),
			status: &eth.SyncStatus{
				CurrentL1:   eth.L1BlockRef{Number: 100},
				UnsafeL2:    eth.L2BlockRef{Number: 10, Time: 20},
				LocalSafeL2: eth.L2BlockRef{Number: 9, Time: 18},
				FinalizedL2: eth.L2BlockRef{Number: 8, Time: 16},
			},
		},
	}
	s := New(gethlog.New(), chains)
	api := &api{a: s}
	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Equal(t, []eth.ChainID{eth.ChainIDFromUInt64(10)}, out.ChainIDs)
	require.Equal(t, eth.BlockID{Number: 100}, out.CurrentL1)
	require.Equal(t, out.Chains[eth.ChainIDFromUInt64(10)].CurrentL1.ID(), out.CurrentL1)
	require.Equal(t, uint64(0), out.SafeTimestamp)
	require.Equal(t, uint64(18), out.LocalSafeTimestamp)
}

func TestSupernode_SyncStatus_EmptyChains(t *testing.T) {
	t.Parallel()
	chains := map[eth.ChainID]cc.ChainContainer{}
	s := New(gethlog.New(), chains)
	api := &api{a: s}

	out, err := api.SyncStatus(context.Background())
	require.NoError(t, err)
	require.Len(t, out.Chains, 0)
	require.Len(t, out.ChainIDs, 0)
	require.Equal(t, eth.BlockID{}, out.CurrentL1)
	require.Equal(t, uint64(0), out.SafeTimestamp)
	require.Equal(t, uint64(0), out.LocalSafeTimestamp)
	require.Equal(t, uint64(0), out.FinalizedTimestamp)
}

func assertErr() error { return fmt.Errorf("mock error") }
