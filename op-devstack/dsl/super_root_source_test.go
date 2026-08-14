package dsl

import (
	"context"
	"sync"
	"testing"

	"github.com/ethereum-optimism/optimism/op-devstack/devtest"
	"github.com/ethereum-optimism/optimism/op-service/apis"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
)

func TestLagControlledSuperRootSourceHoldsProgressAndForwardsData(t *testing.T) {
	dt := devtest.SerialT(t)
	upstream := newStubSuperRootSource(lagTestResponse(10, 100))
	lagged := NewLagControlledSuperRootSource(dt, upstream)

	upstream.setResponse(lagTestResponse(12, 200))
	resp, err := lagged.QueryAPI().SuperRootAtTimestamp(t.Context(), 200)
	require.NoError(t, err)
	require.Equal(t, uint64(10), resp.CurrentL1.Number)
	require.Equal(t, uint64(200), resp.CurrentSafeTimestamp)
	require.NotNil(t, resp.Data)
	require.Equal(t, uint64(200), resp.Data.Super.(*eth.SuperV1).Timestamp)

	status, err := lagged.QueryAPI().SyncStatus(t.Context())
	require.NoError(t, err)
	require.Equal(t, uint64(10), status.CurrentL1.Number)
	require.Equal(t, uint64(200), status.SafeTimestamp)

	lagged.Release()
	resp, err = lagged.QueryAPI().SuperRootAtTimestamp(t.Context(), 200)
	require.NoError(t, err)
	require.Equal(t, uint64(12), resp.CurrentL1.Number)
}

func TestLagControlledSuperRootSourceConcurrentRelease(t *testing.T) {
	dt := devtest.SerialT(t)
	upstream := newStubSuperRootSource(lagTestResponse(10, 100))
	lagged := NewLagControlledSuperRootSource(dt, upstream)

	const releasers = 8
	done := make(chan struct{}, releasers)
	for range releasers {
		go func() {
			lagged.Release()
			done <- struct{}{}
		}()
	}
	<-upstream.awaiting

	select {
	case <-done:
		t.Fatal("release completed before the upstream progressed past the held block")
	default:
	}

	upstream.setResponse(lagTestResponse(11, 101))
	for range releasers {
		<-done
	}
	resp, err := lagged.QueryAPI().SuperRootAtTimestamp(t.Context(), 101)
	require.NoError(t, err)
	require.Equal(t, uint64(11), resp.CurrentL1.Number)
}

func TestLagControlledSuperRootSourceCleanup(t *testing.T) {
	var api apis.SupernodeQueryAPI
	var endpoint string
	t.Run("source", func(t *testing.T) {
		dt := devtest.SerialT(t)
		lagged := NewLagControlledSuperRootSource(dt, newStubSuperRootSource(lagTestResponse(10, 100)))
		api = lagged.QueryAPI()
		endpoint = lagged.UserRPC()
		_, err := api.SuperRootAtTimestamp(t.Context(), 100)
		require.NoError(t, err)
	})

	_, err := api.SuperRootAtTimestamp(t.Context(), 100)
	require.Error(t, err)

	client, err := rpc.DialHTTP(endpoint)
	require.NoError(t, err)
	defer client.Close()
	var resp eth.SuperRootAtTimestampResponse
	err = client.CallContext(t.Context(), &resp, "superroot_atTimestamp", hexutil.Uint64(100))
	require.Error(t, err)
}

func lagTestResponse(currentL1 uint64, timestamp uint64) eth.SuperRootAtTimestampResponse {
	chainID := eth.ChainIDFromUInt64(901)
	super := eth.NewSuperV1(timestamp, eth.ChainIDAndOutput{
		ChainID: chainID,
		Output:  eth.Bytes32{byte(timestamp)},
	})
	return eth.SuperRootAtTimestampResponse{
		CurrentL1:            eth.BlockID{Hash: common.Hash{byte(currentL1)}, Number: currentL1},
		CurrentSafeTimestamp: timestamp,
		ChainIDs:             []eth.ChainID{chainID},
		Data: &eth.SuperRootResponseData{
			VerifiedRequiredL1: eth.BlockID{Number: currentL1},
			Super:              super,
			SuperRoot:          eth.SuperRoot(super),
		},
	}
}

type stubSuperRootAPI struct {
	mu       sync.Mutex
	response eth.SuperRootAtTimestampResponse
	changed  chan struct{}
}

func (s *stubSuperRootAPI) SuperRootAtTimestamp(context.Context, uint64) (eth.SuperRootAtTimestampResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.response, nil
}

func (s *stubSuperRootAPI) SyncStatus(context.Context) (eth.SuperNodeSyncStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return eth.SuperNodeSyncStatusResponse{
		CurrentL1:     s.response.CurrentL1,
		SafeTimestamp: s.response.CurrentSafeTimestamp,
		ChainIDs:      s.response.ChainIDs,
	}, nil
}

func (s *stubSuperRootAPI) setResponse(response eth.SuperRootAtTimestampResponse) {
	s.mu.Lock()
	s.response = response
	close(s.changed)
	s.changed = make(chan struct{})
	s.mu.Unlock()
}

type stubSuperRootSource struct {
	api       *stubSuperRootAPI
	awaitOnce sync.Once
	awaiting  chan struct{}
}

func newStubSuperRootSource(response eth.SuperRootAtTimestampResponse) *stubSuperRootSource {
	return &stubSuperRootSource{
		api:      &stubSuperRootAPI{response: response, changed: make(chan struct{})},
		awaiting: make(chan struct{}),
	}
}

func (s *stubSuperRootSource) QueryAPI() apis.SupernodeQueryAPI {
	return s.api
}

func (s *stubSuperRootSource) UserRPC() string {
	return ""
}

func (s *stubSuperRootSource) SuperRootAtTimestamp(timestamp uint64) eth.SuperRootAtTimestampResponse {
	resp, _ := s.api.SuperRootAtTimestamp(context.Background(), timestamp)
	return resp
}

func (s *stubSuperRootSource) AssertSuperRootAtTimestamp(uint64, common.Hash) {}

func (s *stubSuperRootSource) AwaitValidatedTimestamp(uint64) {}

func (s *stubSuperRootSource) AwaitFullyProcessedL1(targetL1 uint64) {
	s.awaitOnce.Do(func() { close(s.awaiting) })
	for {
		s.api.mu.Lock()
		if s.api.response.CurrentL1.Number > targetL1 {
			s.api.mu.Unlock()
			return
		}
		changed := s.api.changed
		s.api.mu.Unlock()
		<-changed
	}
}

func (s *stubSuperRootSource) setResponse(response eth.SuperRootAtTimestampResponse) {
	s.api.setResponse(response)
}
