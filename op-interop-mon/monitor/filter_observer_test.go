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

type mockFilterChecker struct {
	checkErr error
	failsafe bool
}

func (m *mockFilterChecker) CheckMessage(ctx context.Context, msg messages.Message, ec eth.ChainID, ts uint64) error {
	return m.checkErr
}
func (m *mockFilterChecker) GetFailsafeEnabled(ctx context.Context) (bool, error) {
	return m.failsafe, nil
}
func (m *mockFilterChecker) Close() {}

func TestFilterObserverDivergence(t *testing.T) {
	// monitor says valid, filter rejects -> divergence recorded
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{checkErr: errors.New("filter rejects")}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})

	require.Len(t, mm.actualFilterDivergences, 1)
}

func TestFilterObserverNoDivergenceWhenAgree(t *testing.T) {
	// monitor says valid, filter agrees (nil err) -> no divergence
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})

	require.Empty(t, mm.actualFilterDivergences)
}

func TestFilterObserverFailsafeGauge(t *testing.T) {
	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{failsafe: true}, mm, log.New())
	obs.PollFailsafe(context.Background())
	require.True(t, mm.lastFilterFailsafe)
}
