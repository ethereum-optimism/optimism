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

func (m *MockRollupClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	out := m.Mock.MethodCalled("SyncStatus")
	return out[0].(*eth.SyncStatus), *out[1].(*error)
}

func (m *MockRollupClient) ExpectSyncStatus(status *eth.SyncStatus, err error) {
	m.Mock.On("SyncStatus").Once().Return(status, &err)
}

func (m *MockRollupClient) RollupConfig(ctx context.Context) (*rollup.Config, error) {
	out := m.Mock.MethodCalled("RollupConfig")
	return out[0].(*rollup.Config), *out[1].(*error)
}

func (m *MockRollupClient) ExpectRollupConfig(config *rollup.Config, err error) {
	m.Mock.On("RollupConfig").Once().Return(config, &err)
}

func (m *MockRollupClient) StartSequencer(ctx context.Context, unsafeHead common.Hash) error {
	out := m.Mock.MethodCalled("StartSequencer", unsafeHead)
	return *out[0].(*error)
}

func (m *MockRollupClient) ExpectStartSequencer(unsafeHead common.Hash, err error) {
	m.Mock.On("StartSequencer", unsafeHead).Once().Return(&err)
}

func (m *MockRollupClient) SequencerActive(ctx context.Context) (bool, error) {
	out := m.Mock.MethodCalled("SequencerActive")
	return out[0].(bool), *out[1].(*error)
}

func (m *MockRollupClient) ExpectSequencerActive(active bool, err error) {
	m.Mock.On("SequencerActive").Once().Return(active, &err)
}

func (m *MockRollupClient) ExpectClose() {
	m.Mock.On("Close").Once()
}

func (m *MockRollupClient) Close() {
	m.Mock.MethodCalled("Close")
}
