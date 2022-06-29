package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockEngine struct {
	mock.Mock
}

func (m *MockEngine) L2BlockRefHead(ctx context.Context) (eth.L2BlockRef, error) {
	out := m.Mock.MethodCalled("L2BlockRefHead")
	return out[0].(eth.L2BlockRef), *out[1].(*error)
}

func (m *MockEngine) ExpectL2BlockRefHead(ref eth.L1BlockRef, err error) {
	m.Mock.On("L2BlockRefHead").Once().Return(ref, &err)
}

func (m *MockEngine) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	out := m.Mock.MethodCalled("L2BlockRefByHash", l2Hash)
	return out[0].(eth.L2BlockRef), *out[1].(*error)
}

func (m *MockEngine) ExpectL2BlockRefByHash(l2Hash common.Hash, ref eth.L1BlockRef, err error) {
	m.Mock.On("L2BlockRefByHash", l2Hash).Once().Return(ref, &err)
}

func (m *MockEngine) GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("GetPayload", payloadId)
	return out[0].(*eth.ExecutionPayload), *out[1].(*error)
}

func (m *MockEngine) ExpectGetPayload(payloadId eth.PayloadID, payload *eth.ExecutionPayload, err error) {
	m.Mock.On("GetPayload", payloadId).Once().Return(payload, &err)
}

func (m *MockEngine) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	out := m.Mock.MethodCalled("ForkchoiceUpdate", state, attr)
	return out[0].(*eth.ForkchoiceUpdatedResult), *out[1].(*error)
}

func (m *MockEngine) ExpectForkchoiceUpdate(state *eth.ForkchoiceState, attr *eth.PayloadAttributes, result *eth.ForkchoiceUpdatedResult, err error) {
	m.Mock.On("ForkchoiceUpdate", state, attr).Once().Return(result, &err)
}

func (m *MockEngine) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	out := m.Mock.MethodCalled("NewPayload", payload)
	return out[0].(*eth.PayloadStatusV1), *out[1].(*error)
}

func (m *MockEngine) ExpectNewPayload(payload *eth.ExecutionPayload, result *eth.PayloadStatusV1, err error) {
	m.Mock.On("NewPayload", payload).Once().Return(result, &err)
}

func (m *MockEngine) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("PayloadByHash", hash)
	return out[0].(*eth.ExecutionPayload), *out[1].(*error)
}

func (m *MockEngine) ExpectPayloadByHash(hash common.Hash, payload *eth.ExecutionPayload, err error) {
	m.Mock.On("PayloadByHash", hash).Once().Return(payload, &err)
}

func (m *MockEngine) PayloadByNumber(ctx context.Context, n uint64) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("PayloadByNumber", n)
	return out[0].(*eth.ExecutionPayload), *out[1].(*error)
}

func (m *MockEngine) ExpectPayloadByNumber(hash common.Hash, payload *eth.ExecutionPayload, err error) {
	m.Mock.On("PayloadByNumber", hash).Once().Return(payload, &err)
}
