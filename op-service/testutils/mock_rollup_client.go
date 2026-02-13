package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockRollupClient struct {
	mock.Mock
}

func (m *MockRollupClient) OutputAtBlock(ctx context.Context, blockNum uint64) (*eth.OutputResponse, error) {
	out := m.Mock.Called(blockNum)
	return out.Get(0).(*eth.OutputResponse), out.Error(1)
}

func (m *MockRollupClient) ExpectOutputAtBlock(blockNum uint64, response *eth.OutputResponse, err error) *mock.Call {
	return m.Mock.On("OutputAtBlock", blockNum).Once().Return(response, err)
}

func (m *MockRollupClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	out := m.Mock.Called()
	return out.Get(0).(*eth.SyncStatus), out.Error(1)
}

func (m *MockRollupClient) ExpectSyncStatus(status *eth.SyncStatus, err error) {
	m.Mock.On("SyncStatus").Once().Return(status, err)
}

func (m *MockRollupClient) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	out := m.Mock.Called()
	return out.Get(0).(*rollup.Config), out.Error(1)
}

func (m *MockRollupClient) ExpectRollupConfig(config *rollup.Config, err error) {
	m.Mock.On("RollupConfig").Once().Return(config, err)
}

func (m *MockRollupClient) StartSequencer(ctx context.Context, unsafeHead common.Hash) error {
	out := m.Mock.Called(unsafeHead)
	return out.Error(0)
}

func (m *MockRollupClient) ExpectStartSequencer(unsafeHead common.Hash, err error) {
	m.Mock.On("StartSequencer", unsafeHead).Once().Return(err)
}

func (m *MockRollupClient) SequencerActive(ctx context.Context) (bool, error) {
	out := m.Mock.Called()
	return out.Bool(0), out.Error(1)
}

func (m *MockRollupClient) ExpectSequencerActive(active bool, err error) {
	m.Mock.On("SequencerActive").Once().Return(active, err)
}

func (m *MockRollupClient) ExpectClose() {
	m.Mock.On("Close").Once()
}

func (m *MockRollupClient) MaybeClose() {
	m.Mock.On("Close").Maybe()
}

func (m *MockRollupClient) Close() {
	m.Mock.Called()
}
