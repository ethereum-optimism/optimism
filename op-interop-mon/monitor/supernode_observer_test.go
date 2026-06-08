package monitor

import (
	"context"
	"errors"
	"testing"

	messages "github.com/ethereum-optimism/optimism/op-core/interop/messages"
	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/stretchr/testify/require"
)

type mockSupernodeClient struct {
	status *eth.SuperNodeSyncStatusResponse
	hbErr  error
}

func (m *mockSupernodeClient) SyncStatus(ctx context.Context) (*eth.SuperNodeSyncStatusResponse, error) {
	return m.status, nil
}
func (m *mockSupernodeClient) Heartbeat(ctx context.Context) error { return m.hbErr }
func (m *mockSupernodeClient) Close()                              {}

func TestSupernodeObserverCrossSafetyViolation(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	st := &eth.SuperNodeSyncStatusResponse{
		Chains: map[eth.ChainID]eth.SyncStatus{
			execChain: {SafeL2: eth.L2BlockRef{Number: 250}},
		},
	}
	badJob := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1)},
		executingChain: execChain,
		executingBlock: eth.BlockID{Number: 200}, // <= cross-safe head 250 => violation
	}
	badJob.UpdateStatus(jobStatusInvalid)

	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{badJob.ID(): badJob})
	require.Len(t, mm.actualCrossSafetyViolations, 1)
}

func TestSupernodeObserverNoViolationAboveHead(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	st := &eth.SuperNodeSyncStatusResponse{
		Chains: map[eth.ChainID]eth.SyncStatus{
			execChain: {SafeL2: eth.L2BlockRef{Number: 100}},
		},
	}
	badJob := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1)},
		executingChain: execChain,
		executingBlock: eth.BlockID{Number: 200}, // > cross-safe head 100 => not yet promoted
	}
	badJob.UpdateStatus(jobStatusInvalid)

	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{badJob.ID(): badJob})
	require.Empty(t, mm.actualCrossSafetyViolations)
}

func TestSupernodeObserverHeartbeatDown(t *testing.T) {
	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{hbErr: errors.New("down")}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{})
	require.False(t, mm.lastSupernodeUp)
}

func TestSupernodeObserverRecordsHeads(t *testing.T) {
	execChain := eth.ChainIDFromUInt64(2)
	st := &eth.SuperNodeSyncStatusResponse{
		Chains: map[eth.ChainID]eth.SyncStatus{
			execChain: {SafeL2: eth.L2BlockRef{Number: 250}, FinalizedL2: eth.L2BlockRef{Number: 100}},
		},
	}
	mm := &mockMetrics{}
	obs := NewSupernodeObserver("http://sn", &mockSupernodeClient{status: st}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{})
	require.True(t, mm.lastSupernodeUp)
	// one cross_safe + one finalized head per chain
	require.Len(t, mm.actualSupernodeSafeHeads, 2)
}
