package monitor

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-core/interop"
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

// rpcVerdictErr is a structured JSON-RPC error carrying an interop error code.
// It satisfies go-ethereum's rpc.Error.
type rpcVerdictErr struct {
	msg  string
	code int
}

func (e rpcVerdictErr) Error() string  { return e.msg }
func (e rpcVerdictErr) ErrorCode() int { return e.code }

func TestFilterObserverDivergence(t *testing.T) {
	// monitor says valid, filter rejects (structured RPC error) -> divergence recorded
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	mm := &mockMetrics{}
	rejection := rpcVerdictErr{msg: "conflicting data", code: interop.GetErrorCode(interop.ErrConflict)}
	obs := NewFilterObserver(&mockFilterChecker{checkErr: rejection}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Len(t, mm.actualFilterDivergences, 1)

	// A subsequent cycle must not re-count the same diverging job.
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Len(t, mm.actualFilterDivergences, 1)
}

func TestFilterObserverNoVerdictNotRecorded(t *testing.T) {
	// The filter is in failsafe and rejects every call. That is not a validity
	// verdict, so no divergence is recorded and the job is left unmarked for retry.
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	failsafe := rpcVerdictErr{msg: "failsafe is enabled", code: interop.GetErrorCode(interop.ErrFailsafeEnabled)}
	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{checkErr: failsafe}, mm, log.New())
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Empty(t, mm.actualFilterDivergences)
	require.False(t, job.FilterCheckedFor(jobStatusValid)) // not latched; retried next cycle
}

func TestFilterObserverRechecksOnStatusChange(t *testing.T) {
	// The filter consistently accepts the message (nil error). The monitor first
	// agrees (valid), then flips to invalid after a reorg. The observer must
	// re-query on the flip and record the now-divergent verdict, rather than
	// skipping the job because it was already checked once.
	job := &Job{
		initiating:     &messages.Identifier{ChainID: eth.ChainIDFromUInt64(1), BlockNumber: 10},
		executingChain: eth.ChainIDFromUInt64(2),
	}
	job.UpdateStatus(jobStatusValid)

	mm := &mockMetrics{}
	obs := NewFilterObserver(&mockFilterChecker{}, mm, log.New()) // filter always valid (nil)

	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Empty(t, mm.actualFilterDivergences) // monitor valid, filter valid -> agree

	// Monitor verdict flips to invalid (e.g. initiating-chain reorg).
	job.UpdateStatus(jobStatusInvalid)
	obs.Observe(context.Background(), map[JobID]*Job{job.ID(): job})
	require.Len(t, mm.actualFilterDivergences, 1) // monitor invalid, filter valid -> divergence
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
