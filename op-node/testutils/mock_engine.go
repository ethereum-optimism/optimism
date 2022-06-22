package testutils

import (
	"context"
	"math/big"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockEngine struct {
	mock.Mock
}

func (m *MockEngine) L2BlockRefHead(ctx context.Context) (eth.L2BlockRef, error) {
	out := m.Mock.MethodCalled("L2BlockRefHead")
	return out[0].(eth.L2BlockRef), out[1].(error)
}

func (m *MockEngine) L2BlockRefByHash(ctx context.Context, l2Hash common.Hash) (eth.L2BlockRef, error) {
	out := m.Mock.MethodCalled("L2BlockRefByHash", l2Hash)
	return out[0].(eth.L2BlockRef), out[1].(error)
}

func (m *MockEngine) GetPayload(ctx context.Context, payloadId eth.PayloadID) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("GetPayload", payloadId)
	return out[0].(*eth.ExecutionPayload), out[1].(error)
}

func (m *MockEngine) ForkchoiceUpdate(ctx context.Context, state *eth.ForkchoiceState, attr *eth.PayloadAttributes) (*eth.ForkchoiceUpdatedResult, error) {
	out := m.Mock.MethodCalled("ForkchoiceUpdate", state, attr)
	return out[0].(*eth.ForkchoiceUpdatedResult), out[1].(error)
}

func (m *MockEngine) NewPayload(ctx context.Context, payload *eth.ExecutionPayload) (*eth.PayloadStatusV1, error) {
	out := m.Mock.MethodCalled("NewPayload", payload)
	return out[0].(*eth.PayloadStatusV1), out[1].(error)
}

func (m *MockEngine) PayloadByHash(ctx context.Context, hash common.Hash) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("PayloadByHash", hash)
	return out[0].(*eth.ExecutionPayload), out[1].(error)
}

func (m *MockEngine) PayloadByNumber(ctx context.Context, b *big.Int) (*eth.ExecutionPayload, error) {
	out := m.Mock.MethodCalled("PayloadByNumber", b)
	return out[0].(*eth.ExecutionPayload), out[1].(error)
}

func (m *MockEngine) UnsafeBlockIDs(ctx context.Context, safeHead eth.BlockID, max uint64) ([]eth.BlockID, error) {
	out := m.Mock.MethodCalled("UnsafeBlockIDs", safeHead, max)
	return out[0].([]eth.BlockID), out[1].(error)
}
