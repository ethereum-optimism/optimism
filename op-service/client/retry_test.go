package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-node/testlog"
	"github.com/ethereum/go-ethereum/log"
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
	err, ok := out[0].(*error)
	if ok {
		return *err
	}
	return nil
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

func (m *MockRPC) OnBatchCallContext(err error, b []rpc.BatchElem, action func(callBatches []rpc.BatchElem)) {
	m.On("BatchCallContext", mock.Anything, b).Return(err).Run(func(args mock.Arguments) {
		action(args[1].([]rpc.BatchElem))
	})
}

func (m *MockRPC) ExpectEthSubscribe(sub ethereum.Subscription, err error, channel any, args ...any) {
	m.On("EthSubscribe", mock.Anything, channel, args).Return(&sub, &err)
}

var _ opclient.RPC = (*MockRPC)(nil)

func TestClient_BackoffClient_Strategy(t *testing.T) {
	mockRpc := &MockRPC{}
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 0)
	require.Equal(t, backoffClient.BackoffStrategy(), client.ExponentialBackoff)

	fixedStrategy := &backoff.FixedStrategy{}
	backoffClient = client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 0, fixedStrategy)
	require.Equal(t, backoffClient.BackoffStrategy(), fixedStrategy)
}

func TestClient_BackoffClient_Close(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.On("Close").Return()
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 0)
	backoffClient.Close()
	require.True(t, mockRpc.AssertCalled(t, "Close"))
}

func TestClient_BackoffClient_CallContext(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectCallContext(nil, nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 1)
	err := backoffClient.CallContext(context.Background(), nil, "foo", "bar")
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "CallContext", mock.Anything, nil, "foo", []interface{}{"bar"}))
}

func TestClient_BackoffClient_CallContext_WithRetries(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectCallContext(errors.New("foo"), nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 2, backoff.Fixed(0))
	err := backoffClient.CallContext(context.Background(), nil, "foo", "bar")
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "CallContext", 2))
}

func TestClient_BackoffClient_BatchCallContext(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectBatchCallContext(nil, []rpc.BatchElem{})
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 1)
	err := backoffClient.BatchCallContext(context.Background(), nil)
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "BatchCallContext", mock.Anything, []rpc.BatchElem{}))
}

func TestClient_BackoffClient_BatchCallContext_WithRetries(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectBatchCallContext(errors.New("foo"), []rpc.BatchElem{})
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 2, backoff.Fixed(0))
	err := backoffClient.BatchCallContext(context.Background(), nil)
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "BatchCallContext", 2))
}

func TestClient_BackoffClient_BatchCallContext_WithPartialRetries(t *testing.T) {
	batches := []rpc.BatchElem{
		{Method: "0"},
		{Method: "1"},
		{Method: "2"},
	}
	mockRpc := &MockRPC{}
	mockRpc.OnBatchCallContext(nil, batches, func(batch []rpc.BatchElem) {
		batch[0].Result = batch[0].Method
		batch[1].Error = errors.New("boom")
		batch[2].Error = errors.New("boom")
	})
	mockRpc.OnBatchCallContext(nil, []rpc.BatchElem{batches[1], batches[2]}, func(batch []rpc.BatchElem) {
		batch[0].Error = errors.New("boom again")
		batch[1].Result = batch[1].Method
	})
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 2, backoff.Fixed(0))
	err := backoffClient.BatchCallContext(context.Background(), batches)
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "BatchCallContext", 2))

	// Check our original batches got updated correctly
	require.Equal(t, rpc.BatchElem{Method: "0", Result: "0"}, batches[0])
	require.Equal(t, rpc.BatchElem{Method: "1", Result: nil, Error: errors.New("boom again")}, batches[1])
	require.Equal(t, rpc.BatchElem{Method: "2", Result: "2"}, batches[2])
}

func TestClient_BackoffClient_BatchCallContext_WithPartialRetriesUntilSuccess(t *testing.T) {
	batches := []rpc.BatchElem{
		{Method: "0"},
		{Method: "1"},
		{Method: "2"},
	}
	mockRpc := &MockRPC{}
	mockRpc.OnBatchCallContext(nil, batches, func(batch []rpc.BatchElem) {
		batch[0].Result = batch[0].Method
		batch[1].Error = errors.New("boom")
		batch[2].Error = errors.New("boom")
	})
	mockRpc.OnBatchCallContext(nil, []rpc.BatchElem{batches[1], batches[2]}, func(batch []rpc.BatchElem) {
		batch[0].Error = errors.New("boom again")
		batch[1].Result = batch[1].Method
	})
	mockRpc.OnBatchCallContext(nil, []rpc.BatchElem{batches[1]}, func(batch []rpc.BatchElem) {
		batch[0].Result = batch[0].Method
	})
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 4, backoff.Fixed(0))
	err := backoffClient.BatchCallContext(context.Background(), batches)
	require.NoError(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "BatchCallContext", 3))

	// Check our original batches got updated correctly
	require.Equal(t, rpc.BatchElem{Method: "0", Result: "0"}, batches[0])
	require.Equal(t, rpc.BatchElem{Method: "1", Result: "1"}, batches[1])
	require.Equal(t, rpc.BatchElem{Method: "2", Result: "2"}, batches[2])
}

func TestClient_BackoffClient_EthSubscribe(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectEthSubscribe(ethereum.Subscription(nil), nil, nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 1)
	_, err := backoffClient.EthSubscribe(context.Background(), nil, "foo", "bar")
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "EthSubscribe", mock.Anything, nil, []interface{}{"foo", "bar"}))
}

func TestClient_BackoffClient_EthSubscribe_WithRetries(t *testing.T) {
	mockRpc := &MockRPC{}
	mockRpc.ExpectEthSubscribe(ethereum.Subscription(nil), errors.New("foo"), nil, "foo", "bar")
	backoffClient := client.NewRetryingClient(testlog.Logger(t, log.LvlInfo), mockRpc, 2, backoff.Fixed(0))
	_, err := backoffClient.EthSubscribe(context.Background(), nil, "foo", "bar")
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "EthSubscribe", 2))
}
