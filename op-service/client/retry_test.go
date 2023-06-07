package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum-optimism/optimism/op-service/client"

	opclient "github.com/ethereum-optimism/optimism/op-node/client"
)

type MockRPC struct {
	mock.Mock
}

func (m *MockRPC) Close() {
	m.Called()
}

func (m *MockRPC) CallContext(ctx context.Context, result any, method string, args ...any) error {
	out := m.Mock.MethodCalled("CallContext", ctx, result, method, args)
	return *out[0].(*error)
}

func (m *MockRPC) BatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	out := m.Mock.MethodCalled("BatchCallContext", ctx, b)
	return *out[0].(*error)
}

func (m *MockRPC) EthSubscribe(ctx context.Context, channel any, args ...any) (ethereum.Subscription, error) {
	out := m.Mock.MethodCalled("EthSubscribe", ctx, channel, args)
	return *out[0].(*ethereum.Subscription), *out[1].(*error)
}

func (m *MockRPC) ExpectCallContext(err error, result any, method string, arg string) {
	m.On("CallContext", mock.Anything, result, method, []interface{}{arg}).Return(&err)
}

func (m *MockRPC) ExpectBatchCallContext(err error, b []rpc.BatchElem) {
	m.On("BatchCallContext", mock.Anything, b).Return(&err)
}

func (m *MockRPC) ExpectEthSubscribe(sub ethereum.Subscription, err error, channel any, args ...any) {
	m.On("EthSubscribe", mock.Anything, channel, args).Return(&sub, &err)
}

var _ opclient.RPC = (*MockRPC)(nil)

func TestClient_BackoffClient_Strategy(t *testing.T) {
	mockRpc := &MockRPC{}
	backoffClient := client.NewRetryingClient(mockRpc, 0)
	require.Equal(t, backoffClient.BackoffStrategy(), client.ExponentialBackoff)

	fixedStrategy := &backoff.FixedStrategy{}
	backoffClient = client.NewRetryingClient(mockRpc, 0, fixedStrategy)
	require.Equal(t, backoffClient.BackoffStrategy(), fixedStrategy)
}

func TestClient_BackoffClient_Close(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.On("Close").Return()
	backoffClient := client.NewRetryingClient(mockRpc, 0)
	backoffClient.Close()
	require.True(t, mockRpc.AssertCalled(t, "Close"))
}

func TestClient_BackoffClient_CallContext(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectCallContext(nil, nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(mockRpc, 1)
	err := backoffClient.CallContext(context.Background(), nil, "foo", "bar")
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "CallContext", mock.Anything, nil, "foo", []interface{}{"bar"}))
}

func TestClient_BackoffClient_CallContext_WithRetries(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectCallContext(errors.New("foo"), nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(mockRpc, 2)
	err := backoffClient.CallContext(context.Background(), nil, "foo", "bar")
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "CallContext", 2))
}

func TestClient_BackoffClient_BatchCallContext(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectBatchCallContext(nil, []rpc.BatchElem(nil))
	backoffClient := client.NewRetryingClient(mockRpc, 1)
	err := backoffClient.BatchCallContext(context.Background(), nil)
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "BatchCallContext", mock.Anything, []rpc.BatchElem(nil)))
}

func TestClient_BackoffClient_BatchCallContext_WithRetries(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectBatchCallContext(errors.New("foo"), []rpc.BatchElem(nil))
	backoffClient := client.NewRetryingClient(mockRpc, 2)
	err := backoffClient.BatchCallContext(context.Background(), nil)
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "BatchCallContext", 2))
}

func TestClient_BackoffClient_EthSubscribe(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectEthSubscribe(ethereum.Subscription(nil), nil, nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(mockRpc, 1)
	_, err := backoffClient.EthSubscribe(context.Background(), nil, "foo", "bar")
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "EthSubscribe", mock.Anything, nil, []interface{}{"foo", "bar"}))
}

func TestClient_BackoffClient_EthSubscribe_WithRetries(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectEthSubscribe(ethereum.Subscription(nil), errors.New("foo"), nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(mockRpc, 2)
	_, err := backoffClient.EthSubscribe(context.Background(), nil, "foo", "bar")
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "EthSubscribe", 2))
}
