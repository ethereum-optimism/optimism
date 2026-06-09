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

// rpcVerdictErr is a structured JSON-RPC error representing a filter rejection
// (as opposed to a transport error). It satisfies go-ethereum's rpc.Error.
type rpcVerdictErr struct{ msg string }

func (e rpcVerdictErr) Error() string  { return e.msg }
func (e rpcVerdictErr) ErrorCode() int { return -32000 }

func TestFilterObserverDivergence(t *testing.T) {
	// monitor says valid, filter rejects (structured RPC error) -> divergence recorded
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{checkErr: rpcVerdictErr{msg: "filter rejects"}}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Len(t, mm.actualFilterDivergences, 1)

	// A subsequent cycle must not re-count the same diverging job.
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Len(t, mm.actualFilterDivergences, 1)
}

func TestFilterObserverTransportErrorNoDivergence(t *testing.T) {
	// monitor says valid, filter call fails with a transport error (not a verdict)
	// -> no divergence recorded.
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{checkErr: errors.New("connection refused")}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Empty(t, mm.actualFilterDivergences)
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
