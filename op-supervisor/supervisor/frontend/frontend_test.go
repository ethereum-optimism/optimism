package frontend

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-supervisor/supervisor/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type mockSupervisor struct {
	returnErr error
}

func (m *mockSupervisor) CheckAccessList(ctx context.Context, inboxEntries []common.Hash, minSafety types.SafetyLevel, executingDescriptor types.ExecutingDescriptor) error {
	return m.returnErr
}

// implement other methods as stubs (not used in this test)
func (m *mockSupervisor) CrossDerivedToSource(context.Context, eth.ChainID, eth.BlockID) (eth.BlockRef, error) {
	return eth.BlockRef{}, nil
}
func (m *mockSupervisor) LocalUnsafe(context.Context, eth.ChainID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockSupervisor) LocalSafe(context.Context, eth.ChainID) (types.DerivedIDPair, error) {
	return types.DerivedIDPair{}, nil
}
func (m *mockSupervisor) CrossSafe(context.Context, eth.ChainID) (types.DerivedIDPair, error) {
	return types.DerivedIDPair{}, nil
}
func (m *mockSupervisor) Finalized(context.Context, eth.ChainID) (eth.BlockID, error) {
	return eth.BlockID{}, nil
}
func (m *mockSupervisor) FinalizedL1(context.Context) (eth.BlockRef, error) {
	return eth.BlockRef{}, nil
}
func (m *mockSupervisor) SuperRootAtTimestamp(context.Context, eth.Uint64Quantity) (eth.SuperRootResponse, error) {
	return eth.SuperRootResponse{}, nil
}
func (m *mockSupervisor) AllSafeDerivedAt(context.Context, eth.BlockID) (map[eth.ChainID]eth.BlockID, error) {
	return nil, nil
}
func (m *mockSupervisor) SyncStatus(context.Context) (eth.SupervisorSyncStatus, error) {
	return eth.SupervisorSyncStatus{}, nil
}

func TestCheckAccessList_ConflictMapping(t *testing.T) {
	const ConflictingDataErrorCode eth.ErrorCode = -320600
	qf := &QueryFrontend{Supervisor: &mockSupervisor{returnErr: types.ErrConflict}}

	err := qf.CheckAccessList(context.Background(), nil, types.LocalUnsafe, types.ExecutingDescriptor{})
	require.Error(t, err)
	var inputErr eth.InputError
	require.True(t, errors.As(err, &inputErr), "error should be of type eth.InputError")
	require.Equal(t, ConflictingDataErrorCode, inputErr.Code, "error code should match -320600 for conflicting_data")
}

func TestCheckAccessList_OtherErrorPassthrough(t *testing.T) {
	customErr := errors.New("some other error")
	qf := &QueryFrontend{Supervisor: &mockSupervisor{returnErr: customErr}}

	err := qf.CheckAccessList(context.Background(), nil, types.LocalUnsafe, types.ExecutingDescriptor{})
	require.ErrorIs(t, err, customErr)
}

func TestCheckAccessList_NoError(t *testing.T) {
	qf := &QueryFrontend{Supervisor: &mockSupervisor{returnErr: nil}}
	err := qf.CheckAccessList(context.Background(), nil, types.LocalUnsafe, types.ExecutingDescriptor{})
	require.NoError(t, err)
}
