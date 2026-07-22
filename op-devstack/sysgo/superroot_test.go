package sysgo

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/eth"
)

type transientSuperRootService struct {
	calls                atomic.Int32
	firstRequestCanceled chan error
}

func (s *transientSuperRootService) AtTimestamp(ctx context.Context, _ hexutil.Uint64) (eth.SuperRootAtTimestampResponse, error) {
	switch s.calls.Add(1) {
	case 1:
		<-ctx.Done()
		s.firstRequestCanceled <- ctx.Err()
		return eth.SuperRootAtTimestampResponse{}, ctx.Err()
	case 2:
		return eth.SuperRootAtTimestampResponse{
			Data: &eth.SuperRootResponseData{SuperRoot: eth.Bytes32{1}},
		}, nil
	default:
		<-ctx.Done()
		return eth.SuperRootAtTimestampResponse{}, ctx.Err()
	}
}

func TestGetSuperRootReusesSuccessfulPollResponse(t *testing.T) {
	service := &transientSuperRootService{
		firstRequestCanceled: make(chan error, 1),
	}
	rpcServer := rpc.NewServer()
	require.NoError(t, rpcServer.RegisterName("superroot", service))
	server := httptest.NewServer(rpcServer)
	t.Cleanup(server.Close)

	root := getSuperRoot(devtest.SerialT(t), server.URL, 1)
	require.ErrorIs(t, <-service.firstRequestCanceled, context.Canceled)
	require.Equal(t, eth.Bytes32{1}, root)
	require.Equal(t, int32(2), service.calls.Load())
}
