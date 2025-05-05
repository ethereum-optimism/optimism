package noopbuilder

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/backend/work"
	"github.com/ethereum-optimism/optimism/op-test-sequencer/sequencer/seqtypes"
)

func TestNoopBuilder(t *testing.T) {
	id := seqtypes.BuilderID("foobar")
	x := NewBuilder(id, work.NewJobRegistry())

	job, err := x.NewJob(context.Background(), nil)
	require.NoError(t, err)
	require.Contains(t, job.String(), "noop-job-")
	require.NotEmpty(t, job.ID())
	require.NotNil(t, x.registry.GetJob(job.ID()))

	_, err = job.Seal(context.Background())
	require.ErrorIs(t, err, ErrNoBuild)

	require.NoError(t, job.Cancel(context.Background()))

	require.NoError(t, x.Close())
	require.Equal(t, "noop-builder-foobar", x.String())
	job.Close()
	require.Nil(t, x.registry.GetJob(job.ID()))
}
