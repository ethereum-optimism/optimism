package testutils

import (
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/mock"
)

type MockDebugClient struct {
	mock.Mock
}

func (m *MockDebugClient) ExpectNodeByHash(hash common.Hash, res []byte, err error) {
	m.Mock.On("NodeByHash", hash).Once().Return(res, &err)
}

func (m *MockDebugClient) NodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	out := m.Mock.MethodCalled("NodeByHash", hash)
	return out[0].([]byte), *out[1].(*error)
}

func (m *MockDebugClient) ExpectCodeByHash(hash common.Hash, res []byte, err error) {
	m.Mock.On("CodeByHash", hash).Once().Return(res, &err)
}

func (m *MockDebugClient) CodeByHash(ctx context.Context, hash common.Hash) ([]byte, error) {
	out := m.Mock.MethodCalled("CodeByHash", hash)
	return out[0].([]byte), *out[1].(*error)
}
