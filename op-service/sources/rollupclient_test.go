package sources

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRollupClient_NullResult checks that a JSON null result, which leaves the decoded
// pointer nil, is returned as an error rather than as a nil value without error.
func TestRollupClient_NullResult(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		method string
		call   func(*RollupClient) (any, error)
	}{
		{"optimism_outputAtBlock", func(c *RollupClient) (any, error) { return c.OutputAtBlock(ctx, 1) }},
		{"optimism_safeHeadAtL1Block", func(c *RollupClient) (any, error) { return c.SafeHeadAtL1Block(ctx, 1) }},
		{"optimism_syncStatus", func(c *RollupClient) (any, error) { return c.SyncStatus(ctx) }},
		{"optimism_rollupConfig", func(c *RollupClient) (any, error) { return c.RollupConfig(ctx) }},
		{"optimism_dependencySet", func(c *RollupClient) (any, error) { return c.DependencySet(ctx) }},
	}
	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			rpc := new(mockRPC)
			defer rpc.AssertExpectations(t)
			rpc.On("CallContext", ctx, mock.Anything, tc.method, mock.Anything).Return([]error{nil})

			out, err := tc.call(NewRollupClient(rpc))
			require.ErrorContains(t, err, tc.method+" returned null")
			require.Nil(t, out)
		})
	}
}

func TestFollowClient_NullSyncStatus(t *testing.T) {
	ctx := context.Background()
	rpc := new(mockRPC)
	defer rpc.AssertExpectations(t)
	rpc.On("CallContext", ctx, mock.Anything, "optimism_syncStatus", mock.Anything).Return([]error{nil})

	client, err := NewFollowClient(rpc)
	require.NoError(t, err)
	_, err = client.GetFollowStatus(ctx)
	require.ErrorContains(t, err, "optimism_syncStatus returned null")
}
