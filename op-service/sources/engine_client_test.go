package sources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-node/rollup"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum-optimism/optimism/op-service/log"
	"github.com/ethereum-optimism/optimism/op-service/testlog"
)

func TestEngineAPIClient_GetPayloadWithoutExecutionPayload(t *testing.T) {
	ctx := context.Background()
	rpc := new(mockRPC)
	defer rpc.AssertExpectations(t)
	rpc.On("CallContext", ctx, mock.Anything, mock.Anything, mock.Anything).Return([]error{nil})

	client := NewEngineAPIClient(rpc, testlog.Logger(t, log.LevelInfo), &rollup.Config{})
	envelope, err := client.GetPayload(ctx, eth.PayloadInfo{ID: eth.PayloadID{1}})
	require.ErrorContains(t, err, "no execution payload")
	require.Nil(t, envelope)
}

func TestOPStackClient_SealBlock(t *testing.T) {
	ctx := context.Background()
	t.Run("NullResult", func(t *testing.T) {
		rpc := new(mockRPC)
		defer rpc.AssertExpectations(t)
		rpc.On("CallContext", ctx, mock.Anything, "opstack_sealBlockV1", mock.Anything).Return([]error{nil})

		envelope, err := NewOPStackClient(rpc).SealBlock(ctx, eth.PayloadInfo{})
		require.ErrorContains(t, err, "opstack_sealBlockV1 returned null")
		require.Nil(t, envelope)
	})
	t.Run("NoExecutionPayload", func(t *testing.T) {
		rpc := new(mockRPC)
		defer rpc.AssertExpectations(t)
		rpc.On("CallContext", ctx, mock.Anything, "opstack_sealBlockV1", mock.Anything).Run(func(args mock.Arguments) {
			*args[1].(**eth.ExecutionPayloadEnvelope) = &eth.ExecutionPayloadEnvelope{}
		}).Return([]error{nil})

		envelope, err := NewOPStackClient(rpc).SealBlock(ctx, eth.PayloadInfo{})
		require.ErrorContains(t, err, "no execution payload")
		require.Nil(t, envelope)
	})
}
