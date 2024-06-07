package testutils

import (
	"context"
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type MockEngine struct {
	MockL2Client
}

func (m *MockEngine) GetPayload(ctx context.Context, payloadInfo eth.PayloadInfo) (*eth.ExecutionPayloadEnvelope, error) {
	out := m.Mock.Called(payloadInfo.ID)
	return out.Get(0).(*eth.ExecutionPayloadEnvelope), out.Error(1)
}

func (m *MockEngine) ExpectGetPayload(payloadId eth.PayloadID, payload *eth.ExecutionPayloadEnvelope, err error) {
	m.Mock.On("GetPayload", payloadId).Once().Return(payload, err)
}

func (m *MockEngine) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	out := m.Mock.Called(mustJson(state), mustJson(attr))
	return out.Get(0).(*eth.ForkchoiceUpdatedResult), out.Error(1)
}

func (m *MockEngine) ExpectForkchoiceUpdate(state *eth.ForkchoiceState, attr *eth.PayloadAttributes, result *eth.ForkchoiceUpdatedResult, err error) {
	m.Mock.On("ForkchoiceUpdate", mustJson(state), mustJson(attr)).Once().Return(result, err)
}

func (m *MockEngine) NewPayload(ctx context.Context, payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash) (*eth.PayloadStatusV1, error) {
	out := m.Mock.Called(mustJson(payload), mustJson(parentBeaconBlockRoot))
	return out.Get(0).(*eth.PayloadStatusV1), out.Error(1)
}

func (m *MockEngine) ExpectNewPayload(payload *eth.ExecutionPayload, parentBeaconBlockRoot *common.Hash, result *eth.PayloadStatusV1, err error) {
	m.Mock.On("NewPayload", mustJson(payload), mustJson(parentBeaconBlockRoot)).Once().Return(result, err)
}

func mustJson[E any](elem E) string {
	data, err := json.MarshalIndent(elem, "  ", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}
