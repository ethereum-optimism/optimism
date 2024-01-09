package testutils

import (
	"context"

	"github.com/ethereum-optimism/optimism/op-service/client"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/stretchr/testify/mock"
)

var _ client.RPC = &MockRPCClient{}

type MockRPCClient struct {
	mock.Mock
}

func (m *MockRPCClient) Close() {
	m.Mock.Called()
}

func (m *MockRPCClient) CallContext(ctx context.Context, result any, method string, args ...any) error {
	out := m.Called(ctx, result, method, args)
	return out.Error(0)
}

func (m *MockRPCClient) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	out := m.Called(ctx, b)
	return out.Error(0)
}

func (m *MockRPCClient) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	out := m.Called(ctx, channel, args)
	return out.Get(0).(ethereum.Subscription), out.Error(1)
}
