package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MockEngine struct {
	MockL2Client
}

func (m *MockEngine) GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	out := m.Mock.Called(payloadId)
	return out.Get(0).(*eth.ExecutionPayload), out.Error(1)
}

func (m *MockEngine) ExpectGetPayload(payloadId eth.PayloadID, payload *eth.ExecutionPayload, err error) {
	m.Mock.On("GetPayload", payloadId).Once().Return(payload, err)
}

func (m *MockEngine) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	out := m.Mock.Called(state, attr)
	return out.Get(0).(*eth.ForkchoiceUpdatedResult), out.Error(1)
}

func (m *MockEngine) ExpectForkchoiceUpdate(state *eth.ForkchoiceState, attr *eth.PayloadAttributes, result *eth.ForkchoiceUpdatedResult, err error) {
	m.Mock.On("ForkchoiceUpdate", state, attr).Once().Return(result, err)
}

func (m *MockEngine) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	out := m.Mock.Called(payload)
	return out.Get(0).(*eth.PayloadStatusV1), out.Error(1)
}

func (m *MockEngine) ExpectNewPayload(payload *eth.ExecutionPayload, result *eth.PayloadStatusV1, err error) {
	m.Mock.On("NewPayload", payload).Once().Return(result, err)
}
