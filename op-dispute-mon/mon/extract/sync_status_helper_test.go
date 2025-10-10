package extract

import (
	"context"
	"errors"
	"testing"

	"github.com/ethereum-optimism/optimism/op-service/eth"
	"github.com/stretchr/testify/require"
)

// stub that does not implement syncStatusProvider
type noStatusClient struct{}

// stub that implements syncStatusProvider
type statusClient struct {
	status *eth.SyncStatus
	err    error
}

func (s *statusClient) SyncStatus(ctx context.Context) (*eth.SyncStatus, error) {
	return s.status, s.err
}

func TestIsNodeBehind_NonProvider(t *testing.T) {
	ctx := context.Background()
	behind, err := isNodeBehind(ctx, &noStatusClient{}, 100)
	require.NoError(t, err)
	require.False(t, behind)
}

func TestIsNodeBehind_ErrorFromProvider(t *testing.T) {
	ctx := context.Background()
	c := &statusClient{err: errors.New("boom")}
	behind, err := isNodeBehind(ctx, c, 100)
	require.Error(t, err)
	require.False(t, behind)
}

func TestIsNodeBehind_NilStatus(t *testing.T) {
	ctx := context.Background()
	c := &statusClient{status: nil, err: nil}
	behind, err := isNodeBehind(ctx, c, 100)
	require.NoError(t, err)
	require.False(t, behind)
}

func TestIsNodeBehind_BoundaryAndSlack(t *testing.T) {
	ctx := context.Background()
	st := &eth.SyncStatus{UnsafeL2: eth.L2BlockRef{Number: 100}}
	c := &statusClient{status: st}

	// l2 == unsafe => not behind
	behind, err := isNodeBehind(ctx, c, 100)
	require.NoError(t, err)
	require.False(t, behind)

	// l2 == unsafe + behindSlack => not behind
	behind, err = isNodeBehind(ctx, c, 100+lagTolerance)
	require.NoError(t, err)
	require.False(t, behind)

	// l2 == unsafe + behindSlack + 1 => behind
	behind, err = isNodeBehind(ctx, c, 100+lagTolerance+1)
	require.NoError(t, err)
	require.True(t, behind)
}
