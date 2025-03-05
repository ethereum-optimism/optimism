package work

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

type mockJob struct {
	id seqtypes.BuildJobID
	BuildJob
}

func (m *mockJob) ID() seqtypes.BuildJobID {
	return m.id
}

var _ BuildJob = (*mockJob)(nil)

func TestRegistry(t *testing.T) {
	reg := NewJobRegistry()
	require.Equal(t, 0, reg.Len())
	reg.Clear()

	jobA := &mockJob{id: "A"}
	require.NoError(t, reg.RegisterJob(jobA))
	jobA2 := &mockJob{id: "A"}
	require.ErrorIs(t, reg.RegisterJob(jobA2), seqtypes.ErrConflictingJob)
	require.Equal(t, 1, reg.Len())

	reg.UnregisterJob(jobA.ID())
	require.Equal(t, 0, reg.Len())

	job1 := &mockJob{id: "1"}
	job2 := &mockJob{id: "2"}
	job3 := &mockJob{id: "3"}
	require.NoError(t, reg.RegisterJob(job1))
	require.NoError(t, reg.RegisterJob(job2))
	require.NoError(t, reg.RegisterJob(job3))
	require.Equal(t, job2, reg.GetJob("2"))

	reg.Clear()
	require.Equal(t, 0, reg.Len())
}
