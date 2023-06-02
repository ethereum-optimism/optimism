package client_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/ethereum-optimism/optimism/op-service/backoff"
	"github.com/ethereum-optimism/optimism/op-service/client"
	"github.com/ethereum-optimism/optimism/op-service/client/mocks"
)

func TestClient_BackoffClient_Strategy(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	backoffClient := client.NewBackoffClient(mockRpc, 0)
	require.Equal(t, backoffClient.BackoffStrategy(), client.ExponentialBackoff)

	fixedStrategy := &backoff.FixedStrategy{}
	backoffClient = client.NewBackoffClient(mockRpc, 0, fixedStrategy)
	require.Equal(t, backoffClient.BackoffStrategy(), fixedStrategy)
}

func TestClient_BackoffClient_Close(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	mockRpc.On("Close").Return()
	backoffClient := client.NewBackoffClient(mockRpc, 0)
	backoffClient.Close()
	require.True(t, mockRpc.AssertCalled(t, "Close"))
}

func TestClient_BackoffClient_CallContext(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	mockRpc.On("CallContext", mock.Anything, nil, "foo", "bar").Return(nil)
	backoffClient := client.NewBackoffClient(mockRpc, 1)
	err := backoffClient.CallContext(context.Background(), nil, "foo", "bar")
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "CallContext", mock.Anything, nil, "foo", "bar"))
}

func TestClient_BackoffClient_CallContext_WithRetries(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	mockRpc.On("CallContext", mock.Anything, nil, "foo", "bar").Return(errors.New("foo"))
	backoffClient := client.NewBackoffClient(mockRpc, 2)
	err := backoffClient.CallContext(context.Background(), nil, "foo", "bar")
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "CallContext", 2))
}

func TestClient_BackoffClient_BatchCallContext(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	mockRpc.On("BatchCallContext", mock.Anything, []rpc.BatchElem(nil)).Return(nil)
	backoffClient := client.NewBackoffClient(mockRpc, 1)
	err := backoffClient.BatchCallContext(context.Background(), nil)
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "BatchCallContext", mock.Anything, []rpc.BatchElem(nil)))
}

func TestClient_BackoffClient_BatchCallContext_WithRetries(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	mockRpc.On("BatchCallContext", mock.Anything, []rpc.BatchElem(nil)).Return(errors.New("foo"))
	backoffClient := client.NewBackoffClient(mockRpc, 2)
	err := backoffClient.BatchCallContext(context.Background(), nil)
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "BatchCallContext", 2))
}

func TestClient_BackoffClient_EthSubscribe(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	mockRpc.On("EthSubscribe", mock.Anything, nil, "foo", "bar").Return(nil, nil)
	backoffClient := client.NewBackoffClient(mockRpc, 1)
	_, err := backoffClient.EthSubscribe(context.Background(), nil, "foo", "bar")
	require.NoError(t, err)
	require.True(t, mockRpc.AssertCalled(t, "EthSubscribe", mock.Anything, nil, "foo", "bar"))
}

func TestClient_BackoffClient_EthSubscribe_WithRetries(t *testing.T) {
	mockRpc := mocks.NewInnerRPC(t)
	mockRpc.On("EthSubscribe", mock.Anything, nil, "foo", "bar").Return(nil, errors.New("foo"))
	backoffClient := client.NewBackoffClient(mockRpc, 2)
	_, err := backoffClient.EthSubscribe(context.Background(), nil, "foo", "bar")
	require.Error(t, err)
	require.True(t, mockRpc.AssertNumberOfCalls(t, "EthSubscribe", 2))
}
